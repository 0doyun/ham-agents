package runtime

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/core"
	"github.com/ham-agents/ham-agents/go/internal/store"
)

// DefaultCostPollInterval is how often CostTracker re-scans the transcript
// directory for new usage records when no override is supplied. ADR-3 calls
// for 5 second polling.
const DefaultCostPollInterval = 5 * time.Second

// DefaultClaudeTranscriptDir returns the on-disk location where Claude Code
// drops session JSONL files on macOS. Linux/Windows are out of scope for
// P1-4 v1 (see ADR-3 Risks).
func DefaultClaudeTranscriptDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".claude", "projects"), nil
}

// CostTracker polls a Claude Code transcript directory and ingests new
// assistant-message usage records into a CostStore. Records are deduped on
// RequestID/MessageID and tagged with the matching agent's ID when the
// session can be resolved through the Registry; orphaned records keep an
// empty AgentID per ADR-3.
type CostTracker struct {
	transcriptDir   string
	store           store.CostStore
	registry        *Registry
	pollInterval    time.Duration
	lastSeenOffsets sync.Map // key: file path -> value: int64
	seenIDs         sync.Map // key: dedupKey -> value: struct{}
	clock           func() time.Time
}

// NewCostTracker constructs a tracker. A zero pollInterval is replaced with
// DefaultCostPollInterval. The Registry argument may be nil; in that case
// every record is treated as orphaned.
func NewCostTracker(transcriptDir string, costStore store.CostStore, registry *Registry, pollInterval time.Duration) *CostTracker {
	if pollInterval <= 0 {
		pollInterval = DefaultCostPollInterval
	}
	return &CostTracker{
		transcriptDir: transcriptDir,
		store:         costStore,
		registry:      registry,
		pollInterval:  pollInterval,
		clock:         time.Now,
	}
}

// Start launches the polling goroutine. The tracker stops when ctx is
// cancelled. Start returns immediately.
func (t *CostTracker) Start(ctx context.Context) {
	if t == nil || t.store == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(t.pollInterval)
		defer ticker.Stop()
		// Run an initial tick so callers don't have to wait pollInterval
		// for the first scan.
		if err := t.Tick(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("cost_tracker: initial tick: %v", err)
		}
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := t.Tick(ctx); err != nil && !errors.Is(err, context.Canceled) {
					log.Printf("cost_tracker: tick: %v", err)
				}
			}
		}
	}()
}

// Tick performs one scan-and-ingest cycle. Exposed so tests can drive the
// tracker deterministically without leaning on wall clock timing.
func (t *CostTracker) Tick(ctx context.Context) error {
	if t == nil || t.store == nil {
		return nil
	}
	if strings.TrimSpace(t.transcriptDir) == "" {
		return nil
	}
	files, err := t.discoverTranscriptFiles()
	if err != nil {
		return err
	}
	for _, file := range files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := t.ingestFile(ctx, file); err != nil {
			log.Printf("cost_tracker: ingest %s: %v", file, err)
		}
	}
	return nil
}

// discoverTranscriptFiles walks the transcriptDir one level deep, matching
// the on-disk layout ~/.claude/projects/<encoded>/<session>.jsonl described
// in ADR-3.
func (t *CostTracker) discoverTranscriptFiles() ([]string, error) {
	entries, err := os.ReadDir(t.transcriptDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var files []string
	for _, entry := range entries {
		// Each project lives in its own subdirectory; transcripts can also
		// sit at the top level for legacy installs.
		if entry.IsDir() {
			projectDir := filepath.Join(t.transcriptDir, entry.Name())
			children, err := os.ReadDir(projectDir)
			if err != nil {
				continue
			}
			for _, child := range children {
				if child.IsDir() || !strings.HasSuffix(child.Name(), ".jsonl") {
					continue
				}
				files = append(files, filepath.Join(projectDir, child.Name()))
			}
			continue
		}
		if strings.HasSuffix(entry.Name(), ".jsonl") {
			files = append(files, filepath.Join(t.transcriptDir, entry.Name()))
		}
	}
	return files, nil
}

func (t *CostTracker) ingestFile(ctx context.Context, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	previousAny, _ := t.lastSeenOffsets.Load(path)
	previousSize, _ := previousAny.(int64)
	if info.Size() == previousSize {
		return nil
	}
	// We always reparse the whole file because Claude Code rotates
	// transcript writes by appending JSON objects, and a partial line at
	// the end of a previous read would otherwise be lost. Dedup on
	// RequestID prevents double counting.
	records, err := store.ParseTranscriptFile(path)
	if err != nil {
		return err
	}
	for _, record := range records {
		key := record.DedupKey()
		if key == "" {
			continue
		}
		if _, dup := t.seenIDs.LoadOrStore(key, struct{}{}); dup {
			continue
		}
		t.assignAgent(ctx, &record)
		if err := t.store.Append(ctx, record); err != nil {
			return err
		}
	}
	t.lastSeenOffsets.Store(path, info.Size())
	return nil
}

func (t *CostTracker) assignAgent(ctx context.Context, record *core.CostRecord) {
	if t.registry == nil || record.SessionID == "" {
		return
	}
	agent, err := t.registry.FindAgentBySessionID(ctx, record.SessionID)
	if err != nil {
		return
	}
	record.AgentID = agent.ID
}
