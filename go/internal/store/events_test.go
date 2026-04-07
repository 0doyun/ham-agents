package store_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ham-agents/ham-agents/go/internal/core"
	"github.com/ham-agents/ham-agents/go/internal/store"
)

// TestFileEventStore_Append_TruncatesAtMaxEvents verifies that the event log
// is pruned to maxEventEntries (10000) when the append count crosses a multiple
// of 1000. We pre-write 10001 events directly into the JSONL file (simulating
// an already-large log), then drive 1000 Append calls to trigger truncateLocked,
// and assert that the store retains exactly 10000 events (the newest ones) and
// discards the oldest.
//
// Rationale: maxEventEntries = 10000, and truncateLocked fires every 1000
// Append calls. Pre-writing the file lets us stay well above the limit without
// actually calling Append 10001 times.
func TestFileEventStore_Append_TruncatesAtMaxEvents(t *testing.T) {
	t.Parallel()

	const maxEvents = 10000 // must match store.maxEventEntries

	ctx := context.Background()
	dir := t.TempDir()
	eventPath := filepath.Join(dir, "events.jsonl")

	// Pre-write maxEvents+1 events directly to the file so the log is already
	// over the limit before the store instance is created.
	prewriteCount := maxEvents + 1
	{
		f, err := os.Create(eventPath)
		if err != nil {
			t.Fatalf("create event file: %v", err)
		}
		enc := json.NewEncoder(f)
		for i := 0; i < prewriteCount; i++ {
			ev := core.Event{
				ID:      fmt.Sprintf("pre-%05d", i),
				AgentID: fmt.Sprintf("agent-%05d", i),
				Type:    core.EventTypeAgentRegistered,
				Summary: fmt.Sprintf("pre-written event %d", i),
			}
			if err := enc.Encode(ev); err != nil {
				f.Close()
				t.Fatalf("encode pre-written event %d: %v", i, err)
			}
		}
		f.Close()
	}

	// Create a fresh store instance (appendCount starts at 0).
	eventStore := store.NewFileEventStore(eventPath)

	// Append 1000 events to trigger the first truncateLocked call (on the
	// 1000th Append). These events use sequential IDs so we can verify later
	// which ones were retained.
	for i := 0; i < 1000; i++ {
		ev := core.Event{
			ID:      fmt.Sprintf("new-%05d", i),
			AgentID: fmt.Sprintf("agent-new-%05d", i),
			Type:    core.EventTypeAgentRegistered,
			Summary: fmt.Sprintf("new event %d", i),
		}
		if err := eventStore.Append(ctx, ev); err != nil {
			t.Fatalf("append event %d: %v", i, err)
		}
	}

	// Load all events and verify the truncation invariants.
	events, err := eventStore.Load(ctx)
	if err != nil {
		t.Fatalf("load events after truncation: %v", err)
	}

	// After truncation the store must hold exactly maxEvents entries.
	if len(events) != maxEvents {
		t.Errorf("expected %d events after truncation, got %d", maxEvents, len(events))
	}

	// The very first pre-written event (pre-00000) must be gone — it was the
	// oldest and should have been pruned.
	for _, ev := range events {
		if ev.ID == "pre-00000" {
			t.Errorf("oldest event pre-00000 should have been pruned but was retained")
			break
		}
	}

	// The very last new event must still be present (newest event preserved).
	lastNewID := fmt.Sprintf("new-%05d", 999)
	found := false
	for _, ev := range events {
		if ev.ID == lastNewID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("newest event %q was not found after truncation", lastNewID)
	}

	// Order must be maintained: IDs should appear in monotonically increasing
	// (pre-written then new) order within the retained slice. Spot-check that
	// the last element is the last new event.
	if len(events) > 0 {
		last := events[len(events)-1]
		if last.ID != lastNewID {
			t.Errorf("last retained event should be %q, got %q", lastNewID, last.ID)
		}
	}
}
