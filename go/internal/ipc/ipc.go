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
	CommandHookTaskCompleted     Command = "hook.task-completed"
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

func (c *Client) HookAgentFinished(ctx context.Context, agentID string, sessionID string, sessionRef string, description string, omcMode string) error {
	_, err := c.request(ctx, Request{Command: CommandHookAgentFinished, AgentID: agentID, SessionID: sessionID, SessionRef: sessionRef, Description: description, OmcMode: omcMode})
	return err
}

func (c *Client) HookStop(ctx context.Context, agentID string, sessionID string, sessionRef string, omcMode string) error {
	_, err := c.request(ctx, Request{Command: CommandHookStop, AgentID: agentID, SessionID: sessionID, SessionRef: sessionRef, OmcMode: omcMode})
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
