package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/core"
)

const inboxRingSize = 100

// InboxManager keeps a ring buffer of recent user-facing notifications and persists them.
type InboxManager struct {
	mu               sync.Mutex
	items            []core.InboxItem // newest at end
	path             string
	resolveAgentName func(agentID string) string
}

// NewInboxManager creates an InboxManager backed by the given path.
// If the file does not exist, an empty inbox is returned without error.
func NewInboxManager(path string) (*InboxManager, error) {
	m := &InboxManager{path: path}
	if err := m.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return m, nil
}

// SetAgentNameResolver installs an optional callback that maps an agent ID to a
// human-readable display name. It must be goroutine-safe (the resolver is called
// while the manager lock is NOT held). Call this before HandleEvent fires, or
// ensure the resolver itself is safe for concurrent use.
func (m *InboxManager) SetAgentNameResolver(fn func(agentID string) string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resolveAgentName = fn
}

// HandleEvent converts a hook event into an InboxItem if its Type matches one of the
// 6 inbox-eligible hooks. Other event types are ignored.
func (m *InboxManager) HandleEvent(event core.Event) {
	item, ok := buildInboxItem(event)
	if !ok {
		return
	}
	m.mu.Lock()
	resolver := m.resolveAgentName
	m.mu.Unlock()

	if resolver != nil {
		item.AgentName = resolver(item.AgentID)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = append(m.items, item)
	if len(m.items) > inboxRingSize {
		m.items = m.items[len(m.items)-inboxRingSize:]
	}
	_ = m.persistLocked()
}

// List returns inbox items, optionally filtered by type and unread state. Newest first.
func (m *InboxManager) List(typeFilter core.InboxItemType, unreadOnly bool) []core.InboxItem {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]core.InboxItem, 0, len(m.items))
	for i := len(m.items) - 1; i >= 0; i-- {
		item := m.items[i]
		if typeFilter != "" && item.Type != typeFilter {
			continue
		}
		if unreadOnly && item.Read {
			continue
		}
		result = append(result, item)
	}
	return result
}

// MarkRead sets Read=true on the item with the given ID and persists.
func (m *InboxManager) MarkRead(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.items {
		if m.items[i].ID == id {
			m.items[i].Read = true
			break
		}
	}
	_ = m.persistLocked()
}

// MarkAllRead sets Read=true on all items and persists.
func (m *InboxManager) MarkAllRead() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.items {
		m.items[i].Read = true
	}
	_ = m.persistLocked()
}

// UnreadCount returns the number of unread inbox items.
func (m *InboxManager) UnreadCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for _, item := range m.items {
		if !item.Read {
			count++
		}
	}
	return count
}

// buildInboxItem maps an event to an inbox item if it matches one of the 6 hook types.
func buildInboxItem(event core.Event) (core.InboxItem, bool) {
	var itemType core.InboxItemType
	var summary string
	var actionable bool

	switch event.HookOrigin {
	case "hook.permission-request":
		itemType = core.InboxItemPermissionRequest
		actionable = true
		if event.ToolName != "" {
			summary = fmt.Sprintf("Tool %s requested permission", event.ToolName)
		} else {
			summary = "Permission requested"
		}
	case "hook.notification":
		itemType = core.InboxItemNotification
		if event.Summary != "" {
			summary = event.Summary
		} else {
			summary = "Notification received"
		}
	case "hook.task-completed":
		itemType = core.InboxItemTaskComplete
		if event.TaskName != "" {
			summary = fmt.Sprintf("Task '%s' completed", event.TaskName)
		} else if event.Summary != "" {
			summary = event.Summary
		} else {
			summary = "Task completed"
		}
	default:
		return core.InboxItem{}, false
	}

	id := event.ID
	if id == "" {
		id = fmt.Sprintf("inbox-%d", time.Now().UnixNano())
	}

	occurredAt := event.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	return core.InboxItem{
		ID:         id,
		AgentID:    event.AgentID,
		Type:       itemType,
		Summary:    summary,
		ToolName:   event.ToolName,
		OccurredAt: occurredAt,
		Read:       false,
		Actionable: actionable,
	}, true
}

func (m *InboxManager) load() error {
	data, err := os.ReadFile(m.path)
	if err != nil {
		return err
	}
	var items []core.InboxItem
	if err := json.Unmarshal(data, &items); err != nil {
		return fmt.Errorf("decode inbox: %w", err)
	}
	m.items = items
	return nil
}

func (m *InboxManager) persistLocked() error {
	if err := os.MkdirAll(filepath.Dir(m.path), 0o755); err != nil {
		return fmt.Errorf("create inbox directory: %w", err)
	}
	data, err := json.Marshal(m.items)
	if err != nil {
		return fmt.Errorf("marshal inbox: %w", err)
	}
	tmpPath := m.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write inbox tmp: %w", err)
	}
	if err := os.Rename(tmpPath, m.path); err != nil {
		return fmt.Errorf("rename inbox: %w", err)
	}
	return nil
}
