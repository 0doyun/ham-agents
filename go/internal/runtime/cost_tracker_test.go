package runtime_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/core"
	"github.com/ham-agents/ham-agents/go/internal/runtime"
	"github.com/ham-agents/ham-agents/go/internal/store"
)

const trackerSampleLine = `{"type":"assistant","uuid":"u1","sessionId":"%s","timestamp":"2026-04-08T12:00:00.000Z","requestId":"%s","cwd":"/tmp/proj","isSidechain":false,"message":{"id":"%s","role":"assistant","model":"claude-opus-4-6","usage":{"input_tokens":1,"output_tokens":2,"service_tier":"standard"}}}`

func writeTranscript(t *testing.T, dir, project, session string, lines []string) string {
	t.Helper()
	projectDir := filepath.Join(dir, project)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	path := filepath.Join(projectDir, session+".jsonl")
	contents := ""
	for _, line := range lines {
		contents += line + "\n"
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}
	return path
}

func sampleLine(session, requestID, messageID string) string {
	return formatLine(trackerSampleLine, session, requestID, messageID)
}

func formatLine(format string, args ...string) string {
	out := format
	for _, a := range args {
		idx := indexOfPercent(out)
		if idx < 0 {
			break
		}
		out = out[:idx] + a + out[idx+2:]
	}
	return out
}

func indexOfPercent(s string) int {
	for i := 0; i < len(s)-1; i++ {
		if s[i] == '%' && s[i+1] == 's' {
			return i
		}
	}
	return -1
}

func newTrackerHarness(t *testing.T) (*runtime.CostTracker, *store.FileCostStore, string) {
	t.Helper()
	transcriptDir := t.TempDir()
	costPath := filepath.Join(t.TempDir(), "cost.jsonl")
	costStore := store.NewFileCostStore(costPath)
	tracker := runtime.NewCostTracker(transcriptDir, costStore, nil, 50*time.Millisecond)
	return tracker, costStore, transcriptDir
}

func TestCostTracker_TickIngestsNewRecords(t *testing.T) {
	t.Parallel()

	tracker, costStore, dir := newTrackerHarness(t)
	writeTranscript(t, dir, "-tmp-proj", "sess1", []string{
		sampleLine("sess1", "req_1", "msg_1"),
		sampleLine("sess1", "req_2", "msg_2"),
	})

	if err := tracker.Tick(context.Background()); err != nil {
		t.Fatalf("tick: %v", err)
	}

	records, err := costStore.Load(context.Background(), store.CostFilter{})
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].EstimatedUSD <= 0 {
		t.Fatalf("expected positive USD on opus record, got %.6f", records[0].EstimatedUSD)
	}
}

func TestCostTracker_DeduplicatesByRequestID(t *testing.T) {
	t.Parallel()

	tracker, costStore, dir := newTrackerHarness(t)
	path := writeTranscript(t, dir, "-tmp-dedup", "sess1", []string{
		sampleLine("sess1", "req_dup", "msg_a"),
	})

	if err := tracker.Tick(context.Background()); err != nil {
		t.Fatalf("first tick: %v", err)
	}
	// Append another record so file size grows; the first record reappears
	// because we always reparse the whole file. Dedup must drop it.
	contents, _ := os.ReadFile(path)
	contents = append(contents, []byte(sampleLine("sess1", "req_new", "msg_b")+"\n")...)
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	if err := tracker.Tick(context.Background()); err != nil {
		t.Fatalf("second tick: %v", err)
	}

	records, _ := costStore.Load(context.Background(), store.CostFilter{})
	if len(records) != 2 {
		t.Fatalf("expected 2 unique records, got %d (%+v)", len(records), records)
	}
}

func TestCostTracker_OrphanWhenNoSessionMatch(t *testing.T) {
	t.Parallel()

	tracker, costStore, dir := newTrackerHarness(t)
	writeTranscript(t, dir, "-orphan", "sess-orphan", []string{
		sampleLine("sess-orphan", "req_orphan", "msg_orphan"),
	})
	if err := tracker.Tick(context.Background()); err != nil {
		t.Fatalf("tick: %v", err)
	}
	records, _ := costStore.Load(context.Background(), store.CostFilter{})
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].AgentID != "" {
		t.Fatalf("orphan record should have empty AgentID, got %q", records[0].AgentID)
	}
}

func TestCostTracker_AssignsAgentIDFromSession(t *testing.T) {
	t.Parallel()

	transcriptDir := t.TempDir()
	costPath := filepath.Join(t.TempDir(), "cost.jsonl")
	costStore := store.NewFileCostStore(costPath)

	statePath := filepath.Join(t.TempDir(), "state.json")
	registry := runtime.NewRegistry(store.NewFileAgentStore(statePath), nil)
	agent, err := registry.RegisterManaged(context.Background(), runtime.RegisterManagedInput{
		Provider:    "claude",
		DisplayName: "test-agent",
		ProjectPath: "/tmp/proj",
		SessionRef:  "sess-known",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := registry.RecordHookSessionSeen(context.Background(), agent.ID, "sess-known"); err != nil {
		t.Fatalf("record session: %v", err)
	}

	tracker := runtime.NewCostTracker(transcriptDir, costStore, registry, 50*time.Millisecond)
	writeTranscript(t, transcriptDir, "-known", "sess-known", []string{
		sampleLine("sess-known", "req_known", "msg_known"),
	})
	if err := tracker.Tick(context.Background()); err != nil {
		t.Fatalf("tick: %v", err)
	}

	records, _ := costStore.Load(context.Background(), store.CostFilter{})
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].AgentID != agent.ID {
		t.Fatalf("expected AgentID %q, got %q", agent.ID, records[0].AgentID)
	}
}

func TestCostTracker_WarmsUpFromExistingStore(t *testing.T) {
	t.Parallel()

	// Simulate a daemon restart: pre-populate the cost store with a record,
	// then start a fresh tracker against a transcript file containing the
	// same record. The tracker must skip the duplicate.
	transcriptDir := t.TempDir()
	costPath := filepath.Join(t.TempDir(), "cost.jsonl")
	costStore := store.NewFileCostStore(costPath)

	preexisting := core.CostRecord{
		Model:        "claude-opus-4-6",
		InputTokens:  1,
		OutputTokens: 2,
		EstimatedUSD: 0.001,
		RecordedAt:   time.Now().UTC(),
		RequestID:    "req_warm",
		Source:       "assistant",
	}
	if err := costStore.Append(context.Background(), preexisting); err != nil {
		t.Fatalf("seed: %v", err)
	}

	writeTranscript(t, transcriptDir, "-warm", "sess-warm", []string{
		sampleLine("sess-warm", "req_warm", "msg_warm"),
	})

	tracker := runtime.NewCostTracker(transcriptDir, costStore, nil, 50*time.Millisecond)
	if err := tracker.Tick(context.Background()); err != nil {
		t.Fatalf("tick: %v", err)
	}
	records, _ := costStore.Load(context.Background(), store.CostFilter{})
	if len(records) != 1 {
		t.Fatalf("expected dedup against pre-existing store, got %d records", len(records))
	}
}

func TestCostTracker_SeenIDsNotUnbounded(t *testing.T) {
	t.Parallel()

	// Verify that seenIDs are rebuilt from the store each Tick (ephemeral)
	// rather than accumulating across calls. Two Ticks with the same
	// transcript should not grow any persistent map.
	tracker, costStore, dir := newTrackerHarness(t)
	writeTranscript(t, dir, "-bounded", "sess1", []string{
		sampleLine("sess1", "req_b1", "msg_b1"),
	})

	if err := tracker.Tick(context.Background()); err != nil {
		t.Fatalf("tick 1: %v", err)
	}
	records, _ := costStore.Load(context.Background(), store.CostFilter{})
	if len(records) != 1 {
		t.Fatalf("after tick 1: expected 1 record, got %d", len(records))
	}
	// Second tick with same content: should not duplicate.
	if err := tracker.Tick(context.Background()); err != nil {
		t.Fatalf("tick 2: %v", err)
	}
	records, _ = costStore.Load(context.Background(), store.CostFilter{})
	if len(records) != 1 {
		t.Fatalf("after tick 2: expected 1 record (dedup), got %d", len(records))
	}
}
