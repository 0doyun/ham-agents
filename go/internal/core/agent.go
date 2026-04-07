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
	AgentStatusWriting     AgentStatus = "writing"
	AgentStatusSearching   AgentStatus = "searching"
	AgentStatusSpawning    AgentStatus = "spawning"
)

type NotificationPolicy string

const (
	NotificationPolicyDefault      NotificationPolicy = "default"
	NotificationPolicyMuted        NotificationPolicy = "muted"
	NotificationPolicyPriorityOnly NotificationPolicy = "priority_only"
)

type Agent struct {
	ID                      string             `json:"id"`
	DisplayName             string             `json:"display_name"`
	Provider                string             `json:"provider"`
	Host                    string             `json:"host"`
	Mode                    AgentMode          `json:"mode"`
	ProjectPath             string             `json:"project_path"`
	Role                    string             `json:"role,omitempty"`
	Status                  AgentStatus        `json:"status"`
	StatusConfidence        float64            `json:"status_confidence"`
	StatusReason            string             `json:"status_reason,omitempty"`
	ErrorType               string             `json:"error_type,omitempty"`
	RegisteredAt            time.Time          `json:"registered_at,omitempty"`
	LastEventAt             time.Time          `json:"last_event_at"`
	LastUserVisibleSummary  string             `json:"last_user_visible_summary,omitempty"`
	RecentTools             []string           `json:"recent_tools,omitempty"`
	RecentToolsDetailed     []ToolActivity     `json:"recent_tools_detailed,omitempty"`
	OmcMode                 string             `json:"omc_mode,omitempty"`
	NotificationPolicy      NotificationPolicy `json:"notification_policy"`
	SessionID               string             `json:"session_id,omitempty"`
	SessionRef              string             `json:"session_ref,omitempty"`
	SessionTitle            string             `json:"session_title,omitempty"`
	SessionIsActive         bool               `json:"session_is_active,omitempty"`
	SessionTTY              string             `json:"session_tty,omitempty"`
	SessionWindowIndex      int                `json:"session_window_index,omitempty"`
	SessionTabIndex         int                `json:"session_tab_index,omitempty"`
	SessionWorkingDirectory string             `json:"session_working_directory,omitempty"`
	SessionActivity         string             `json:"session_activity,omitempty"`
	SessionProcessID        int                `json:"session_process_id,omitempty"`
	SessionCommand          string             `json:"session_command,omitempty"`
	AvatarVariant           string             `json:"avatar_variant"`
	LastAssistantMessage    string             `json:"last_assistant_message,omitempty"`
	SubAgentCount           int                `json:"sub_agent_count,omitempty"`
	SubAgents               []SubAgentInfo     `json:"sub_agents,omitempty"`
	TeamRole                string             `json:"team_role,omitempty"`
	TeamTaskTotal           int                `json:"team_task_total,omitempty"`
	TeamTaskCompleted       int                `json:"team_task_completed,omitempty"`
}

type ToolActivity struct {
	ToolName     string     `json:"tool_name"`
	InputPreview string     `json:"input_preview,omitempty"`
	ActivityDesc string     `json:"activity_desc,omitempty"`
	ToolType     string     `json:"tool_type,omitempty"`
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	DurationMs   int        `json:"duration_ms,omitempty"`
}

// MaxRecentToolsDetailed is the maximum number of entries kept in RecentToolsDetailed.
const MaxRecentToolsDetailed = 20

// ClassifyToolType returns a tool type category based on the tool name.
func ClassifyToolType(toolName string) string {
	switch toolName {
	case "Bash":
		return "bash"
	case "Read", "Write", "Edit", "NotebookEdit":
		return "file"
	case "Grep", "Glob":
		return "search"
	case "Agent":
		return "agent"
	case "WebFetch", "WebSearch":
		return "web"
	default:
		if len(toolName) > 4 && toolName[:4] == "mcp_" {
			return "mcp"
		}
		return "util"
	}
}

type SubAgentInfo struct {
	AgentID   string      `json:"agent_id"`
	AgentType string      `json:"agent_type"`
	Status    AgentStatus `json:"status"`
	StartTime time.Time   `json:"start_time"`
	EndTime   *time.Time  `json:"end_time,omitempty"`
	Summary   string      `json:"summary,omitempty"`
}

type RuntimeSnapshot struct {
	Agents             []Agent            `json:"agents"`
	GeneratedAt        time.Time          `json:"generated_at"`
	AttentionCount     int                `json:"attention_count"`
	AttentionBreakdown AttentionBreakdown `json:"attention_breakdown"`
	AttentionOrder     []string           `json:"attention_order"`
	AttentionSubtitles map[string]string  `json:"attention_subtitles"`
}

type AttentionBreakdown struct {
	Error        int `json:"error"`
	WaitingInput int `json:"waiting_input"`
	Disconnected int `json:"disconnected"`
}

type EventType string

const (
	EventTypeAgentRegistered                EventType = "agent.registered"
	EventTypeAgentRoleUpdated               EventType = "agent.role_updated"
	EventTypeAgentNotificationPolicyUpdated EventType = "agent.notification_policy_updated"
	EventTypeAgentDisconnected              EventType = "agent.disconnected"
	EventTypeAgentReconnected               EventType = "agent.reconnected"
	EventTypeAgentRemoved                   EventType = "agent.removed"
	EventTypeAgentStatusUpdated             EventType = "agent.status_updated"
	EventTypeAgentProcessStarted            EventType = "agent.process_started"
	EventTypeAgentProcessOutput             EventType = "agent.process_output"
	EventTypeAgentProcessExited             EventType = "agent.process_exited"
	EventTypeAgentLayoutChanged             EventType = "agent.layout_changed"
	EventTypeTeammateIdle                   EventType = "team.teammate_idle"
	EventTypeTeamTaskCreated                EventType = "team.task_created"
	EventTypeTeamTaskCompleted              EventType = "team.task_completed"
)

type Event struct {
	ID                   string    `json:"id"`
	AgentID              string    `json:"agent_id"`
	Type                 EventType `json:"type"`
	Summary              string    `json:"summary"`
	OccurredAt           time.Time `json:"occurred_at"`
	PresentationLabel    string    `json:"presentation_label,omitempty"`
	PresentationEmphasis string    `json:"presentation_emphasis,omitempty"`
	PresentationSummary  string    `json:"presentation_summary,omitempty"`
	LifecycleStatus      string    `json:"lifecycle_status,omitempty"`
	LifecycleMode        string    `json:"lifecycle_mode,omitempty"`
	LifecycleReason      string    `json:"lifecycle_reason,omitempty"`
	LifecycleConfidence  float64   `json:"lifecycle_confidence,omitempty"`
	SessionID            string    `json:"session_id,omitempty"`
	ParentAgentID        string    `json:"parent_agent_id,omitempty"`
	TaskName             string    `json:"task_name,omitempty"`
	TaskDesc             string    `json:"task_desc,omitempty"`
	ArtifactType         string    `json:"artifact_type,omitempty"`
	ArtifactRef          string    `json:"artifact_ref,omitempty"`
	ArtifactData         string    `json:"artifact_data,omitempty"`
	ToolName             string    `json:"tool_name,omitempty"`
	ToolInput            string    `json:"tool_input,omitempty"`
	ToolType             string    `json:"tool_type,omitempty"`
	ToolDuration         int       `json:"tool_duration_ms,omitempty"`
}

type OpenTargetKind string

const (
	OpenTargetKindExternalURL  OpenTargetKind = "external_url"
	OpenTargetKindItermSession OpenTargetKind = "iterm_session"
	OpenTargetKindTmuxPane     OpenTargetKind = "tmux_pane"
	OpenTargetKindWorkspace    OpenTargetKind = "workspace"
)

type OpenTarget struct {
	Kind        OpenTargetKind `json:"kind"`
	Value       string         `json:"value"`
	Application string         `json:"application,omitempty"`
	SessionID   string         `json:"session_id,omitempty"`
}

type AttachableSession struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	SessionRef       string `json:"session_ref"`
	IsActive         bool   `json:"is_active"`
	TTY              string `json:"tty,omitempty"`
	WindowIndex      int    `json:"window_index,omitempty"`
	TabIndex         int    `json:"tab_index,omitempty"`
	WorkingDirectory string `json:"working_directory,omitempty"`
	Activity         string `json:"activity,omitempty"`
	ProcessID        int    `json:"process_id,omitempty"`
	Command          string `json:"command,omitempty"`
}

func (s RuntimeSnapshot) TotalCount() int {
	return len(s.Agents)
}

func (s RuntimeSnapshot) RunningCount() int {
	count := 0
	for _, agent := range s.Agents {
		if IsRunningStatus(agent.Status) {
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
