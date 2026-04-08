package store_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/core"
	"github.com/ham-agents/ham-agents/go/internal/store"
)

func newTempCostStore(t *testing.T) (*store.FileCostStore, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "cost.jsonl")
	return store.NewFileCostStore(path), path
}

func mustAppend(t *testing.T, s store.CostStore, record core.CostRecord) {
	t.Helper()
	if err := s.Append(context.Background(), record); err != nil {
		t.Fatalf("append: %v", err)
	}
}

func TestFileCostStore_AppendThenLoad(t *testing.T) {
	t.Parallel()

	s, _ := newTempCostStore(t)
	now := time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)
	mustAppend(t, s, core.CostRecord{
		AgentID:      "agent-1",
		Model:        "claude-opus-4-6",
		InputTokens:  100,
		OutputTokens: 200,
		EstimatedUSD: 1.23,
		RecordedAt:   now,
		RequestID:    "req_1",
	})
	mustAppend(t, s, core.CostRecord{
		AgentID:      "agent-2",
		Model:        "claude-sonnet-4-6",
		InputTokens:  50,
		OutputTokens: 60,
		EstimatedUSD: 0.45,
		RecordedAt:   now.Add(time.Hour),
		RequestID:    "req_2",
	})

	records, err := s.Load(context.Background(), store.CostFilter{})
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].RequestID != "req_1" || records[1].RequestID != "req_2" {
		t.Fatalf("order: %v", records)
	}
}

func TestFileCostStore_LoadFilterByAgent(t *testing.T) {
	t.Parallel()

	s, _ := newTempCostStore(t)
	now := time.Now().UTC()
	mustAppend(t, s, core.CostRecord{AgentID: "a1", Model: "claude-opus-4-6", RecordedAt: now, RequestID: "r1"})
	mustAppend(t, s, core.CostRecord{AgentID: "a2", Model: "claude-opus-4-6", RecordedAt: now, RequestID: "r2"})
	mustAppend(t, s, core.CostRecord{AgentID: "a1", Model: "claude-sonnet-4-6", RecordedAt: now, RequestID: "r3"})

	records, err := s.Load(context.Background(), store.CostFilter{AgentID: "a1"})
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 a1 records, got %d", len(records))
	}
	for _, r := range records {
		if r.AgentID != "a1" {
			t.Fatalf("unexpected agent in filter: %q", r.AgentID)
		}
	}
}

func TestFileCostStore_LoadFilterBySinceUntil(t *testing.T) {
	t.Parallel()

	s, _ := newTempCostStore(t)
	base := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		mustAppend(t, s, core.CostRecord{
			Model:      "claude-opus-4-6",
			RecordedAt: base.Add(time.Duration(i) * 24 * time.Hour),
			RequestID:  "r" + string(rune('0'+i)),
		})
	}

	since := base.Add(2 * 24 * time.Hour)
	until := base.Add(4 * 24 * time.Hour)
	records, err := s.Load(context.Background(), store.CostFilter{Since: since, Until: until})
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(records) != 2 { // days 2 and 3 (4 is excluded by Until being exclusive)
		t.Fatalf("expected 2 records in window, got %d (%v)", len(records), records)
	}
}

func TestFileCostStore_LoadFilterByModel(t *testing.T) {
	t.Parallel()

	s, _ := newTempCostStore(t)
	now := time.Now().UTC()
	mustAppend(t, s, core.CostRecord{Model: "claude-opus-4-6", RecordedAt: now, RequestID: "o1"})
	mustAppend(t, s, core.CostRecord{Model: "claude-haiku-4-5", RecordedAt: now, RequestID: "h1"})

	records, err := s.Load(context.Background(), store.CostFilter{Model: "claude-haiku-4-5"})
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(records) != 1 || records[0].Model != "claude-haiku-4-5" {
		t.Fatalf("filter by model failed: %v", records)
	}
}

func TestFileCostStore_Prune(t *testing.T) {
	t.Parallel()

	s, _ := newTempCostStore(t)
	base := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		mustAppend(t, s, core.CostRecord{
			Model:      "claude-opus-4-6",
			RecordedAt: base.Add(time.Duration(i) * 24 * time.Hour),
			RequestID:  "r" + string(rune('0'+i)),
		})
	}

	cutoff := base.Add(3 * 24 * time.Hour)
	if err := s.Prune(context.Background(), cutoff); err != nil {
		t.Fatalf("prune: %v", err)
	}

	records, err := s.Load(context.Background(), store.CostFilter{})
	if err != nil {
		t.Fatalf("load after prune: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records after prune, got %d", len(records))
	}
	for _, r := range records {
		if r.RecordedAt.Before(cutoff) {
			t.Fatalf("found record older than cutoff: %v", r.RecordedAt)
		}
	}
}

func TestFileCostStore_Prune_ZeroCutoffNoOp(t *testing.T) {
	t.Parallel()

	s, _ := newTempCostStore(t)
	mustAppend(t, s, core.CostRecord{Model: "claude-opus-4-6", RecordedAt: time.Now().UTC(), RequestID: "r1"})

	if err := s.Prune(context.Background(), time.Time{}); err != nil {
		t.Fatalf("prune with zero cutoff should be no-op: %v", err)
	}
	records, _ := s.Load(context.Background(), store.CostFilter{})
	if len(records) != 1 {
		t.Fatalf("expected record to remain, got %d", len(records))
	}
}

func TestFileCostStore_LoadEmptyFile(t *testing.T) {
	t.Parallel()

	s, _ := newTempCostStore(t)
	records, err := s.Load(context.Background(), store.CostFilter{})
	if err != nil {
		t.Fatalf("load empty: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected 0 records, got %d", len(records))
	}
}

func TestDefaultCostLogPath_HamAgentsHome(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HAM_AGENTS_HOME", dir)

	got, err := store.DefaultCostLogPath()
	if err != nil {
		t.Fatalf("default path: %v", err)
	}
	want := filepath.Join(dir, "cost.jsonl")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestDefaultCostLogPath_FallsBackToHome(t *testing.T) {
	t.Setenv("HAM_AGENTS_HOME", "")
	got, err := store.DefaultCostLogPath()
	if err != nil {
		t.Fatalf("default path: %v", err)
	}
	homeDir, _ := os.UserHomeDir()
	want := filepath.Join(homeDir, "Library", "Application Support", "ham-agents", "cost.jsonl")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
