package runtime_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ham-agents/ham-agents/go/internal/core"
	"github.com/ham-agents/ham-agents/go/internal/runtime"
	"github.com/ham-agents/ham-agents/go/internal/store"
)

func TestRegisterManagedPersistsAndBuildsSnapshot(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	statePath := filepath.Join(root, "managed-agents.json")
	eventPath := filepath.Join(root, "events.jsonl")
	registry := runtime.NewRegistry(
		store.NewFileAgentStore(statePath),
		store.NewFileEventStore(eventPath),
	)

	agent, err := registry.RegisterManaged(ctx, runtime.RegisterManagedInput{
		Provider:    "codex",
		DisplayName: "builder",
		ProjectPath: "/tmp/project",
		Role:        "reviewer",
	})
	if err != nil {
		t.Fatalf("register managed: %v", err)
	}

	if agent.DisplayName != "builder" {
		t.Fatalf("unexpected display name %q", agent.DisplayName)
	}
	if agent.Provider != "codex" {
		t.Fatalf("unexpected provider %q", agent.Provider)
	}

	listed, err := registry.List(ctx)
	if err != nil {
		t.Fatalf("list agents: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(listed))
	}

	snapshot, err := registry.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if snapshot.TotalCount() != 1 {
		t.Fatalf("expected total count 1, got %d", snapshot.TotalCount())
	}
	if snapshot.RunningCount() != 1 {
		t.Fatalf("expected running count 1, got %d", snapshot.RunningCount())
	}

	events, err := store.NewFileEventStore(eventPath).Load(ctx)
	if err != nil {
		t.Fatalf("load events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].AgentID != agent.ID {
		t.Fatalf("unexpected event agent id %q", events[0].AgentID)
	}
}

func TestRegisterManagedSucceedsWhenEventLogAppendFails(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	statePath := filepath.Join(root, "managed-agents.json")
	eventPath := filepath.Join(root, "events")
	if err := os.MkdirAll(eventPath, 0o755); err != nil {
		t.Fatalf("create event directory: %v", err)
	}

	registry := runtime.NewRegistry(
		store.NewFileAgentStore(statePath),
		store.NewFileEventStore(eventPath),
	)

	agent, err := registry.RegisterManaged(ctx, runtime.RegisterManagedInput{
		Provider:    "claude",
		DisplayName: "builder",
		ProjectPath: "/tmp/project",
	})
	if err != nil {
		t.Fatalf("register managed with failing event log should succeed: %v", err)
	}
	if agent.ID == "" {
		t.Fatal("expected agent id to be populated")
	}

	listed, err := registry.List(ctx)
	if err != nil {
		t.Fatalf("list agents: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected persisted agent despite event failure, got %d", len(listed))
	}
}

func TestEventsReturnsMostRecentEntriesWithinLimit(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	statePath := filepath.Join(root, "managed-agents.json")
	eventPath := filepath.Join(root, "events.jsonl")
	registry := runtime.NewRegistry(
		store.NewFileAgentStore(statePath),
		store.NewFileEventStore(eventPath),
	)

	for _, name := range []string{"alpha", "beta"} {
		if _, err := registry.RegisterManaged(ctx, runtime.RegisterManagedInput{
			Provider:    "claude",
			DisplayName: name,
			ProjectPath: "/tmp/project",
		}); err != nil {
			t.Fatalf("register managed %s: %v", name, err)
		}
	}

	events, err := registry.Events(ctx, 1)
	if err != nil {
		t.Fatalf("load events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != core.EventTypeAgentRegistered {
		t.Fatalf("unexpected event type %q", events[0].Type)
	}
}

func TestUpdateNotificationPolicyPersistsChange(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	registry := runtime.NewRegistry(
		store.NewFileAgentStore(filepath.Join(root, "managed-agents.json")),
		store.NewFileEventStore(filepath.Join(root, "events.jsonl")),
	)

	agent, err := registry.RegisterManaged(ctx, runtime.RegisterManagedInput{
		Provider:    "claude",
		DisplayName: "builder",
		ProjectPath: "/tmp/project",
	})
	if err != nil {
		t.Fatalf("register managed: %v", err)
	}

	updated, err := registry.UpdateNotificationPolicy(ctx, agent.ID, core.NotificationPolicyMuted)
	if err != nil {
		t.Fatalf("update policy: %v", err)
	}
	if updated.NotificationPolicy != core.NotificationPolicyMuted {
		t.Fatalf("expected muted policy, got %q", updated.NotificationPolicy)
	}

	listed, err := registry.List(ctx)
	if err != nil {
		t.Fatalf("list agents: %v", err)
	}
	if listed[0].NotificationPolicy != core.NotificationPolicyMuted {
		t.Fatalf("expected persisted muted policy, got %q", listed[0].NotificationPolicy)
	}
}

func TestUpdateRolePersistsChange(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	registry := runtime.NewRegistry(
		store.NewFileAgentStore(filepath.Join(root, "managed-agents.json")),
		store.NewFileEventStore(filepath.Join(root, "events.jsonl")),
	)

	agent, err := registry.RegisterManaged(ctx, runtime.RegisterManagedInput{
		Provider:    "claude",
		DisplayName: "builder",
		ProjectPath: "/tmp/project",
	})
	if err != nil {
		t.Fatalf("register managed: %v", err)
	}

	updated, err := registry.UpdateRole(ctx, agent.ID, "reviewer")
	if err != nil {
		t.Fatalf("update role: %v", err)
	}
	if updated.Role != "reviewer" {
		t.Fatalf("expected reviewer role, got %q", updated.Role)
	}

	listed, err := registry.List(ctx)
	if err != nil {
		t.Fatalf("list agents: %v", err)
	}
	if listed[0].Role != "reviewer" {
		t.Fatalf("expected persisted reviewer role, got %q", listed[0].Role)
	}
}

func TestRemoveDeletesAgentFromRegistry(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	registry := runtime.NewRegistry(
		store.NewFileAgentStore(filepath.Join(root, "managed-agents.json")),
		store.NewFileEventStore(filepath.Join(root, "events.jsonl")),
	)

	agent, err := registry.RegisterManaged(ctx, runtime.RegisterManagedInput{
		Provider:    "claude",
		DisplayName: "builder",
		ProjectPath: "/tmp/project",
	})
	if err != nil {
		t.Fatalf("register managed: %v", err)
	}

	if err := registry.Remove(ctx, agent.ID); err != nil {
		t.Fatalf("remove agent: %v", err)
	}

	listed, err := registry.List(ctx)
	if err != nil {
		t.Fatalf("list agents: %v", err)
	}
	if len(listed) != 0 {
		t.Fatalf("expected empty registry, got %d agents", len(listed))
	}
}
