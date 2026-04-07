package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/ham-agents/ham-agents/go/internal/core"
)

type EventStore interface {
	Append(ctx context.Context, event core.Event) error
	Load(ctx context.Context) ([]core.Event, error)
}

// maxEventEntries is the maximum number of events retained in the log.
// When exceeded, the oldest entries are pruned during Append.
const maxEventEntries = 10000

const (
	artifactInlineMaxBytes = 4 * 1024        // 4 KB  — keep inline
	artifactFileMaxBytes   = 1 * 1024 * 1024 // 1 MB  — truncate above this
)

type FileEventStore struct {
	path          string
	mu            sync.Mutex
	appendCount   int
	artifactStore ArtifactStore
}

func NewFileEventStore(path string) *FileEventStore {
	return &FileEventStore{path: path}
}

// WithArtifactStore attaches an ArtifactStore so that Append offloads
// ArtifactData larger than artifactInlineMaxBytes to files.
// Safe to call before first Append; returns the receiver for chaining.
func (s *FileEventStore) WithArtifactStore(as ArtifactStore) *FileEventStore {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.artifactStore = as
	return s
}

func DefaultEventLogPath() (string, error) {
	if root := os.Getenv("HAM_AGENTS_HOME"); root != "" {
		return filepath.Join(root, "events.jsonl"), nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}

	return filepath.Join(homeDir, "Library", "Application Support", "ham-agents", "events.jsonl"), nil
}

func (s *FileEventStore) Append(ctx context.Context, event core.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create event log directory: %w", err)
	}

	// Offload large ArtifactData to the artifact store when one is configured.
	if s.artifactStore != nil && len(event.ArtifactData) > artifactInlineMaxBytes {
		data := []byte(event.ArtifactData)
		if len(data) > artifactFileMaxBytes {
			data = data[:artifactFileMaxBytes]
			// Mark truncation by appending ":truncated" to ArtifactType.
			event.ArtifactType = event.ArtifactType + ":truncated"
		}
		ref, saveErr := s.artifactStore.Save(event.AgentID, event.ID, data)
		if saveErr != nil {
			// Artifact offload is best-effort; keep ArtifactData inline on failure.
			log.Printf("store: artifact offload failed for event %s: %v", event.ID, saveErr)
		} else {
			event.ArtifactRef = ref
			event.ArtifactData = ""
		}
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	file, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open event log: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(append(payload, '\n')); err != nil {
		return fmt.Errorf("append event: %w", err)
	}

	s.appendCount++
	if s.appendCount%1000 == 0 {
		s.truncateLocked(ctx)
	}

	return nil
}

// truncateLocked prunes the event log to the most recent maxEventEntries.
// Must be called with s.mu held.
func (s *FileEventStore) truncateLocked(ctx context.Context) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return
	}

	lines := bytesSplitLines(data)
	if len(lines) <= maxEventEntries {
		return
	}

	// Keep only the most recent entries.
	kept := lines[len(lines)-maxEventEntries:]
	var buf []byte
	for _, line := range kept {
		if len(line) == 0 {
			continue
		}
		buf = append(buf, line...)
		buf = append(buf, '\n')
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, buf, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmpPath, s.path)
}

func (s *FileEventStore) Load(ctx context.Context) ([]core.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	payload, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []core.Event{}, nil
		}
		return nil, fmt.Errorf("read event log: %w", err)
	}

	if len(payload) == 0 {
		return []core.Event{}, nil
	}

	lines := bytesSplitLines(payload)
	events := make([]core.Event, 0, len(lines))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		var event core.Event
		if err := json.Unmarshal(line, &event); err != nil {
			return nil, fmt.Errorf("decode event: %w", err)
		}
		events = append(events, event)
	}

	return events, nil
}

func bytesSplitLines(payload []byte) [][]byte {
	lines := make([][]byte, 0)
	start := 0
	for index, value := range payload {
		if value != '\n' {
			continue
		}
		lines = append(lines, payload[start:index])
		start = index + 1
	}
	if start < len(payload) {
		lines = append(lines, payload[start:])
	}
	return lines
}
