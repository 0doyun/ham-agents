package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/core"
)

// CostFilter narrows the result set returned by CostStore.Load. Zero values
// mean "no filter" — an empty AgentID matches every record, a zero Since
// matches every timestamp from the beginning of time, and a zero Until
// matches every timestamp up to now.
type CostFilter struct {
	AgentID string
	Since   time.Time
	Until   time.Time
	Model   string
}

// CostStore persists CostRecords to durable storage. The interface mirrors
// EventStore so callers can swap implementations in tests.
type CostStore interface {
	Append(ctx context.Context, record core.CostRecord) error
	Load(ctx context.Context, filter CostFilter) ([]core.CostRecord, error)
	Prune(ctx context.Context, before time.Time) error
}

// FileCostStore writes CostRecords as JSONL to a single file. Reads scan the
// file linearly and apply the CostFilter in memory; this is acceptable for
// the v1 volume (a few hundred records per day) and avoids dragging in a
// database dependency. See ADR-3 for the volume estimate.
type FileCostStore struct {
	path string
	mu   sync.Mutex
}

// NewFileCostStore returns a CostStore backed by the JSONL file at path.
// The file is created lazily on first Append.
func NewFileCostStore(path string) *FileCostStore {
	return &FileCostStore{path: path}
}

// DefaultCostLogPath mirrors DefaultEventLogPath: $HAM_AGENTS_HOME/cost.jsonl
// when set, otherwise the macOS Application Support directory under
// ~/Library/Application Support/ham-agents/cost.jsonl.
func DefaultCostLogPath() (string, error) {
	if root := os.Getenv("HAM_AGENTS_HOME"); root != "" {
		return filepath.Join(root, "cost.jsonl"), nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}
	return filepath.Join(homeDir, "Library", "Application Support", "ham-agents", "cost.jsonl"), nil
}

func (s *FileCostStore) Append(ctx context.Context, record core.CostRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create cost log directory: %w", err)
	}

	payload, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal cost record: %w", err)
	}

	file, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open cost log: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(append(payload, '\n')); err != nil {
		return fmt.Errorf("append cost record: %w", err)
	}
	return nil
}

func (s *FileCostStore) Load(ctx context.Context, filter CostFilter) ([]core.CostRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	records, err := s.readAllLocked()
	if err != nil {
		return nil, err
	}

	if filter.AgentID == "" && filter.Model == "" && filter.Since.IsZero() && filter.Until.IsZero() {
		return records, nil
	}

	filtered := make([]core.CostRecord, 0, len(records))
	for _, record := range records {
		if filter.AgentID != "" && record.AgentID != filter.AgentID {
			continue
		}
		if filter.Model != "" && record.Model != filter.Model {
			continue
		}
		if !filter.Since.IsZero() && record.RecordedAt.Before(filter.Since) {
			continue
		}
		if !filter.Until.IsZero() && !record.RecordedAt.Before(filter.Until) {
			continue
		}
		filtered = append(filtered, record)
	}
	return filtered, nil
}

// Prune drops every record whose RecordedAt is strictly before the cutoff,
// then atomically rewrites the JSONL file. A zero cutoff is a no-op so
// callers can guard against accidentally wiping the log.
func (s *FileCostStore) Prune(ctx context.Context, before time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if before.IsZero() {
		return nil
	}

	records, err := s.readAllLocked()
	if err != nil {
		return err
	}

	kept := make([]core.CostRecord, 0, len(records))
	for _, record := range records {
		if record.RecordedAt.Before(before) {
			continue
		}
		kept = append(kept, record)
	}
	if len(kept) == len(records) {
		return nil
	}

	var buf []byte
	for _, record := range kept {
		payload, err := json.Marshal(record)
		if err != nil {
			return fmt.Errorf("marshal cost record during prune: %w", err)
		}
		buf = append(buf, payload...)
		buf = append(buf, '\n')
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, buf, 0o644); err != nil {
		return fmt.Errorf("write pruned cost log: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("rename pruned cost log: %w", err)
	}
	return nil
}

// readAllLocked reads every record from disk. Caller must hold s.mu.
func (s *FileCostStore) readAllLocked() ([]core.CostRecord, error) {
	payload, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []core.CostRecord{}, nil
		}
		return nil, fmt.Errorf("read cost log: %w", err)
	}
	if len(payload) == 0 {
		return []core.CostRecord{}, nil
	}
	lines := bytesSplitLines(payload)
	records := make([]core.CostRecord, 0, len(lines))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var record core.CostRecord
		if err := json.Unmarshal(line, &record); err != nil {
			return nil, fmt.Errorf("decode cost record: %w", err)
		}
		records = append(records, record)
	}
	return records, nil
}
