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
	CommandRunManaged Command = "run.managed"
	CommandAttachSession Command = "attach.session"
	CommandObserveSource Command = "observe.source"
	CommandOpenTarget    Command = "agents.open_target"
	CommandListAgents Command = "agents.list"
	CommandStatus     Command = "agents.status"
	CommandEvents     Command = "events.list"
	CommandSetNotificationPolicy Command = "agents.set_notification_policy"
	CommandSetRole               Command = "agents.set_role"
	CommandRemoveAgent           Command = "agents.remove"
)

type Request struct {
	Command     Command `json:"command"`
	AgentID     string  `json:"agent_id,omitempty"`
	Provider    string  `json:"provider,omitempty"`
	DisplayName string  `json:"display_name,omitempty"`
	ProjectPath string  `json:"project_path,omitempty"`
	Role        string  `json:"role,omitempty"`
	SessionRef  string  `json:"session_ref,omitempty"`
	Limit       int     `json:"limit,omitempty"`
	Policy      string  `json:"policy,omitempty"`
}

type Response struct {
	Agent    *core.Agent           `json:"agent,omitempty"`
	Agents   []core.Agent          `json:"agents,omitempty"`
	Events   []core.Event          `json:"events,omitempty"`
	OpenTarget *core.OpenTarget    `json:"open_target,omitempty"`
	Snapshot *core.RuntimeSnapshot `json:"snapshot,omitempty"`
	Error    string                `json:"error,omitempty"`
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
