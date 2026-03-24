package core

import "time"

type AgentMode string

const (
	AgentModeManaged  AgentMode = "managed"
	AgentModeAttached AgentMode = "attached"
	AgentModeObserved AgentMode = "observed"
)

type AgentStatus string

const (
	AgentStatusBooting      AgentStatus = "booting"
	AgentStatusIdle         AgentStatus = "idle"
	AgentStatusThinking     AgentStatus = "thinking"
	AgentStatusReading      AgentStatus = "reading"
	AgentStatusRunningTool  AgentStatus = "running_tool"
	AgentStatusWaitingInput AgentStatus = "waiting_input"
	AgentStatusDone         AgentStatus = "done"
	AgentStatusError        AgentStatus = "error"
	AgentStatusDisconnected AgentStatus = "disconnected"
	AgentStatusSleeping     AgentStatus = "sleeping"
)

type NotificationPolicy string

const (
	NotificationPolicyDefault      NotificationPolicy = "default"
	NotificationPolicyMuted        NotificationPolicy = "muted"
	NotificationPolicyPriorityOnly NotificationPolicy = "priority_only"
)

type Agent struct {
	ID                     string             `json:"id"`
	DisplayName            string             `json:"display_name"`
	Provider               string             `json:"provider"`
	Host                   string             `json:"host"`
	Mode                   AgentMode          `json:"mode"`
	ProjectPath            string             `json:"project_path"`
	Role                   string             `json:"role,omitempty"`
	Status                 AgentStatus        `json:"status"`
	StatusConfidence       float64            `json:"status_confidence"`
	LastEventAt            time.Time          `json:"last_event_at"`
	LastUserVisibleSummary string             `json:"last_user_visible_summary,omitempty"`
	NotificationPolicy     NotificationPolicy `json:"notification_policy"`
	SessionRef             string             `json:"session_ref,omitempty"`
	AvatarVariant          string             `json:"avatar_variant"`
}

type RuntimeSnapshot struct {
	Agents      []Agent   `json:"agents"`
	GeneratedAt time.Time `json:"generated_at"`
}

type EventType string

const (
	EventTypeAgentRegistered EventType = "agent.registered"
)

type Event struct {
	ID         string    `json:"id"`
	AgentID    string    `json:"agent_id"`
	Type       EventType `json:"type"`
	Summary    string    `json:"summary"`
	OccurredAt time.Time `json:"occurred_at"`
}

func (s RuntimeSnapshot) TotalCount() int {
	return len(s.Agents)
}

func (s RuntimeSnapshot) RunningCount() int {
	count := 0
	for _, agent := range s.Agents {
		switch agent.Status {
		case AgentStatusBooting, AgentStatusThinking, AgentStatusReading, AgentStatusRunningTool:
			count++
		}
	}
	return count
}

func (s RuntimeSnapshot) WaitingCount() int {
	count := 0
	for _, agent := range s.Agents {
		if agent.Status == AgentStatusWaitingInput {
			count++
		}
	}
	return count
}

func (s RuntimeSnapshot) DoneCount() int {
	count := 0
	for _, agent := range s.Agents {
		if agent.Status == AgentStatusDone {
			count++
		}
	}
	return count
}
