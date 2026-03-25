package runtime_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/core"
	"github.com/ham-agents/ham-agents/go/internal/runtime"
	"github.com/ham-agents/ham-agents/go/internal/store"
)

func TestManagedServiceCapturesOutputAndExit(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	registry := runtime.NewRegistry(
		store.NewFileAgentStore(filepath.Join(root, "managed-agents.json")),
		store.NewFileEventStore(filepath.Join(root, "events.jsonl")),
	)
	service := runtime.NewManagedService(registry)

	t.Setenv("HAM_MANAGED_PROVIDER_TESTPROC_SHELL", "printf 'Need input?\\n'; printf 'finished successfully\\n'; exit 0")

	agent, err := service.Start(ctx, runtime.RegisterManagedInput{
		Provider:    "testproc",
		DisplayName: "runner",
		ProjectPath: root,
	})
	if err != nil {
		t.Fatalf("start managed process: %v", err)
	}
	if agent.SessionProcessID == 0 {
		t.Fatalf("expected managed start to record a pid, got %#v", agent)
	}

	waitForManagedStatus(t, registry, agent.ID, func(agent core.Agent) bool {
		return agent.Status == core.AgentStatusDone
	})

	listed, err := registry.List(ctx)
	if err != nil {
		t.Fatalf("list agents: %v", err)
	}
	if listed[0].Status != core.AgentStatusDone {
		t.Fatalf("expected done status, got %q", listed[0].Status)
	}

	events, err := registry.Events(ctx, 0)
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	var sawStarted, sawOutput, sawExited bool
	for _, event := range events {
		switch event.Type {
		case core.EventTypeAgentProcessStarted:
			sawStarted = true
		case core.EventTypeAgentProcessOutput:
			sawOutput = true
		case core.EventTypeAgentProcessExited:
			sawExited = true
		}
	}
	if !sawStarted || !sawOutput || !sawExited {
		t.Fatalf("expected managed lifecycle events, got %#v", events)
	}
}

func TestManagedServiceStopMarksAgentStopped(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	registry := runtime.NewRegistry(
		store.NewFileAgentStore(filepath.Join(root, "managed-agents.json")),
		store.NewFileEventStore(filepath.Join(root, "events.jsonl")),
	)
	service := runtime.NewManagedService(registry)

	t.Setenv("HAM_MANAGED_PROVIDER_LONGPROC_SHELL", "trap 'exit 0' TERM; while true; do sleep 1; done")

	agent, err := service.Start(ctx, runtime.RegisterManagedInput{
		Provider:    "longproc",
		DisplayName: "runner",
		ProjectPath: root,
	})
	if err != nil {
		t.Fatalf("start managed process: %v", err)
	}

	if err := service.Stop(ctx, agent.ID); err != nil {
		t.Fatalf("stop managed process: %v", err)
	}

	waitForManagedStatus(t, registry, agent.ID, func(agent core.Agent) bool {
		return agent.Status == core.AgentStatusDone && agent.StatusReason == "Managed process stopped."
	})
}

func waitForManagedStatus(t *testing.T, registry *runtime.Registry, agentID string, matches func(core.Agent) bool) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		agents, err := registry.List(context.Background())
		if err != nil {
			t.Fatalf("list agents: %v", err)
		}
		for _, agent := range agents {
			if agent.ID == agentID && matches(agent) {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}

	agents, err := registry.List(context.Background())
	if err != nil {
		t.Fatalf("list agents: %v", err)
	}
	t.Fatalf("managed status predicate did not match before timeout; agents=%#v", agents)
}
