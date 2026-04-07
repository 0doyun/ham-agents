package runtime_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ham-agents/ham-agents/go/internal/core"
	"github.com/ham-agents/ham-agents/go/internal/runtime"
	"github.com/ham-agents/ham-agents/go/internal/store"
)

// newRegistryForHookTest creates a fresh Registry backed by temp-dir file stores
// and registers one managed agent, returning the registry and agent ID.
func newRegistryForHookTest(t *testing.T) (*runtime.Registry, string) {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	reg := runtime.NewRegistry(
		store.NewFileAgentStore(filepath.Join(root, "agents.json")),
		store.NewFileEventStore(filepath.Join(root, "events.jsonl")),
	)
	agent, err := reg.RegisterManaged(ctx, runtime.RegisterManagedInput{
		Provider:    "claude",
		DisplayName: "test-agent",
		ProjectPath: "/tmp/test-project",
		Role:        "worker",
	})
	if err != nil {
		t.Fatalf("register managed agent: %v", err)
	}
	return reg, agent.ID
}

// loadAgent is a test helper that loads and returns the single agent from the registry.
func loadAgent(t *testing.T, reg *runtime.Registry, agentID string) core.Agent {
	t.Helper()
	ctx := context.Background()
	agents, err := reg.List(ctx)
	if err != nil {
		t.Fatalf("list agents: %v", err)
	}
	for _, a := range agents {
		if a.ID == agentID {
			return a
		}
	}
	t.Fatalf("agent %q not found after hook call", agentID)
	return core.Agent{}
}

// TestRegistry_RecordHook_StateTransitions exercises every RecordHook* method
// and asserts the expected AgentStatus after each call.
func TestRegistry_RecordHook_StateTransitions(t *testing.T) {
	t.Parallel()

	type hookCall func(reg *runtime.Registry, agentID string) error

	cases := []struct {
		name           string
		setup          hookCall // optional additional setup before the primary call
		call           hookCall
		expectedStatus core.AgentStatus
	}{
		{
			name: "RecordHookSessionStart -> Booting",
			call: func(reg *runtime.Registry, agentID string) error {
				return reg.RecordHookSessionStart(context.Background(), agentID, "sess-001", "")
			},
			expectedStatus: core.AgentStatusBooting,
		},
		{
			name: "RecordHookToolStart(Bash) -> RunningTool",
			call: func(reg *runtime.Registry, agentID string) error {
				return reg.RecordHookToolStart(context.Background(), agentID, "Bash", "go test ./...", "")
			},
			expectedStatus: core.AgentStatusRunningTool,
		},
		{
			name: "RecordHookToolStart(Read) -> Reading",
			call: func(reg *runtime.Registry, agentID string) error {
				return reg.RecordHookToolStart(context.Background(), agentID, "Read", "/etc/hosts", "")
			},
			expectedStatus: core.AgentStatusReading,
		},
		{
			name: "RecordHookToolStart(Write) -> Writing",
			call: func(reg *runtime.Registry, agentID string) error {
				return reg.RecordHookToolStart(context.Background(), agentID, "Write", "/tmp/out.txt", "")
			},
			expectedStatus: core.AgentStatusWriting,
		},
		{
			name: "RecordHookToolDone -> Thinking",
			setup: func(reg *runtime.Registry, agentID string) error {
				return reg.RecordHookToolStart(context.Background(), agentID, "Bash", "ls", "")
			},
			call: func(reg *runtime.Registry, agentID string) error {
				return reg.RecordHookToolDone(context.Background(), agentID, "Bash", "ls", "")
			},
			expectedStatus: core.AgentStatusThinking,
		},
		{
			name: "RecordHookStop -> Idle",
			call: func(reg *runtime.Registry, agentID string) error {
				return reg.RecordHookStop(context.Background(), agentID, "All done.", "")
			},
			expectedStatus: core.AgentStatusIdle,
		},
		{
			name: "RecordHookStopFailure -> Error",
			call: func(reg *runtime.Registry, agentID string) error {
				return reg.RecordHookStopFailure(context.Background(), agentID, "timeout", "")
			},
			expectedStatus: core.AgentStatusError,
		},
		{
			name: "RecordHookPermissionRequest -> WaitingInput",
			call: func(reg *runtime.Registry, agentID string) error {
				return reg.RecordHookPermissionRequest(context.Background(), agentID, "Bash", "")
			},
			expectedStatus: core.AgentStatusWaitingInput,
		},
		{
			name: "RecordHookElicitation -> WaitingInput",
			call: func(reg *runtime.Registry, agentID string) error {
				return reg.RecordHookElicitation(context.Background(), agentID, "")
			},
			expectedStatus: core.AgentStatusWaitingInput,
		},
		{
			name: "RecordHookElicitationResult -> Thinking",
			setup: func(reg *runtime.Registry, agentID string) error {
				return reg.RecordHookElicitation(context.Background(), agentID, "")
			},
			call: func(reg *runtime.Registry, agentID string) error {
				return reg.RecordHookElicitationResult(context.Background(), agentID, "")
			},
			expectedStatus: core.AgentStatusThinking,
		},
		{
			name: "RecordHookPostCompact -> Thinking",
			setup: func(reg *runtime.Registry, agentID string) error {
				return reg.RecordHookPreCompact(context.Background(), agentID, "auto", "")
			},
			call: func(reg *runtime.Registry, agentID string) error {
				return reg.RecordHookPostCompact(context.Background(), agentID, "auto", "Summarized 100 messages.", "")
			},
			expectedStatus: core.AgentStatusThinking,
		},
		{
			name: "RecordHookUserPrompt -> Thinking",
			call: func(reg *runtime.Registry, agentID string) error {
				return reg.RecordHookUserPrompt(context.Background(), agentID, "What is 2+2?", "")
			},
			expectedStatus: core.AgentStatusThinking,
		},
		{
			// RecordHookToolFailed with isInterrupt=true -> WaitingInput
			name: "RecordHookToolFailed(interrupt) -> WaitingInput",
			setup: func(reg *runtime.Registry, agentID string) error {
				return reg.RecordHookToolStart(context.Background(), agentID, "Bash", "sleep 10", "")
			},
			call: func(reg *runtime.Registry, agentID string) error {
				return reg.RecordHookToolFailed(context.Background(), agentID, "Bash", "interrupted", true, "")
			},
			expectedStatus: core.AgentStatusWaitingInput,
		},
		{
			// RecordHookToolFailed with isInterrupt=false -> Thinking
			name: "RecordHookToolFailed(non-interrupt) -> Thinking",
			setup: func(reg *runtime.Registry, agentID string) error {
				return reg.RecordHookToolStart(context.Background(), agentID, "Bash", "bad-cmd", "")
			},
			call: func(reg *runtime.Registry, agentID string) error {
				return reg.RecordHookToolFailed(context.Background(), agentID, "Bash", "exit status 1", false, "")
			},
			expectedStatus: core.AgentStatusThinking,
		},
		{
			name: "RecordHookAgentSpawned does not change agent status (stays as registered)",
			call: func(reg *runtime.Registry, agentID string) error {
				return reg.RecordHookAgentSpawned(context.Background(), agentID, "sub-executor", "")
			},
			// AgentSpawned does not mutate Status; the agent remains in whatever
			// status it had at registration (Booting for a newly registered managed agent).
			expectedStatus: core.AgentStatusBooting,
		},
		{
			name: "RecordHookNotification(permission_prompt) -> WaitingInput",
			call: func(reg *runtime.Registry, agentID string) error {
				return reg.RecordHookNotification(context.Background(), agentID, "permission_prompt", "")
			},
			expectedStatus: core.AgentStatusWaitingInput,
		},
		{
			name: "RecordHookNotification(idle_prompt) -> Idle",
			call: func(reg *runtime.Registry, agentID string) error {
				return reg.RecordHookNotification(context.Background(), agentID, "idle_prompt", "")
			},
			expectedStatus: core.AgentStatusIdle,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reg, agentID := newRegistryForHookTest(t)

			if tc.setup != nil {
				if err := tc.setup(reg, agentID); err != nil {
					t.Fatalf("setup call failed: %v", err)
				}
			}

			if err := tc.call(reg, agentID); err != nil {
				t.Fatalf("hook call failed: %v", err)
			}

			agent := loadAgent(t, reg, agentID)
			if agent.Status != tc.expectedStatus {
				t.Errorf("expected status %q, got %q (reason: %q)", tc.expectedStatus, agent.Status, agent.StatusReason)
			}
		})
	}
}
