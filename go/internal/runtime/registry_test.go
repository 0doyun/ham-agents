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

	events, err := registry.Events(ctx, 0)
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	if events[len(events)-1].Type != core.EventTypeAgentStatusUpdated {
		t.Fatalf("expected observed status event from list refresh, got %q", events[len(events)-1].Type)
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

	events, err := registry.Events(ctx, 0)
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	if events[len(events)-1].Type != core.EventTypeAgentStatusUpdated {
		t.Fatalf("expected observed status event, got %q", events[len(events)-1].Type)
	}
	if events[len(events)-1].Summary != "Status changed to waiting_input. Question-like output detected." {
		t.Fatalf("unexpected observed status summary %q", events[len(events)-1].Summary)
	}
}

func TestSnapshotRefreshesObservedAgentAndPersistsLifecycleEvent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	source := filepath.Join(root, "observed.log")
	if err := os.WriteFile(source, []byte("task failed with error"), 0o644); err != nil {
		t.Fatalf("write observed source: %v", err)
	}

	registry := runtime.NewRegistry(
		store.NewFileAgentStore(filepath.Join(root, "managed-agents.json")),
		store.NewFileEventStore(filepath.Join(root, "events.jsonl")),
	)

	_, err := registry.RegisterObserved(ctx, runtime.RegisterObservedInput{
		Provider:    "log",
		DisplayName: "observer",
		ProjectPath: "/tmp/project",
		SessionRef:  source,
	})
	if err != nil {
		t.Fatalf("register observed: %v", err)
	}

	snapshot, err := registry.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if snapshot.Agents[0].Status != core.AgentStatusError {
		t.Fatalf("expected error status, got %q", snapshot.Agents[0].Status)
	}

	events, err := registry.Events(ctx, 0)
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	if events[len(events)-1].Type != core.EventTypeAgentStatusUpdated {
		t.Fatalf("expected observed status event from snapshot refresh, got %q", events[len(events)-1].Type)
	}
	if events[len(events)-1].Summary != "Status changed to error. Error-like output detected." {
		t.Fatalf("unexpected snapshot-driven status summary %q", events[len(events)-1].Summary)
	}
}

func TestRefreshObservedDoesNotEmitLifecycleEventWhenStatusStaysSame(t *testing.T) {
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

	if _, err := registry.RegisterObserved(ctx, runtime.RegisterObservedInput{
		Provider:    "log",
		DisplayName: "observer",
		ProjectPath: "/tmp/project",
		SessionRef:  source,
	}); err != nil {
		t.Fatalf("register observed: %v", err)
	}

	if err := registry.RefreshObserved(ctx); err != nil {
		t.Fatalf("first refresh observed: %v", err)
	}
	eventsAfterFirstRefresh, err := registry.Events(ctx, 0)
	if err != nil {
		t.Fatalf("events after first refresh: %v", err)
	}

	if err := registry.RefreshObserved(ctx); err != nil {
		t.Fatalf("second refresh observed: %v", err)
	}
	eventsAfterSecondRefresh, err := registry.Events(ctx, 0)
	if err != nil {
		t.Fatalf("events after second refresh: %v", err)
	}

	if len(eventsAfterSecondRefresh) != len(eventsAfterFirstRefresh) {
		t.Fatalf("expected no extra lifecycle event on unchanged observed status, got %d -> %d", len(eventsAfterFirstRefresh), len(eventsAfterSecondRefresh))
	}
}

func TestListObservedRefreshSavesExactlyOncePerObservedTransition(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	source := filepath.Join(root, "observed.log")
	if err := os.WriteFile(source, []byte("question?"), 0o644); err != nil {
		t.Fatalf("write observed source: %v", err)
	}

	countingStore := &countingAgentStore{
		AgentStore: store.NewFileAgentStore(filepath.Join(root, "managed-agents.json")),
	}
	registry := runtime.NewRegistry(
		countingStore,
		store.NewFileEventStore(filepath.Join(root, "events.jsonl")),
	)

	if _, err := registry.RegisterObserved(ctx, runtime.RegisterObservedInput{
		Provider:    "log",
		DisplayName: "observer",
		ProjectPath: "/tmp/project",
		SessionRef:  source,
	}); err != nil {
		t.Fatalf("register observed: %v", err)
	}

	beforeListSaves := countingStore.saveCalls
	if _, err := registry.List(ctx); err != nil {
		t.Fatalf("list agents: %v", err)
	}

	if countingStore.saveCalls-beforeListSaves != 1 {
		t.Fatalf("expected exactly one save during list-driven observed refresh, got %d", countingStore.saveCalls-beforeListSaves)
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

type countingAgentStore struct {
	store.AgentStore
	saveCalls int
}

func (s *countingAgentStore) SaveAgents(ctx context.Context, agents []core.Agent) error {
	s.saveCalls++
	return s.AgentStore.SaveAgents(ctx, agents)
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
	if listed[0].SessionTTY != "" || listed[0].SessionWorkingDirectory != "" || listed[0].SessionActivity != "" {
		t.Fatalf("expected stale shell-state metadata to be cleared, got tty=%q cwd=%q activity=%q", listed[0].SessionTTY, listed[0].SessionWorkingDirectory, listed[0].SessionActivity)
	}
	if listed[0].SessionProcessID != 0 || listed[0].SessionCommand != "" {
		t.Fatalf("expected stale process metadata to be cleared, got pid=%d command=%q", listed[0].SessionProcessID, listed[0].SessionCommand)
	}

	events, err := registry.Events(ctx, 0)
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	if events[len(events)-1].Type != core.EventTypeAgentDisconnected {
		t.Fatalf("expected disconnected event, got %q", events[len(events)-1].Type)
	}
	if events[len(events)-1].Summary != "Status changed to disconnected. Session missing from iTerm session list." {
		t.Fatalf("unexpected disconnected summary %q", events[len(events)-1].Summary)
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
		{ID: "abc", Title: "Claude", SessionRef: "iterm2://session/abc", IsActive: true, TTY: "ttys001", WorkingDirectory: "/tmp/project", Activity: "claude", ProcessID: 12345, Command: "/usr/local/bin/claude"},
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
	if listed[0].SessionTitle != "Claude" {
		t.Fatalf("expected synced session title, got %q", listed[0].SessionTitle)
	}
	if !listed[0].SessionIsActive {
		t.Fatal("expected restored session to be marked active")
	}
	if listed[0].SessionTTY != "ttys001" {
		t.Fatalf("expected synced tty, got %q", listed[0].SessionTTY)
	}
	if listed[0].SessionWorkingDirectory != "/tmp/project" {
		t.Fatalf("expected synced working directory, got %q", listed[0].SessionWorkingDirectory)
	}
	if listed[0].SessionActivity != "claude" {
		t.Fatalf("expected synced activity, got %q", listed[0].SessionActivity)
	}
	if listed[0].SessionProcessID != 12345 {
		t.Fatalf("expected synced process id, got %d", listed[0].SessionProcessID)
	}
	if listed[0].SessionCommand != "/usr/local/bin/claude" {
		t.Fatalf("expected synced command, got %q", listed[0].SessionCommand)
	}

	events, err := registry.Events(ctx, 0)
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	if events[len(events)-1].Type != core.EventTypeAgentReconnected {
		t.Fatalf("expected reconnected event, got %q", events[len(events)-1].Type)
	}
	if events[len(events)-1].Summary != "Status changed to idle. Session reachable in iTerm again." {
		t.Fatalf("unexpected reconnected summary %q", events[len(events)-1].Summary)
	}
}

func TestRefreshAttachedSyncsMetadataWithoutDisconnect(t *testing.T) {
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

	if err := registry.RefreshAttached(ctx, []core.AttachableSession{
		{ID: "abc", Title: "Claude Review", SessionRef: "iterm2://session/abc", IsActive: true, TTY: "ttys001", WorkingDirectory: "/tmp/project", Activity: "claude", ProcessID: 12345, Command: "/usr/local/bin/claude"},
	}); err != nil {
		t.Fatalf("refresh attached metadata: %v", err)
	}

	listed, err := registry.List(ctx)
	if err != nil {
		t.Fatalf("list agents: %v", err)
	}
	if listed[0].ID != agent.ID {
		t.Fatalf("expected agent %q, got %q", agent.ID, listed[0].ID)
	}
	if listed[0].SessionTitle != "Claude Review" {
		t.Fatalf("expected session title sync, got %q", listed[0].SessionTitle)
	}
	if !listed[0].SessionIsActive {
		t.Fatal("expected session active marker to sync")
	}
	if listed[0].SessionTTY != "ttys001" {
		t.Fatalf("expected tty sync, got %q", listed[0].SessionTTY)
	}
	if listed[0].SessionWorkingDirectory != "/tmp/project" {
		t.Fatalf("expected working directory sync, got %q", listed[0].SessionWorkingDirectory)
	}
	if listed[0].SessionActivity != "claude" {
		t.Fatalf("expected activity sync, got %q", listed[0].SessionActivity)
	}
	if listed[0].SessionProcessID != 12345 {
		t.Fatalf("expected process id sync, got %d", listed[0].SessionProcessID)
	}
	if listed[0].SessionCommand != "/usr/local/bin/claude" {
		t.Fatalf("expected command sync, got %q", listed[0].SessionCommand)
	}
	if listed[0].Status != core.AgentStatusIdle {
		t.Fatalf("expected idle status, got %q", listed[0].Status)
	}
}

func TestRefreshAttachedDoesNotPersistWhenNothingChanged(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	countingStore := &countingAgentStore{
		AgentStore: store.NewFileAgentStore(filepath.Join(root, "managed-agents.json")),
	}
	registry := runtime.NewRegistry(
		countingStore,
		store.NewFileEventStore(filepath.Join(root, "events.jsonl")),
	)

	if _, err := registry.RegisterAttached(ctx, runtime.RegisterAttachedInput{
		Provider:    "iterm2",
		DisplayName: "ops",
		ProjectPath: "/tmp/project",
		SessionRef:  "iterm2://session/abc",
	}); err != nil {
		t.Fatalf("register attached: %v", err)
	}

	if err := registry.RefreshAttached(ctx, []core.AttachableSession{
		{ID: "abc", Title: "ops", SessionRef: "iterm2://session/abc"},
	}); err != nil {
		t.Fatalf("initial refresh attached: %v", err)
	}

	before := countingStore.saveCalls
	if err := registry.RefreshAttached(ctx, []core.AttachableSession{
		{ID: "abc", Title: "ops", SessionRef: "iterm2://session/abc"},
	}); err != nil {
		t.Fatalf("second refresh attached: %v", err)
	}
	if countingStore.saveCalls != before {
		t.Fatalf("expected no extra save when nothing changed, got %d -> %d", before, countingStore.saveCalls)
	}

	events, err := registry.Events(ctx, 0)
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected no additional lifecycle events, got %d", len(events))
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

func TestFollowEventsReturnsEntriesAfterCursor(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	registry := runtime.NewRegistry(
		store.NewFileAgentStore(filepath.Join(root, "managed-agents.json")),
		store.NewFileEventStore(filepath.Join(root, "events.jsonl")),
	)

	if _, err := registry.RegisterManaged(ctx, runtime.RegisterManagedInput{
		Provider:    "claude",
		DisplayName: "alpha",
		ProjectPath: "/tmp/project",
	}); err != nil {
		t.Fatalf("register managed alpha: %v", err)
	}

	initialEvents, err := registry.Events(ctx, 10)
	if err != nil {
		t.Fatalf("load initial events: %v", err)
	}

	if _, err := registry.RegisterManaged(ctx, runtime.RegisterManagedInput{
		Provider:    "claude",
		DisplayName: "beta",
		ProjectPath: "/tmp/project",
	}); err != nil {
		t.Fatalf("register managed beta: %v", err)
	}

	followed, err := registry.FollowEvents(ctx, initialEvents[len(initialEvents)-1].ID, 10, 0)
	if err != nil {
		t.Fatalf("follow events: %v", err)
	}
	if len(followed) != 1 {
		t.Fatalf("expected 1 followed event, got %d", len(followed))
	}
	if followed[0].AgentID == initialEvents[len(initialEvents)-1].AgentID {
		t.Fatalf("expected only newer event, got %#v", followed[0])
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

	events, err := registry.Events(ctx, 0)
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	if events[len(events)-1].Type != core.EventTypeAgentNotificationPolicyUpdated {
		t.Fatalf("expected notification policy event, got %q", events[len(events)-1].Type)
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

	events, err := registry.Events(ctx, 0)
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	if events[len(events)-1].Type != core.EventTypeAgentRoleUpdated {
		t.Fatalf("expected role updated event, got %q", events[len(events)-1].Type)
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

	events, err := registry.Events(ctx, 0)
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	if events[len(events)-1].Type != core.EventTypeAgentRemoved {
		t.Fatalf("expected agent removed event, got %q", events[len(events)-1].Type)
	}
}
