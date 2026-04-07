package core

import "time"

// InboxItemType discriminates user-facing notifications surfaced from hook events.
type InboxItemType string

const (
	InboxItemPermissionRequest InboxItemType = "permission_request"
	InboxItemNotification      InboxItemType = "notification"
	InboxItemTaskComplete      InboxItemType = "task_complete"
	InboxItemError             InboxItemType = "error"
	InboxItemStop              InboxItemType = "stop"
)

// InboxItem is a single user-facing entry in the notification inbox.
type InboxItem struct {
	ID         string        `json:"id"`
	AgentID    string        `json:"agent_id"`
	AgentName  string        `json:"agent_name"`
	Type       InboxItemType `json:"type"`
	Summary    string        `json:"summary"`
	ToolName   string        `json:"tool_name,omitempty"`
	OccurredAt time.Time     `json:"occurred_at"`
	Read       bool          `json:"read"`
	Actionable bool          `json:"actionable"`
}
