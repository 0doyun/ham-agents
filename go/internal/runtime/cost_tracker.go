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

// DefaultCostPollInterval was the original 5-second polling interval used
// before P1-4.1 converted CostTracker to on-demand mode. Kept only for
// backward compatibility with callers that reference it; new code should
// call Tick directly from the IPC handler instead of starting a goroutine.
//
// Deprecated: use Tick() on-demand via the cost.summary IPC handler.
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

// CostTracker scans a Claude Code transcript directory on-demand and ingests
// new assistant-message usage records into a CostStore. Records are deduped
// against the existing store contents so duplicates are never persisted.
// The tracker is invoked via Tick() from the cost.summary IPC handler; there
// is no background goroutine (converted from 5-second polling in P1-4.1).
type CostTracker struct {
	transcriptDir   string
	store           store.CostStore
	registry        *Registry
	lastSeenOffsets sync.Map // key: file path -> value: int64
	clock           func() time.Time
}

// NewCostTracker constructs a tracker. The pollInterval parameter is ignored
// (kept for backward-compatible call sites); all scanning happens on-demand
// via Tick(). The Registry argument may be nil; in that case every record is
// treated as orphaned.
func NewCostTracker(transcriptDir string, costStore store.CostStore, registry *Registry, pollInterval time.Duration) *CostTracker {
	return &CostTracker{
		transcriptDir: transcriptDir,
		store:         costStore,
		registry:      registry,
		clock:         time.Now,
	}
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
	seenIDs := t.buildSeenIDs(ctx)
	for _, file := range files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := t.ingestFile(ctx, file, seenIDs); err != nil {
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

func (t *CostTracker) ingestFile(ctx context.Context, path string, seenIDs map[string]struct{}) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	previousAny, _ := t.lastSeenOffsets.Load(path)
	previousSize, _ := previousAny.(int64)
	if info.Size() == previousSize {
		return nil
	}
	records, err := store.ParseTranscriptFile(path)
	if err != nil {
		return err
	}
	for _, record := range records {
		key := record.DedupKey()
		if key == "" {
			continue
		}
		if _, dup := seenIDs[key]; dup {
			continue
		}
		seenIDs[key] = struct{}{}
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

// buildSeenIDs loads all existing records from the store and returns a
// dedup set so the current Tick does not re-persist them. The set is
// ephemeral — it lives only for the duration of the Tick call, which
// prevents the unbounded growth that plagued the old sync.Map approach.
func (t *CostTracker) buildSeenIDs(ctx context.Context) map[string]struct{} {
	seen := make(map[string]struct{})
	records, err := t.store.Load(ctx, store.CostFilter{})
	if err != nil {
		log.Printf("cost_tracker: buildSeenIDs load failed: %v", err)
		return seen
	}
	for _, record := range records {
		if key := record.DedupKey(); key != "" {
			seen[key] = struct{}{}
		}
	}
	return seen
}
