package runtime

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/core"
)

// makeEvent constructs a hook-origin event for inbox tests. The second
// parameter is the canonical hook command name (e.g. "hook.notification").
// An empty hookOrigin produces a non-hook event that the inbox should ignore.
func makeEvent(id string, hookOrigin string, agentID string) core.Event {
	return core.Event{
		ID:         id,
		AgentID:    agentID,
		HookOrigin: hookOrigin,
		OccurredAt: time.Now().UTC(),
	}
}

func makeEventWithTool(id string, hookOrigin string, agentID, toolName string) core.Event {
	e := makeEvent(id, hookOrigin, agentID)
	e.ToolName = toolName
	return e
}

func makeEventWithTask(id string, hookOrigin string, agentID, taskName string) core.Event {
	e := makeEvent(id, hookOrigin, agentID)
	e.TaskName = taskName
	return e
}

func newTestInboxManager(t *testing.T) *InboxManager {
	t.Helper()
	path := filepath.Join(t.TempDir(), "inbox.json")
	m, err := NewInboxManager(path)
	if err != nil {
		t.Fatalf("NewInboxManager: %v", err)
	}
	return m
}

func TestInboxManager_HandleEvent_PermissionRequest_CreatesItem(t *testing.T) {
	m := newTestInboxManager(t)
	m.HandleEvent(makeEventWithTool("e1", "hook.permission-request", "agent-1", "Bash"))

	items := m.List("", false)
	if len(items) != 1 {
		t.Fatalf("want 1 item, got %d", len(items))
	}
	if items[0].Type != core.InboxItemPermissionRequest {
		t.Errorf("want type=%q, got %q", core.InboxItemPermissionRequest, items[0].Type)
	}
	if !items[0].Actionable {
		t.Error("want Actionable=true for permission_request")
	}
	if items[0].AgentID != "agent-1" {
		t.Errorf("want AgentID=agent-1, got %q", items[0].AgentID)
	}
}

func TestInboxManager_HandleEvent_AllThreeHooks_CreateItems(t *testing.T) {
	m := newTestInboxManager(t)

	m.HandleEvent(makeEventWithTool("e1", "hook.permission-request", "a", "Bash"))
	m.HandleEvent(makeEvent("e2", "hook.notification", "a"))
	m.HandleEvent(makeEventWithTask("e3", "hook.task-completed", "a", "my-task"))
	// These should be ignored now:
	m.HandleEvent(makeEvent("e4", "hook.stop", "a"))
	m.HandleEvent(makeEvent("e5", "hook.stop-failure", "a"))
	m.HandleEvent(makeEventWithTool("e6", "hook.tool-failed", "a", "Read"))

	items := m.List("", false)
	if len(items) != 3 {
		t.Fatalf("want 3 items, got %d", len(items))
	}

	// Items are returned newest-first, so reverse order of insertion.
	wantTypes := []core.InboxItemType{
		core.InboxItemTaskComplete,      // e3 task-completed
		core.InboxItemNotification,      // e2 notification
		core.InboxItemPermissionRequest, // e1 permission-request
	}
	for i, want := range wantTypes {
		if items[i].Type != want {
			t.Errorf("items[%d]: want type=%q, got %q", i, want, items[i].Type)
		}
	}
}

func TestInboxManager_HandleEvent_NonInboxHook_Ignored(t *testing.T) {
	m := newTestInboxManager(t)
	// Events with no HookOrigin (or a non-inbox hook name) are not inbox-eligible.
	m.HandleEvent(makeEvent("e1", "", "a"))
	m.HandleEvent(makeEvent("e2", "hook.tool-start", "a"))
	m.HandleEvent(makeEvent("e3", "hook.session-start", "a"))

	items := m.List("", false)
	if len(items) != 0 {
		t.Errorf("want 0 items for non-inbox hooks, got %d", len(items))
	}
}

func TestInboxManager_RingBuffer_DropsOldest(t *testing.T) {
	m := newTestInboxManager(t)

	for i := 0; i <= 100; i++ {
		e := makeEvent(fmt.Sprintf("e%d", i), "hook.notification", "a")
		e.Summary = fmt.Sprintf("notification-%d", i)
		m.HandleEvent(e)
	}

	items := m.List("", false)
	if len(items) != 100 {
		t.Fatalf("want 100 items (ring buffer), got %d", len(items))
	}

	// Newest item first: e100 (summary "notification-100").
	if items[0].Summary != "notification-100" {
		t.Errorf("want newest item first, got summary=%q", items[0].Summary)
	}

	// Oldest item (e0 "notification-0") should be gone; last item should be e1.
	last := items[len(items)-1]
	if last.Summary == "notification-0" {
		t.Error("oldest item e0 should have been dropped by ring buffer")
	}
}

func TestInboxManager_MarkRead_DecrementsUnreadCount(t *testing.T) {
	m := newTestInboxManager(t)
	m.HandleEvent(makeEvent("e1", "hook.notification", "a"))
	m.HandleEvent(makeEvent("e2", "hook.notification", "a"))
	m.HandleEvent(makeEvent("e3", "hook.notification", "a"))

	if got := m.UnreadCount(); got != 3 {
		t.Fatalf("want UnreadCount=3, got %d", got)
	}

	items := m.List("", false)
	m.MarkRead(items[0].ID) // mark newest read

	if got := m.UnreadCount(); got != 2 {
		t.Errorf("want UnreadCount=2 after MarkRead, got %d", got)
	}
}

func TestInboxManager_MarkAllRead(t *testing.T) {
	m := newTestInboxManager(t)
	for i := 0; i < 5; i++ {
		m.HandleEvent(makeEvent(fmt.Sprintf("e%d", i), "hook.notification", "a"))
	}

	if got := m.UnreadCount(); got != 5 {
		t.Fatalf("want UnreadCount=5, got %d", got)
	}

	m.MarkAllRead()

	if got := m.UnreadCount(); got != 0 {
		t.Errorf("want UnreadCount=0 after MarkAllRead, got %d", got)
	}
}

func TestInboxManager_Persist_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "inbox.json")

	// Manager A: add 3 items.
	mA, err := NewInboxManager(path)
	if err != nil {
		t.Fatalf("NewInboxManager A: %v", err)
	}
	mA.HandleEvent(makeEvent("e1", "hook.notification", "agent-1"))
	mA.HandleEvent(makeEventWithTool("e2", "hook.permission-request", "agent-2", "Write"))
	mA.HandleEvent(makeEventWithTask("e3", "hook.task-completed", "agent-3", "build"))

	// Manager B: load same path.
	mB, err := NewInboxManager(path)
	if err != nil {
		t.Fatalf("NewInboxManager B: %v", err)
	}

	itemsB := mB.List("", false)
	if len(itemsB) != 3 {
		t.Fatalf("want 3 items after reload, got %d", len(itemsB))
	}

	// IDs should match (newest first from B, so e3, e2, e1).
	wantIDs := []string{"e3", "e2", "e1"}
	for i, wantID := range wantIDs {
		if itemsB[i].ID != wantID {
			t.Errorf("items[%d].ID: want %q, got %q", i, wantID, itemsB[i].ID)
		}
	}
}

func TestInboxManager_HandleEvent_ResolvesAgentName(t *testing.T) {
	m := newTestInboxManager(t)
	m.SetAgentNameResolver(func(id string) string {
		if id == "a1" {
			return "alice"
		}
		return ""
	})
	m.HandleEvent(makeEventWithTool("e1", "hook.permission-request", "a1", "Bash"))
	items := m.List("", false)
	if len(items) != 1 || items[0].AgentName != "alice" {
		t.Fatalf("want AgentName=alice, got %+v", items)
	}
}

func TestInboxManager_HandleEvent_ResolverNil_FallsBackToEmpty(t *testing.T) {
	m := newTestInboxManager(t)
	// No resolver set
	m.HandleEvent(makeEventWithTool("e1", "hook.permission-request", "a1", "Bash"))
	items := m.List("", false)
	if items[0].AgentName != "" {
		t.Fatalf("want empty AgentName fallback, got %q", items[0].AgentName)
	}
}

// Test 10: loading from a missing file returns empty inbox with no error.
func TestInboxManager_LoadFromMissingFile_Empty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent", "inbox.json")
	m, err := NewInboxManager(path)
	if err != nil {
		t.Fatalf("NewInboxManager with missing file should not error, got: %v", err)
	}
	items := m.List("", false)
	if len(items) != 0 {
		t.Errorf("want 0 items for missing file, got %d", len(items))
	}
}
