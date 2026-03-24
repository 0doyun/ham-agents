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

func TestRegisterAttachedPersistsModeAndConfidence(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	registry := runtime.NewRegistry(
		store.NewFileAgentStore(filepath.Join(root, "managed-agents.json")),
		store.NewFileEventStore(filepath.Join(root, "events.jsonl")),
	)

	agent, err := registry.RegisterAttached(ctx, runtime.RegisterAttachedInput{
		Provider:    "iterm2",
		DisplayName: "ops",
		ProjectPath: "/tmp/project",
		Role:        "reviewer",
		SessionRef:  "iterm2://session/abc",
	})
	if err != nil {
		t.Fatalf("register attached: %v", err)
	}

	if agent.Mode != core.AgentModeAttached {
		t.Fatalf("expected attached mode, got %q", agent.Mode)
	}
	if agent.StatusConfidence != 0.6 {
		t.Fatalf("expected 0.6 confidence, got %v", agent.StatusConfidence)
	}
	if agent.SessionRef != "iterm2://session/abc" {
		t.Fatalf("unexpected session ref %q", agent.SessionRef)
	}
}

func TestRegisterObservedPersistsModeAndConfidence(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	registry := runtime.NewRegistry(
		store.NewFileAgentStore(filepath.Join(root, "managed-agents.json")),
		store.NewFileEventStore(filepath.Join(root, "events.jsonl")),
	)

	agent, err := registry.RegisterObserved(ctx, runtime.RegisterObservedInput{
		Provider:    "log",
		DisplayName: "observer",
		ProjectPath: "/tmp/project",
		Role:        "watcher",
		SessionRef:  "/tmp/project/transcript.log",
	})
	if err != nil {
		t.Fatalf("register observed: %v", err)
	}

	if agent.Mode != core.AgentModeObserved {
		t.Fatalf("expected observed mode, got %q", agent.Mode)
	}
	if agent.StatusConfidence != 0.35 {
		t.Fatalf("expected 0.35 confidence, got %v", agent.StatusConfidence)
	}
	if agent.SessionRef != "/tmp/project/transcript.log" {
		t.Fatalf("unexpected source ref %q", agent.SessionRef)
	}
}

func TestListRefreshesObservedAgentFromSource(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	source := filepath.Join(root, "observed.log")
	if err := os.WriteFile(source, []byte("build completed"), 0o644); err != nil {
		t.Fatalf("write observed source: %v", err)
	}

	registry := runtime.NewRegistry(
		store.NewFileAgentStore(filepath.Join(root, "managed-agents.json")),
		store.NewFileEventStore(filepath.Join(root, "events.jsonl")),
	)

	agent, err := registry.RegisterObserved(ctx, runtime.RegisterObservedInput{
		Provider:    "log",
		DisplayName: "observer",
		ProjectPath: "/tmp/project",
		SessionRef:  source,
	})
	if err != nil {
		t.Fatalf("register observed: %v", err)
	}
	if agent.Status != core.AgentStatusIdle {
		t.Fatalf("expected initial idle status, got %q", agent.Status)
	}

	listed, err := registry.List(ctx)
	if err != nil {
		t.Fatalf("list agents: %v", err)
	}
	if listed[0].Status != core.AgentStatusDone {
		t.Fatalf("expected observed refresh to infer done, got %q", listed[0].Status)
	}
}

func TestRefreshObservedUpdatesPersistedStatus(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	source := filepath.Join(root, "observed.log")
	if err := os.WriteFile(source, []byte("question?"), 0o644); err != nil {
		t.Fatalf("write observed source: %v", err)
	}

	registry := runtime.NewRegistry(
		store.NewFileAgentStore(filepath.Join(root, "managed-agents.json")),
		store.NewFileEventStore(filepath.Join(root, "events.jsonl")),
	)

	agent, err := registry.RegisterObserved(ctx, runtime.RegisterObservedInput{
		Provider:    "log",
		DisplayName: "observer",
		ProjectPath: "/tmp/project",
		SessionRef:  source,
	})
	if err != nil {
		t.Fatalf("register observed: %v", err)
	}
	if agent.Status != core.AgentStatusIdle {
		t.Fatalf("expected initial idle status, got %q", agent.Status)
	}

	if err := registry.RefreshObserved(ctx); err != nil {
		t.Fatalf("refresh observed: %v", err)
	}

	listed, err := registry.List(ctx)
	if err != nil {
		t.Fatalf("list agents: %v", err)
	}
	if listed[0].Status != core.AgentStatusWaitingInput {
		t.Fatalf("expected waiting_input status, got %q", listed[0].Status)
	}
}

func TestOpenTargetPrefersSessionRefURL(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	registry := runtime.NewRegistry(
		store.NewFileAgentStore(filepath.Join(root, "managed-agents.json")),
		store.NewFileEventStore(filepath.Join(root, "events.jsonl")),
	)

	agent, err := registry.RegisterAttached(ctx, runtime.RegisterAttachedInput{
		Provider:    "iterm2",
		DisplayName: "ops",
		ProjectPath: "/tmp/project",
		SessionRef:  "iterm2://session/abc",
	})
	if err != nil {
		t.Fatalf("register attached: %v", err)
	}

	target, err := registry.OpenTarget(ctx, agent.ID)
	if err != nil {
		t.Fatalf("open target: %v", err)
	}
	if target.Kind != core.OpenTargetKindItermSession {
		t.Fatalf("expected iterm_session, got %q", target.Kind)
	}
	if target.Value != "iterm2://session/abc" {
		t.Fatalf("unexpected target value %q", target.Value)
	}
	if target.SessionID != "abc" {
		t.Fatalf("unexpected session id %q", target.SessionID)
	}
}

func TestRefreshAttachedMarksMissingSessionsDisconnected(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	registry := runtime.NewRegistry(
		store.NewFileAgentStore(filepath.Join(root, "managed-agents.json")),
		store.NewFileEventStore(filepath.Join(root, "events.jsonl")),
	)

	agent, err := registry.RegisterAttached(ctx, runtime.RegisterAttachedInput{
		Provider:    "iterm2",
		DisplayName: "ops",
		ProjectPath: "/tmp/project",
		SessionRef:  "iterm2://session/abc",
	})
	if err != nil {
		t.Fatalf("register attached: %v", err)
	}

	if err := registry.RefreshAttached(ctx, []core.AttachableSession{}); err != nil {
		t.Fatalf("refresh attached: %v", err)
	}

	listed, err := registry.List(ctx)
	if err != nil {
		t.Fatalf("list agents: %v", err)
	}
	if listed[0].ID != agent.ID {
		t.Fatalf("expected agent %q, got %q", agent.ID, listed[0].ID)
	}
	if listed[0].Status != core.AgentStatusDisconnected {
		t.Fatalf("expected disconnected status, got %q", listed[0].Status)
	}
}

func TestRefreshAttachedRestoresDisconnectedSessionsWhenReachable(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	registry := runtime.NewRegistry(
		store.NewFileAgentStore(filepath.Join(root, "managed-agents.json")),
		store.NewFileEventStore(filepath.Join(root, "events.jsonl")),
	)

	agent, err := registry.RegisterAttached(ctx, runtime.RegisterAttachedInput{
		Provider:    "iterm2",
		DisplayName: "ops",
		ProjectPath: "/tmp/project",
		SessionRef:  "iterm2://session/abc",
	})
	if err != nil {
		t.Fatalf("register attached: %v", err)
	}

	if err := registry.RefreshAttached(ctx, []core.AttachableSession{}); err != nil {
		t.Fatalf("refresh attached missing session: %v", err)
	}
	if err := registry.RefreshAttached(ctx, []core.AttachableSession{
		{ID: "abc", Title: "Claude", SessionRef: "iterm2://session/abc", IsActive: true},
	}); err != nil {
		t.Fatalf("refresh attached restored session: %v", err)
	}

	listed, err := registry.List(ctx)
	if err != nil {
		t.Fatalf("list agents: %v", err)
	}
	if listed[0].ID != agent.ID {
		t.Fatalf("expected agent %q, got %q", agent.ID, listed[0].ID)
	}
	if listed[0].Status != core.AgentStatusIdle {
		t.Fatalf("expected idle status after restore, got %q", listed[0].Status)
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
