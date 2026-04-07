package ipc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/core"
	hamruntime "github.com/ham-agents/ham-agents/go/internal/runtime"
)

type Config struct {
	SocketPath string
}

type Command string

const (
	CommandRunManaged            Command = "run.managed"
	CommandRegisterManaged       Command = "register.managed"
	CommandNotifyManagedExited   Command = "managed.exited"
	CommandRecordOutput          Command = "managed.output"
	CommandAttachSession         Command = "attach.session"
	CommandObserveSource         Command = "observe.source"
	CommandCreateTeam            Command = "teams.create"
	CommandAddTeamMember         Command = "teams.add_member"
	CommandListTeams             Command = "teams.list"
	CommandStopManaged           Command = "managed.stop"
	CommandOpenTarget            Command = "agents.open_target"
	CommandListItermSessions     Command = "iterm.sessions"
	CommandListTmuxSessions      Command = "tmux.sessions"
	CommandListAgents            Command = "agents.list"
	CommandStatus                Command = "agents.status"
	CommandEvents                Command = "events.list"
	CommandFollowEvents          Command = "events.follow"
	CommandSetNotificationPolicy Command = "agents.set_notification_policy"
	CommandSetRole               Command = "agents.set_role"
	CommandRenameAgent           Command = "agents.rename"
	CommandRemoveAgent           Command = "agents.remove"
	CommandGetSettings           Command = "settings.get"
	CommandUpdateSettings        Command = "settings.update"
	CommandShutdown              Command = "daemon.shutdown"
	CommandHookToolStart         Command = "hook.tool-start"
	CommandHookToolDone          Command = "hook.tool-done"
	CommandHookNotification      Command = "hook.notification"
	CommandHookStopFailure       Command = "hook.stop-failure"
	CommandHookSessionStart      Command = "hook.session-start"
	CommandHookSessionEnd        Command = "hook.session-end"
	CommandHookAgentSpawned      Command = "hook.agent-spawned"
	CommandHookAgentFinished     Command = "hook.agent-finished"
	CommandHookStop              Command = "hook.stop"
	CommandHookTeammateIdle      Command = "hook.teammate-idle"
	CommandHookTaskCreated       Command = "hook.task-created"
	CommandHookTaskCompleted      Command = "hook.task-completed"
	CommandHookToolFailed        Command = "hook.tool-failed"
	CommandHookUserPrompt        Command = "hook.user-prompt"
	CommandHookPermissionReq     Command = "hook.permission-request"
	CommandHookPermissionDenied  Command = "hook.permission-denied"
	CommandHookPreCompact        Command = "hook.pre-compact"
	CommandHookPostCompact       Command = "hook.post-compact"
	CommandHookSetup             Command = "hook.setup"
	CommandHookElicitation       Command = "hook.elicitation"
	CommandHookElicitationResult Command = "hook.elicitation-result"
	CommandHookConfigChange      Command = "hook.config-change"
	CommandHookWorktreeCreate    Command = "hook.worktree-create"
	CommandHookWorktreeRemove    Command = "hook.worktree-remove"
	CommandHookInstructions      Command = "hook.instructions-loaded"
	CommandHookCwdChanged        Command = "hook.cwd-changed"
	CommandHookFileChanged       Command = "hook.file-changed"

	CommandInboxList     Command = "inbox.list"
	CommandInboxMarkRead Command = "inbox.mark-read"
)

type Request struct {
	Command          Command        `json:"command"`
	AgentID          string         `json:"agent_id,omitempty"`
	Provider         string         `json:"provider,omitempty"`
	DisplayName      string         `json:"display_name,omitempty"`
	ProjectPath      string         `json:"project_path,omitempty"`
	Role             string         `json:"role,omitempty"`
	SessionRef       string         `json:"session_ref,omitempty"`
	TeamRef          string         `json:"team_ref,omitempty"`
	MemberAgentID    string         `json:"member_agent_id,omitempty"`
	Limit            int            `json:"limit,omitempty"`
	AfterEventID     string         `json:"after_event_id,omitempty"`
	WaitMillis       int            `json:"wait_millis,omitempty"`
	Policy           string         `json:"policy,omitempty"`
	Settings         *core.Settings `json:"settings,omitempty"`
	ExitError        string         `json:"exit_error,omitempty"`
	OutputLine       string         `json:"output_line,omitempty"`
	ToolName         string         `json:"tool_name,omitempty"`
	ToolInputPreview string         `json:"tool_input_preview,omitempty"`
	OmcMode          string         `json:"omc_mode,omitempty"`
	SessionID        string         `json:"session_id,omitempty"`
	NotificationType string         `json:"notification_type,omitempty"`
	ErrorType        string         `json:"error_type,omitempty"`
	HookType         string         `json:"hook_type,omitempty"`
	Description      string         `json:"description,omitempty"`
	TeammateName     string         `json:"teammate_name,omitempty"`
	TeamRole         string         `json:"team_role,omitempty"`
	TaskName         string         `json:"task_name,omitempty"`
	TaskDescription  string         `json:"task_description,omitempty"`
	IsInterrupt      bool           `json:"is_interrupt,omitempty"`
	Prompt           string         `json:"prompt,omitempty"`
	CompactSummary   string         `json:"compact_summary,omitempty"`
	CompactTrigger   string         `json:"compact_trigger,omitempty"`
	WorktreeName     string         `json:"worktree_name,omitempty"`
	WorktreePath     string         `json:"worktree_path,omitempty"`
	OldCwd           string         `json:"old_cwd,omitempty"`
	NewCwd           string         `json:"new_cwd,omitempty"`
	FilePath         string         `json:"file_path,omitempty"`
	FileEvent        string         `json:"file_event,omitempty"`
	LastMessage      string         `json:"last_message,omitempty"`
	Graph            bool           `json:"graph,omitempty"`
	TypeFilter       string         `json:"type_filter,omitempty"`
	UnreadOnly       bool           `json:"unread_only,omitempty"`
	InboxItemID      string         `json:"inbox_item_id,omitempty"`
}

type Response struct {
	Agent              *core.Agent              `json:"agent,omitempty"`
	Team               *core.Team               `json:"team,omitempty"`
	Agents             []core.Agent             `json:"agents,omitempty"`
	Teams              []core.Team              `json:"teams,omitempty"`
	Events             []core.Event             `json:"events,omitempty"`
	AttachableSessions []core.AttachableSession `json:"attachable_sessions,omitempty"`
	OpenTarget         *core.OpenTarget         `json:"open_target,omitempty"`
	Settings           *core.Settings           `json:"settings,omitempty"`
	Snapshot           *core.RuntimeSnapshot    `json:"snapshot,omitempty"`
	SessionGraph       *core.SessionGraph       `json:"session_graph,omitempty"`
	InboxItems         []core.InboxItem         `json:"inbox_items,omitempty"`
	UnreadCount        int                      `json:"unread_count,omitempty"`
	Error              string                   `json:"error,omitempty"`
}

func DefaultSocketPath() (string, error) {
	if path := os.Getenv("HAM_AGENTS_SOCKET"); path != "" {
		return path, nil
	}

	if root := os.Getenv("HAM_AGENTS_HOME"); root != "" {
		return filepath.Join(root, "hamd.sock"), nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}

	return filepath.Join(homeDir, "Library", "Application Support", "ham-agents", "hamd.sock"), nil
}

func DefaultConfig() (Config, error) {
	socketPath, err := DefaultSocketPath()
	if err != nil {
		return Config{}, err
	}

	return Config{SocketPath: socketPath}, nil
}

type Client struct {
	socketPath string
	timeout    time.Duration
}

func NewClient(socketPath string) *Client {
	return &Client{socketPath: socketPath, timeout: 3 * time.Second}
}

func (c *Client) RunManaged(ctx context.Context, input hamruntime.RegisterManagedInput) (core.Agent, error) {
	response, err := c.request(ctx, Request{
		Command:     CommandRunManaged,
		Provider:    input.Provider,
		DisplayName: input.DisplayName,
		ProjectPath: input.ProjectPath,
		Role:        input.Role,
	})
	if err != nil {
		return core.Agent{}, err
	}
	if response.Agent == nil {
		return core.Agent{}, fmt.Errorf("daemon response missing agent payload")
	}
	return *response.Agent, nil
}

func (c *Client) RegisterManaged(ctx context.Context, input hamruntime.RegisterManagedInput) (core.Agent, error) {
	response, err := c.request(ctx, Request{
		Command:     CommandRegisterManaged,
		Provider:    input.Provider,
		DisplayName: input.DisplayName,
		ProjectPath: input.ProjectPath,
		Role:        input.Role,
		SessionRef:  input.SessionRef,
	})
	if err != nil {
		return core.Agent{}, err
	}
	if response.Agent == nil {
		return core.Agent{}, fmt.Errorf("daemon response missing agent payload")
	}
	return *response.Agent, nil
}

func (c *Client) NotifyManagedExited(ctx context.Context, agentID string, exitErr error) error {
	exitError := ""
	if exitErr != nil {
		exitError = exitErr.Error()
	}
	_, err := c.request(ctx, Request{
		Command:   CommandNotifyManagedExited,
		AgentID:   agentID,
		ExitError: exitError,
	})
	return err
}

func (c *Client) RecordOutput(ctx context.Context, agentID string, line string) error {
	_, err := c.request(ctx, Request{
		Command:    CommandRecordOutput,
		AgentID:    agentID,
		OutputLine: line,
	})
	return err
}

func (c *Client) AttachSession(ctx context.Context, input hamruntime.RegisterAttachedInput) (core.Agent, error) {
	response, err := c.request(ctx, Request{
		Command:     CommandAttachSession,
		Provider:    input.Provider,
		DisplayName: input.DisplayName,
		ProjectPath: input.ProjectPath,
		Role:        input.Role,
		SessionRef:  input.SessionRef,
	})
	if err != nil {
		return core.Agent{}, err
	}
	if response.Agent == nil {
		return core.Agent{}, fmt.Errorf("daemon response missing agent payload")
	}
	return *response.Agent, nil
}

func (c *Client) ObserveSource(ctx context.Context, input hamruntime.RegisterObservedInput) (core.Agent, error) {
	response, err := c.request(ctx, Request{
		Command:     CommandObserveSource,
		Provider:    input.Provider,
		DisplayName: input.DisplayName,
		ProjectPath: input.ProjectPath,
		Role:        input.Role,
		SessionRef:  input.SessionRef,
	})
	if err != nil {
		return core.Agent{}, err
	}
	if response.Agent == nil {
		return core.Agent{}, fmt.Errorf("daemon response missing agent payload")
	}
	return *response.Agent, nil
}

func (c *Client) OpenTarget(ctx context.Context, agentID string) (core.OpenTarget, error) {
	response, err := c.request(ctx, Request{
		Command: CommandOpenTarget,
		AgentID: agentID,
	})
	if err != nil {
		return core.OpenTarget{}, err
	}
	if response.OpenTarget == nil {
		return core.OpenTarget{}, fmt.Errorf("daemon response missing open target payload")
	}
	return *response.OpenTarget, nil
}

func (c *Client) CreateTeam(ctx context.Context, name string) (core.Team, error) {
	response, err := c.request(ctx, Request{Command: CommandCreateTeam, DisplayName: name})
	if err != nil {
		return core.Team{}, err
	}
	if response.Team == nil {
		return core.Team{}, fmt.Errorf("daemon response missing team payload")
	}
	return *response.Team, nil
}

func (c *Client) AddTeamMember(ctx context.Context, teamRef string, agentID string) (core.Team, error) {
	response, err := c.request(ctx, Request{Command: CommandAddTeamMember, TeamRef: teamRef, MemberAgentID: agentID})
	if err != nil {
		return core.Team{}, err
	}
	if response.Team == nil {
		return core.Team{}, fmt.Errorf("daemon response missing team payload")
	}
	return *response.Team, nil
}

func (c *Client) ListTeams(ctx context.Context) ([]core.Team, error) {
	response, err := c.request(ctx, Request{Command: CommandListTeams})
	if err != nil {
		return nil, err
	}
	return response.Teams, nil
}

func (c *Client) ListItermSessions(ctx context.Context) ([]core.AttachableSession, error) {
	response, err := c.request(ctx, Request{Command: CommandListItermSessions})
	if err != nil {
		return nil, err
	}
	return response.AttachableSessions, nil
}

func (c *Client) ListTmuxSessions(ctx context.Context) ([]core.AttachableSession, error) {
	response, err := c.request(ctx, Request{Command: CommandListTmuxSessions})
	if err != nil {
		return nil, err
	}
	return response.AttachableSessions, nil
}

func (c *Client) Settings(ctx context.Context) (core.Settings, error) {
	response, err := c.request(ctx, Request{Command: CommandGetSettings})
	if err != nil {
		return core.Settings{}, err
	}
	if response.Settings == nil {
		return core.Settings{}, fmt.Errorf("daemon response missing settings payload")
	}
	return *response.Settings, nil
}

func (c *Client) UpdateSettings(ctx context.Context, settings core.Settings) (core.Settings, error) {
	response, err := c.request(ctx, Request{Command: CommandUpdateSettings, Settings: &settings})
	if err != nil {
		return core.Settings{}, err
	}
	if response.Settings == nil {
		return core.Settings{}, fmt.Errorf("daemon response missing settings payload")
	}
	return *response.Settings, nil
}

func (c *Client) ListAgents(ctx context.Context) ([]core.Agent, error) {
	response, err := c.request(ctx, Request{Command: CommandListAgents})
	if err != nil {
		return nil, err
	}
	return response.Agents, nil
}

func (c *Client) Status(ctx context.Context) (core.RuntimeSnapshot, error) {
	response, err := c.request(ctx, Request{Command: CommandStatus})
	if err != nil {
		return core.RuntimeSnapshot{}, err
	}
	if response.Snapshot == nil {
		return core.RuntimeSnapshot{}, fmt.Errorf("daemon response missing snapshot payload")
	}
	return *response.Snapshot, nil
}

func (c *Client) Events(ctx context.Context, limit int) ([]core.Event, error) {
	response, err := c.request(ctx, Request{Command: CommandEvents, Limit: limit})
	if err != nil {
		return nil, err
	}
	return response.Events, nil
}

func (c *Client) FollowEvents(ctx context.Context, afterEventID string, limit int, wait time.Duration) ([]core.Event, error) {
	response, err := c.request(ctx, Request{
		Command:      CommandFollowEvents,
		AfterEventID: afterEventID,
		Limit:        limit,
		WaitMillis:   int(wait / time.Millisecond),
	})
	if err != nil {
		return nil, err
	}
	return response.Events, nil
}

func (c *Client) UpdateNotificationPolicy(ctx context.Context, agentID string, policy core.NotificationPolicy) (core.Agent, error) {
	response, err := c.request(ctx, Request{
		Command: CommandSetNotificationPolicy,
		AgentID: agentID,
		Policy:  string(policy),
	})
	if err != nil {
		return core.Agent{}, err
	}
	if response.Agent == nil {
		return core.Agent{}, fmt.Errorf("daemon response missing agent payload")
	}
	return *response.Agent, nil
}

func (c *Client) UpdateRole(ctx context.Context, agentID string, role string) (core.Agent, error) {
	response, err := c.request(ctx, Request{
		Command: CommandSetRole,
		AgentID: agentID,
		Role:    role,
	})
	if err != nil {
		return core.Agent{}, err
	}
	if response.Agent == nil {
		return core.Agent{}, fmt.Errorf("daemon response missing agent payload")
	}
	return *response.Agent, nil
}

func (c *Client) RenameAgent(ctx context.Context, agentID string, displayName string) (core.Agent, error) {
	response, err := c.request(ctx, Request{Command: CommandRenameAgent, AgentID: agentID, DisplayName: displayName})
	if err != nil {
		return core.Agent{}, err
	}
	if response.Agent == nil {
		return core.Agent{}, fmt.Errorf("daemon response missing agent payload")
	}
	return *response.Agent, nil
}

func (c *Client) RemoveAgent(ctx context.Context, agentID string) error {
	_, err := c.request(ctx, Request{
		Command: CommandRemoveAgent,
		AgentID: agentID,
	})
	return err
}

func (c *Client) StopManaged(ctx context.Context, agentID string) error {
	_, err := c.request(ctx, Request{
		Command: CommandStopManaged,
		AgentID: agentID,
	})
	return err
}

func (c *Client) Shutdown(ctx context.Context) error {
	_, err := c.request(ctx, Request{Command: CommandShutdown})
	return err
}

func (c *Client) HookToolStart(ctx context.Context, agentID string, sessionID string, sessionRef string, toolName string, toolInputPreview string, omcMode string) error {
	_, err := c.request(ctx, Request{Command: CommandHookToolStart, AgentID: agentID, SessionID: sessionID, SessionRef: sessionRef, ToolName: toolName, ToolInputPreview: toolInputPreview, OmcMode: omcMode})
	return err
}

func (c *Client) HookToolDone(ctx context.Context, agentID string, sessionID string, sessionRef string, toolName string, toolInputPreview string, omcMode string) error {
	_, err := c.request(ctx, Request{Command: CommandHookToolDone, AgentID: agentID, SessionID: sessionID, SessionRef: sessionRef, ToolName: toolName, ToolInputPreview: toolInputPreview, OmcMode: omcMode})
	return err
}

func (c *Client) HookNotification(ctx context.Context, agentID string, sessionID string, sessionRef string, notificationType string, omcMode string) error {
	_, err := c.request(ctx, Request{Command: CommandHookNotification, AgentID: agentID, SessionID: sessionID, SessionRef: sessionRef, NotificationType: notificationType, OmcMode: omcMode})
	return err
}

func (c *Client) HookStopFailure(ctx context.Context, agentID string, sessionID string, sessionRef string, errorType string, omcMode string) error {
	_, err := c.request(ctx, Request{Command: CommandHookStopFailure, AgentID: agentID, SessionID: sessionID, SessionRef: sessionRef, ErrorType: errorType, OmcMode: omcMode})
	return err
}

func (c *Client) HookSessionStart(ctx context.Context, agentID string, sessionID string, sessionRef string, projectPath string, omcMode string) error {
	_, err := c.request(ctx, Request{Command: CommandHookSessionStart, AgentID: agentID, SessionID: sessionID, SessionRef: sessionRef, ProjectPath: projectPath, OmcMode: omcMode})
	return err
}

func (c *Client) HookSessionEnd(ctx context.Context, agentID string, sessionID string, sessionRef string, omcMode string) error {
	_, err := c.request(ctx, Request{Command: CommandHookSessionEnd, AgentID: agentID, SessionID: sessionID, SessionRef: sessionRef, OmcMode: omcMode})
	return err
}

func (c *Client) HookAgentSpawned(ctx context.Context, agentID string, sessionID string, sessionRef string, description string, omcMode string) error {
	_, err := c.request(ctx, Request{Command: CommandHookAgentSpawned, AgentID: agentID, SessionID: sessionID, SessionRef: sessionRef, Description: description, OmcMode: omcMode})
	return err
}

func (c *Client) HookAgentFinished(ctx context.Context, agentID string, sessionID string, sessionRef string, description string, lastMessage string, omcMode string) error {
	_, err := c.request(ctx, Request{Command: CommandHookAgentFinished, AgentID: agentID, SessionID: sessionID, SessionRef: sessionRef, Description: description, LastMessage: lastMessage, OmcMode: omcMode})
	return err
}

func (c *Client) HookStop(ctx context.Context, agentID string, sessionID string, sessionRef string, lastMessage string, omcMode string) error {
	_, err := c.request(ctx, Request{Command: CommandHookStop, AgentID: agentID, SessionID: sessionID, SessionRef: sessionRef, LastMessage: lastMessage, OmcMode: omcMode})
	return err
}

func (c *Client) HookTeammateIdle(ctx context.Context, agentID string, sessionID string, sessionRef string, teammateName string, teamRole string, omcMode string) error {
	_, err := c.request(ctx, Request{Command: CommandHookTeammateIdle, AgentID: agentID, SessionID: sessionID, SessionRef: sessionRef, TeammateName: teammateName, TeamRole: teamRole, OmcMode: omcMode})
	return err
}

func (c *Client) HookTaskCreated(ctx context.Context, agentID string, sessionID string, sessionRef string, taskName string, taskDescription string, omcMode string) error {
	_, err := c.request(ctx, Request{Command: CommandHookTaskCreated, AgentID: agentID, SessionID: sessionID, SessionRef: sessionRef, TaskName: taskName, TaskDescription: taskDescription, OmcMode: omcMode})
	return err
}

func (c *Client) HookTaskCompleted(ctx context.Context, agentID string, sessionID string, sessionRef string, taskName string, omcMode string) error {
	_, err := c.request(ctx, Request{Command: CommandHookTaskCompleted, AgentID: agentID, SessionID: sessionID, SessionRef: sessionRef, TaskName: taskName, OmcMode: omcMode})
	return err
}

func (c *Client) HookToolFailed(ctx context.Context, agentID string, sessionID string, sessionRef string, toolName string, errorMsg string, isInterrupt bool, omcMode string) error {
	_, err := c.request(ctx, Request{Command: CommandHookToolFailed, AgentID: agentID, SessionID: sessionID, SessionRef: sessionRef, ToolName: toolName, Description: errorMsg, IsInterrupt: isInterrupt, OmcMode: omcMode})
	return err
}

func (c *Client) HookUserPrompt(ctx context.Context, agentID string, sessionID string, sessionRef string, prompt string, omcMode string) error {
	_, err := c.request(ctx, Request{Command: CommandHookUserPrompt, AgentID: agentID, SessionID: sessionID, SessionRef: sessionRef, Prompt: prompt, OmcMode: omcMode})
	return err
}

func (c *Client) HookPermissionRequest(ctx context.Context, agentID string, sessionID string, sessionRef string, toolName string, omcMode string) error {
	_, err := c.request(ctx, Request{Command: CommandHookPermissionReq, AgentID: agentID, SessionID: sessionID, SessionRef: sessionRef, ToolName: toolName, OmcMode: omcMode})
	return err
}

func (c *Client) HookPermissionDenied(ctx context.Context, agentID string, sessionID string, sessionRef string, toolName string, reason string, omcMode string) error {
	_, err := c.request(ctx, Request{Command: CommandHookPermissionDenied, AgentID: agentID, SessionID: sessionID, SessionRef: sessionRef, ToolName: toolName, Description: reason, OmcMode: omcMode})
	return err
}

func (c *Client) HookPreCompact(ctx context.Context, agentID string, sessionID string, sessionRef string, trigger string, omcMode string) error {
	_, err := c.request(ctx, Request{Command: CommandHookPreCompact, AgentID: agentID, SessionID: sessionID, SessionRef: sessionRef, CompactTrigger: trigger, OmcMode: omcMode})
	return err
}

func (c *Client) HookPostCompact(ctx context.Context, agentID string, sessionID string, sessionRef string, trigger string, compactSummary string, omcMode string) error {
	_, err := c.request(ctx, Request{Command: CommandHookPostCompact, AgentID: agentID, SessionID: sessionID, SessionRef: sessionRef, CompactTrigger: trigger, CompactSummary: compactSummary, OmcMode: omcMode})
	return err
}

func (c *Client) HookSetup(ctx context.Context, agentID string, sessionID string, sessionRef string, omcMode string) error {
	_, err := c.request(ctx, Request{Command: CommandHookSetup, AgentID: agentID, SessionID: sessionID, SessionRef: sessionRef, OmcMode: omcMode})
	return err
}

func (c *Client) HookElicitation(ctx context.Context, agentID string, sessionID string, sessionRef string, omcMode string) error {
	_, err := c.request(ctx, Request{Command: CommandHookElicitation, AgentID: agentID, SessionID: sessionID, SessionRef: sessionRef, OmcMode: omcMode})
	return err
}

func (c *Client) HookElicitationResult(ctx context.Context, agentID string, sessionID string, sessionRef string, omcMode string) error {
	_, err := c.request(ctx, Request{Command: CommandHookElicitationResult, AgentID: agentID, SessionID: sessionID, SessionRef: sessionRef, OmcMode: omcMode})
	return err
}

func (c *Client) HookConfigChange(ctx context.Context, agentID string, sessionID string, sessionRef string, source string, omcMode string) error {
	_, err := c.request(ctx, Request{Command: CommandHookConfigChange, AgentID: agentID, SessionID: sessionID, SessionRef: sessionRef, Description: source, OmcMode: omcMode})
	return err
}

func (c *Client) HookWorktreeCreate(ctx context.Context, agentID string, sessionID string, sessionRef string, name string, omcMode string) error {
	_, err := c.request(ctx, Request{Command: CommandHookWorktreeCreate, AgentID: agentID, SessionID: sessionID, SessionRef: sessionRef, WorktreeName: name, OmcMode: omcMode})
	return err
}

func (c *Client) HookWorktreeRemove(ctx context.Context, agentID string, sessionID string, sessionRef string, worktreePath string, omcMode string) error {
	_, err := c.request(ctx, Request{Command: CommandHookWorktreeRemove, AgentID: agentID, SessionID: sessionID, SessionRef: sessionRef, WorktreePath: worktreePath, OmcMode: omcMode})
	return err
}

func (c *Client) HookInstructionsLoaded(ctx context.Context, agentID string, sessionID string, sessionRef string, filePath string, omcMode string) error {
	_, err := c.request(ctx, Request{Command: CommandHookInstructions, AgentID: agentID, SessionID: sessionID, SessionRef: sessionRef, FilePath: filePath, OmcMode: omcMode})
	return err
}

func (c *Client) HookCwdChanged(ctx context.Context, agentID string, sessionID string, sessionRef string, oldCwd string, newCwd string, omcMode string) error {
	_, err := c.request(ctx, Request{Command: CommandHookCwdChanged, AgentID: agentID, SessionID: sessionID, SessionRef: sessionRef, OldCwd: oldCwd, NewCwd: newCwd, OmcMode: omcMode})
	return err
}

func (c *Client) HookFileChanged(ctx context.Context, agentID string, sessionID string, sessionRef string, filePath string, fileEvent string, omcMode string) error {
	_, err := c.request(ctx, Request{Command: CommandHookFileChanged, AgentID: agentID, SessionID: sessionID, SessionRef: sessionRef, FilePath: filePath, FileEvent: fileEvent, OmcMode: omcMode})
	return err
}

func (c *Client) request(ctx context.Context, request Request) (Response, error) {
	dialer := net.Dialer{Timeout: c.timeout}
	connection, err := dialer.DialContext(ctx, "unix", c.socketPath)
	if err != nil {
		return Response{}, fmt.Errorf("connect to hamd: %w", err)
	}
	defer connection.Close()

	if deadline, ok := ctx.Deadline(); ok {
		_ = connection.SetDeadline(deadline)
	} else {
		_ = connection.SetDeadline(time.Now().Add(c.timeout))
	}

	if err := json.NewEncoder(connection).Encode(request); err != nil {
		return Response{}, fmt.Errorf("send request: %w", err)
	}

	var response Response
	if err := json.NewDecoder(connection).Decode(&response); err != nil {
		return Response{}, fmt.Errorf("decode response: %w", err)
	}
	if response.Error != "" {
		return Response{}, fmt.Errorf(response.Error)
	}

	return response, nil
}
