package runtime_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	if snapshot.AttentionCount != 0 {
		t.Fatalf("expected attention count 0, got %d", snapshot.AttentionCount)
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
	if events[0].PresentationLabel != "Managed" || events[0].PresentationEmphasis != "info" {
		t.Fatalf("unexpected event presentation %#v", events[0])
	}
	if events[0].PresentationSummary != "Managed session registered." {
		t.Fatalf("unexpected event presentation summary %#v", events[0])
	}
	if events[0].LifecycleStatus != "booting" || events[0].LifecycleMode != "managed" {
		t.Fatalf("unexpected event lifecycle metadata %#v", events[0])
	}
	if events[0].LifecycleReason != "Managed launch requested." {
		t.Fatalf("unexpected event lifecycle reason %#v", events[0])
	}
	if events[0].LifecycleConfidence != 1 {
		t.Fatalf("unexpected event lifecycle confidence %#v", events[0])
	}
}

func TestSnapshotBuildsAttentionSummary(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	registry := runtime.NewRegistry(
		store.NewFileAgentStore(filepath.Join(root, "managed-agents.json")),
		store.NewFileEventStore(filepath.Join(root, "events.jsonl")),
	)

	_, err := registry.RegisterManaged(ctx, runtime.RegisterManagedInput{
		Provider:    "codex",
		DisplayName: "erroring",
		ProjectPath: "/tmp/project",
		Role:        "reviewer",
	})
	if err != nil {
		t.Fatalf("register erroring agent: %v", err)
	}
	_, err = registry.RegisterManaged(ctx, runtime.RegisterManagedInput{
		Provider:    "codex",
		DisplayName: "waiting",
		ProjectPath: "/tmp/project",
		Role:        "reviewer",
	})
	if err != nil {
		t.Fatalf("register waiting agent: %v", err)
	}

	agents, err := registry.List(ctx)
	if err != nil {
		t.Fatalf("list agents: %v", err)
	}
	agents[0].Status = core.AgentStatusError
	agents[1].Status = core.AgentStatusWaitingInput
	if err := store.NewFileAgentStore(filepath.Join(root, "managed-agents.json")).SaveAgents(ctx, agents); err != nil {
		t.Fatalf("save agents: %v", err)
	}

	snapshot, err := registry.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	if snapshot.AttentionCount != 2 {
		t.Fatalf("expected attention count 2, got %d", snapshot.AttentionCount)
	}
	if snapshot.AttentionBreakdown.Error != 1 || snapshot.AttentionBreakdown.WaitingInput != 1 || snapshot.AttentionBreakdown.Disconnected != 0 {
		t.Fatalf("unexpected attention breakdown %#v", snapshot.AttentionBreakdown)
	}
	if len(snapshot.AttentionOrder) != 2 || snapshot.AttentionOrder[0] != agents[0].ID || snapshot.AttentionOrder[1] != agents[1].ID {
		t.Fatalf("unexpected attention order %#v", snapshot.AttentionOrder)
	}
	if snapshot.AttentionSubtitles[agents[0].ID] == "" || snapshot.AttentionSubtitles[agents[1].ID] == "" {
		t.Fatalf("expected attention subtitles, got %#v", snapshot.AttentionSubtitles)
	}
}

func TestSnapshotAttentionOrderUsesSeverityThenRecency(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	statePath := filepath.Join(root, "managed-agents.json")
	eventPath := filepath.Join(root, "events.jsonl")
	agentStore := store.NewFileAgentStore(statePath)
	registry := runtime.NewRegistry(agentStore, store.NewFileEventStore(eventPath))

	if err := agentStore.SaveAgents(ctx, []core.Agent{
		{
			ID:          "agent-1",
			DisplayName: "waiting-older",
			Status:      core.AgentStatusWaitingInput,
			LastEventAt: time.Unix(1, 0).UTC(),
		},
		{
			ID:          "agent-2",
			DisplayName: "error",
			Status:      core.AgentStatusError,
			LastEventAt: time.Unix(2, 0).UTC(),
		},
		{
			ID:          "agent-3",
			DisplayName: "waiting-newer",
			Status:      core.AgentStatusWaitingInput,
			LastEventAt: time.Unix(3, 0).UTC(),
		},
		{
			ID:          "agent-4",
			DisplayName: "calm",
			Status:      core.AgentStatusThinking,
			LastEventAt: time.Unix(4, 0).UTC(),
		},
	}); err != nil {
		t.Fatalf("save agents: %v", err)
	}

	snapshot, err := registry.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	if got, want := snapshot.AttentionOrder, []string{"agent-2", "agent-3", "agent-1"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("unexpected attention order %#v", got)
	}
}

func TestSnapshotAttentionSubtitleUsesConfidenceAndReason(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	statePath := filepath.Join(root, "managed-agents.json")
	eventPath := filepath.Join(root, "events.jsonl")
	agentStore := store.NewFileAgentStore(statePath)
	registry := runtime.NewRegistry(agentStore, store.NewFileEventStore(eventPath))

	if err := agentStore.SaveAgents(ctx, []core.Agent{
		{
			ID:               "agent-1",
			DisplayName:      "waiting",
			Status:           core.AgentStatusWaitingInput,
			StatusConfidence: 0.45,
			StatusReason:     "Needs approval.",
		},
	}); err != nil {
		t.Fatalf("save agents: %v", err)
	}

	snapshot, err := registry.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	if got := snapshot.AttentionSubtitles["agent-1"]; got != "likely needs input · low confidence · Needs approval." {
		t.Fatalf("unexpected subtitle %q", got)
	}
}

func TestStatusUpdatedEventCarriesLifecycleReason(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	statePath := filepath.Join(root, "managed-agents.json")
	eventPath := filepath.Join(root, "events.jsonl")
	registry := runtime.NewRegistry(
		store.NewFileAgentStore(statePath),
		store.NewFileEventStore(eventPath),
	)

	if _, err := registry.RegisterObserved(ctx, runtime.RegisterObservedInput{
		Provider:    "log",
		DisplayName: "watcher",
		ProjectPath: "/tmp/project",
		SessionRef:  filepath.Join(root, "watcher.log"),
	}); err != nil {
		t.Fatalf("register observed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "watcher.log"), []byte("Need help?\n"), 0o644); err != nil {
		t.Fatalf("write observed log: %v", err)
	}

	if err := registry.RefreshObserved(ctx); err != nil {
		t.Fatalf("refresh observed: %v", err)
	}

	events, err := store.NewFileEventStore(eventPath).Load(ctx)
	if err != nil {
		t.Fatalf("load events: %v", err)
	}
	last := events[len(events)-1]
	if last.Type != core.EventTypeAgentStatusUpdated {
		t.Fatalf("expected status updated event, got %q", last.Type)
	}
	if last.LifecycleReason != "Question-like output detected." {
		t.Fatalf("unexpected lifecycle reason %q", last.LifecycleReason)
	}
	if last.LifecycleConfidence != 0.45 {
		t.Fatalf("unexpected lifecycle confidence %v", last.LifecycleConfidence)
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

func TestRegisterAttachedReusesExistingSessionRef(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	registry := runtime.NewRegistry(
		store.NewFileAgentStore(filepath.Join(root, "managed-agents.json")),
		store.NewFileEventStore(filepath.Join(root, "events.jsonl")),
	)

	first, err := registry.RegisterAttached(ctx, runtime.RegisterAttachedInput{
		Provider:    "iterm2",
		DisplayName: "ops",
		ProjectPath: "/tmp/project",
		SessionRef:  "iterm2://session/abc",
	})
	if err != nil {
		t.Fatalf("register attached: %v", err)
	}

	second, err := registry.RegisterAttached(ctx, runtime.RegisterAttachedInput{
		Provider:    "iterm2",
		DisplayName: "ops-reused",
		ProjectPath: "/tmp/project-two",
		SessionRef:  "iterm2://session/abc",
	})
	if err != nil {
		t.Fatalf("register attached reuse: %v", err)
	}

	if second.ID != first.ID {
		t.Fatalf("expected attach reuse to keep same agent id, first=%q second=%q", first.ID, second.ID)
	}
	if second.DisplayName != "ops-reused" || second.ProjectPath != "/tmp/project-two" {
		t.Fatalf("expected metadata to refresh on reuse, got %#v", second)
	}
}

func TestRegisterObservedReusesExistingSourceRef(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	registry := runtime.NewRegistry(
		store.NewFileAgentStore(filepath.Join(root, "managed-agents.json")),
		store.NewFileEventStore(filepath.Join(root, "events.jsonl")),
	)

	first, err := registry.RegisterObserved(ctx, runtime.RegisterObservedInput{
		Provider:    "transcript",
		DisplayName: "watcher",
		ProjectPath: "/tmp/project",
		SessionRef:  filepath.Join(root, "watcher.log"),
	})
	if err != nil {
		t.Fatalf("register observed: %v", err)
	}

	second, err := registry.RegisterObserved(ctx, runtime.RegisterObservedInput{
		Provider:    "transcript",
		DisplayName: "watcher-two",
		ProjectPath: "/tmp/project-two",
		SessionRef:  filepath.Join(root, "watcher.log"),
	})
	if err != nil {
		t.Fatalf("register observed reuse: %v", err)
	}

	if second.ID != first.ID {
		t.Fatalf("expected observed reuse to keep same agent id, first=%q second=%q", first.ID, second.ID)
	}
	if second.DisplayName != "watcher-two" || second.ProjectPath != "/tmp/project-two" {
		t.Fatalf("expected observed metadata to refresh on reuse, got %#v", second)
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
	if events[len(events)-1].Summary != "Status changed to waiting_input. Observed question-like output." {
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
	if events[len(events)-1].Summary != "Status changed to error. Observed error-like output." {
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

func TestOpenTargetRecognizesTmuxSessionRef(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	registry := runtime.NewRegistry(
		store.NewFileAgentStore(filepath.Join(root, "managed-agents.json")),
		store.NewFileEventStore(filepath.Join(root, "events.jsonl")),
	)

	agent, err := registry.RegisterAttached(ctx, runtime.RegisterAttachedInput{
		Provider:    "tmux",
		DisplayName: "ops",
		ProjectPath: "/tmp/project",
		SessionRef:  "tmux://demo:1.0",
	})
	if err != nil {
		t.Fatalf("register attached: %v", err)
	}

	target, err := registry.OpenTarget(ctx, agent.ID)
	if err != nil {
		t.Fatalf("open target: %v", err)
	}
	if target.Kind != core.OpenTargetKindTmuxPane {
		t.Fatalf("expected tmux_pane, got %q", target.Kind)
	}
	if target.Value != "tmux://demo:1.0" {
		t.Fatalf("unexpected target value %q", target.Value)
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

func TestRefreshAttachedInfersRunningToolFromSessionCommand(t *testing.T) {
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
		SessionRef:  "iterm2://session/tool",
	})
	if err != nil {
		t.Fatalf("register attached: %v", err)
	}

	if err := registry.RefreshAttached(ctx, []core.AttachableSession{
		{ID: "tool", Title: "Tool Run", SessionRef: "iterm2://session/tool", IsActive: true, Command: "go test ./...", Activity: "go test"},
	}); err != nil {
		t.Fatalf("refresh attached tool: %v", err)
	}

	listed, err := registry.List(ctx)
	if err != nil {
		t.Fatalf("list agents: %v", err)
	}
	if listed[0].ID != agent.ID {
		t.Fatalf("expected agent %q, got %q", agent.ID, listed[0].ID)
	}
	if listed[0].Status != core.AgentStatusRunningTool {
		t.Fatalf("expected running_tool status, got %q", listed[0].Status)
	}
	if listed[0].StatusReason != "Tool-like attached session activity detected." {
		t.Fatalf("unexpected reason %q", listed[0].StatusReason)
	}

	events, err := registry.Events(ctx, 0)
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	if events[len(events)-1].Type != core.EventTypeAgentStatusUpdated {
		t.Fatalf("expected status updated event, got %q", events[len(events)-1].Type)
	}
	if events[len(events)-1].Summary != "Status changed to running_tool. Tool-like attached session activity detected." {
		t.Fatalf("unexpected status summary %q", events[len(events)-1].Summary)
	}
}

func TestRefreshAttachedInfersReadingFromSessionCommand(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	registry := runtime.NewRegistry(
		store.NewFileAgentStore(filepath.Join(root, "managed-agents.json")),
		store.NewFileEventStore(filepath.Join(root, "events.jsonl")),
	)

	if _, err := registry.RegisterAttached(ctx, runtime.RegisterAttachedInput{
		Provider:    "iterm2",
		DisplayName: "ops",
		ProjectPath: "/tmp/project",
		SessionRef:  "iterm2://session/read",
	}); err != nil {
		t.Fatalf("register attached: %v", err)
	}

	if err := registry.RefreshAttached(ctx, []core.AttachableSession{
		{ID: "read", Title: "Log Read", SessionRef: "iterm2://session/read", IsActive: true, Command: "less /tmp/project/build.log", Activity: "less"},
	}); err != nil {
		t.Fatalf("refresh attached reading: %v", err)
	}

	listed, err := registry.List(ctx)
	if err != nil {
		t.Fatalf("list agents: %v", err)
	}
	if listed[0].Status != core.AgentStatusReading {
		t.Fatalf("expected reading status, got %q", listed[0].Status)
	}
	if listed[0].StatusReason != "Reading-like attached session activity detected." {
		t.Fatalf("unexpected reason %q", listed[0].StatusReason)
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

func TestUpdateNotificationPolicyNoOpDoesNotPersistOrEmitEvent(t *testing.T) {
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

	agent, err := registry.RegisterManaged(ctx, runtime.RegisterManagedInput{
		Provider:    "claude",
		DisplayName: "builder",
		ProjectPath: "/tmp/project",
	})
	if err != nil {
		t.Fatalf("register managed: %v", err)
	}

	eventsBefore, err := registry.Events(ctx, 0)
	if err != nil {
		t.Fatalf("events before no-op policy update: %v", err)
	}
	saveCallsBefore := countingStore.saveCalls

	updated, err := registry.UpdateNotificationPolicy(ctx, agent.ID, core.NotificationPolicyDefault)
	if err != nil {
		t.Fatalf("no-op update policy: %v", err)
	}
	if updated.NotificationPolicy != core.NotificationPolicyDefault {
		t.Fatalf("expected unchanged default policy, got %q", updated.NotificationPolicy)
	}
	if countingStore.saveCalls != saveCallsBefore {
		t.Fatalf("expected no extra save on no-op policy update, got %d -> %d", saveCallsBefore, countingStore.saveCalls)
	}

	eventsAfter, err := registry.Events(ctx, 0)
	if err != nil {
		t.Fatalf("events after no-op policy update: %v", err)
	}
	if len(eventsAfter) != len(eventsBefore) {
		t.Fatalf("expected no extra event on no-op policy update, got %d -> %d", len(eventsBefore), len(eventsAfter))
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

func TestUpdateRoleNoOpDoesNotPersistOrEmitEvent(t *testing.T) {
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

	agent, err := registry.RegisterManaged(ctx, runtime.RegisterManagedInput{
		Provider:    "claude",
		DisplayName: "builder",
		ProjectPath: "/tmp/project",
		Role:        "reviewer",
	})
	if err != nil {
		t.Fatalf("register managed: %v", err)
	}

	eventsBefore, err := registry.Events(ctx, 0)
	if err != nil {
		t.Fatalf("events before no-op role update: %v", err)
	}
	saveCallsBefore := countingStore.saveCalls

	updated, err := registry.UpdateRole(ctx, agent.ID, "reviewer")
	if err != nil {
		t.Fatalf("no-op update role: %v", err)
	}
	if updated.Role != "reviewer" {
		t.Fatalf("expected unchanged reviewer role, got %q", updated.Role)
	}
	if countingStore.saveCalls != saveCallsBefore {
		t.Fatalf("expected no extra save on no-op role update, got %d -> %d", saveCallsBefore, countingStore.saveCalls)
	}

	eventsAfter, err := registry.Events(ctx, 0)
	if err != nil {
		t.Fatalf("events after no-op role update: %v", err)
	}
	if len(eventsAfter) != len(eventsBefore) {
		t.Fatalf("expected no extra event on no-op role update, got %d -> %d", len(eventsBefore), len(eventsAfter))
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
	if events[len(events)-1].LifecycleMode != "managed" || events[len(events)-1].LifecycleStatus != "booting" {
		t.Fatalf("expected removed event to retain lifecycle metadata %#v", events[len(events)-1])
	}
	if events[len(events)-1].LifecycleReason != "Managed launch requested." || events[len(events)-1].LifecycleConfidence != 1 {
		t.Fatalf("expected removed event to retain lifecycle detail %#v", events[len(events)-1])
	}
	if events[len(events)-1].PresentationSummary != "Stopped tracking while booting. Managed launch requested." {
		t.Fatalf("expected removed event to expose lifecycle-aware presentation summary %#v", events[len(events)-1])
	}
}

func TestRecordHookTeammateIdle(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	registry := runtime.NewRegistry(
		store.NewFileAgentStore(filepath.Join(root, "agents.json")),
		store.NewFileEventStore(filepath.Join(root, "events.jsonl")),
	)

	agent, err := registry.RegisterManaged(ctx, runtime.RegisterManagedInput{
		Provider: "claude", DisplayName: "lead", ProjectPath: "/tmp",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := registry.RecordHookTeammateIdle(ctx, agent.ID, "worker-1", "teammate", "team"); err != nil {
		t.Fatalf("RecordHookTeammateIdle: %v", err)
	}

	agents, _ := registry.List(ctx)
	if len(agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(agents))
	}
	if agents[0].TeamRole != "teammate" {
		t.Fatalf("expected team_role=teammate, got %q", agents[0].TeamRole)
	}
	if agents[0].LastUserVisibleSummary != "Teammate idle: worker-1" {
		t.Fatalf("unexpected summary %q", agents[0].LastUserVisibleSummary)
	}
}

func TestRecordHookTaskCreatedAndCompleted(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	registry := runtime.NewRegistry(
		store.NewFileAgentStore(filepath.Join(root, "agents.json")),
		store.NewFileEventStore(filepath.Join(root, "events.jsonl")),
	)

	agent, err := registry.RegisterManaged(ctx, runtime.RegisterManagedInput{
		Provider: "claude", DisplayName: "lead", ProjectPath: "/tmp",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	// Create 2 tasks.
	if err := registry.RecordHookTaskCreated(ctx, agent.ID, "write tests", "unit tests for hooks", ""); err != nil {
		t.Fatalf("RecordHookTaskCreated: %v", err)
	}
	if err := registry.RecordHookTaskCreated(ctx, agent.ID, "fix bug", "", ""); err != nil {
		t.Fatalf("RecordHookTaskCreated 2: %v", err)
	}

	agents, _ := registry.List(ctx)
	if agents[0].TeamTaskTotal != 2 {
		t.Fatalf("expected TeamTaskTotal=2, got %d", agents[0].TeamTaskTotal)
	}
	if agents[0].TeamTaskCompleted != 0 {
		t.Fatalf("expected TeamTaskCompleted=0, got %d", agents[0].TeamTaskCompleted)
	}

	// Complete 1 task.
	if err := registry.RecordHookTaskCompleted(ctx, agent.ID, "write tests", ""); err != nil {
		t.Fatalf("RecordHookTaskCompleted: %v", err)
	}

	agents, _ = registry.List(ctx)
	if agents[0].TeamTaskTotal != 2 {
		t.Fatalf("expected TeamTaskTotal=2, got %d", agents[0].TeamTaskTotal)
	}
	if agents[0].TeamTaskCompleted != 1 {
		t.Fatalf("expected TeamTaskCompleted=1, got %d", agents[0].TeamTaskCompleted)
	}
	if agents[0].LastUserVisibleSummary != "Task completed: write tests" {
		t.Fatalf("unexpected summary %q", agents[0].LastUserVisibleSummary)
	}

	// Verify events.
	events, _ := registry.Events(ctx, 10)
	taskEvents := 0
	for _, ev := range events {
		if ev.Type == core.EventTypeTeamTaskCreated || ev.Type == core.EventTypeTeamTaskCompleted {
			taskEvents++
		}
	}
	if taskEvents != 3 {
		t.Fatalf("expected 3 team task events, got %d", taskEvents)
	}
}

func TestRecordHookToolFailed(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	agentStore := store.NewFileAgentStore(filepath.Join(root, "managed-agents.json"))
	registry := runtime.NewRegistry(agentStore, store.NewFileEventStore(filepath.Join(root, "events.jsonl")))

	agent, err := registry.RegisterManaged(ctx, runtime.RegisterManagedInput{
		Provider: "claude", DisplayName: "test", ProjectPath: "/tmp",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	// isInterrupt=true → WaitingInput
	if err := registry.RecordHookToolFailed(ctx, agent.ID, "Bash", "timeout", true, ""); err != nil {
		t.Fatalf("tool failed interrupt: %v", err)
	}
	snap, _ := registry.Snapshot(ctx)
	if snap.Agents[0].Status != core.AgentStatusWaitingInput {
		t.Fatalf("expected waiting_input, got %q", snap.Agents[0].Status)
	}

	// isInterrupt=false → Thinking
	if err := registry.RecordHookToolFailed(ctx, agent.ID, "Bash", "exit 1", false, ""); err != nil {
		t.Fatalf("tool failed no interrupt: %v", err)
	}
	snap, _ = registry.Snapshot(ctx)
	if snap.Agents[0].Status != core.AgentStatusThinking {
		t.Fatalf("expected thinking, got %q", snap.Agents[0].Status)
	}
}

func TestRecordHookUserPrompt(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	agentStore := store.NewFileAgentStore(filepath.Join(root, "managed-agents.json"))
	registry := runtime.NewRegistry(agentStore, store.NewFileEventStore(filepath.Join(root, "events.jsonl")))

	agent, err := registry.RegisterManaged(ctx, runtime.RegisterManagedInput{
		Provider: "claude", DisplayName: "test", ProjectPath: "/tmp",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := registry.RecordHookUserPrompt(ctx, agent.ID, "fix the bug in main.go please", ""); err != nil {
		t.Fatalf("user prompt: %v", err)
	}
	snap, _ := registry.Snapshot(ctx)
	if snap.Agents[0].Status != core.AgentStatusThinking {
		t.Fatalf("expected thinking, got %q", snap.Agents[0].Status)
	}
	if !strings.Contains(snap.Agents[0].LastUserVisibleSummary, "Prompt:") {
		t.Fatalf("expected prompt preview in summary, got %q", snap.Agents[0].LastUserVisibleSummary)
	}
}

func TestRecordHookPermissionRequest(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	agentStore := store.NewFileAgentStore(filepath.Join(root, "managed-agents.json"))
	registry := runtime.NewRegistry(agentStore, store.NewFileEventStore(filepath.Join(root, "events.jsonl")))

	agent, err := registry.RegisterManaged(ctx, runtime.RegisterManagedInput{
		Provider: "claude", DisplayName: "test", ProjectPath: "/tmp",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := registry.RecordHookPermissionRequest(ctx, agent.ID, "Bash", ""); err != nil {
		t.Fatalf("permission request: %v", err)
	}
	snap, _ := registry.Snapshot(ctx)
	if snap.Agents[0].Status != core.AgentStatusWaitingInput {
		t.Fatalf("expected waiting_input, got %q", snap.Agents[0].Status)
	}
}

func TestRecordHookPreCompactPostCompact(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	agentStore := store.NewFileAgentStore(filepath.Join(root, "managed-agents.json"))
	registry := runtime.NewRegistry(agentStore, store.NewFileEventStore(filepath.Join(root, "events.jsonl")))

	agent, err := registry.RegisterManaged(ctx, runtime.RegisterManagedInput{
		Provider: "claude", DisplayName: "test", ProjectPath: "/tmp",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := registry.RecordHookPreCompact(ctx, agent.ID, "auto", ""); err != nil {
		t.Fatalf("pre-compact: %v", err)
	}
	snap, _ := registry.Snapshot(ctx)
	if snap.Agents[0].LastUserVisibleSummary != "Compacting context..." {
		t.Fatalf("expected compacting summary, got %q", snap.Agents[0].LastUserVisibleSummary)
	}

	if err := registry.RecordHookPostCompact(ctx, agent.ID, "auto", "reduced to 50k tokens", ""); err != nil {
		t.Fatalf("post-compact: %v", err)
	}
	snap, _ = registry.Snapshot(ctx)
	if snap.Agents[0].Status != core.AgentStatusThinking {
		t.Fatalf("expected thinking after post-compact, got %q", snap.Agents[0].Status)
	}
	if !strings.Contains(snap.Agents[0].LastUserVisibleSummary, "reduced to 50k tokens") {
		t.Fatalf("expected compact summary, got %q", snap.Agents[0].LastUserVisibleSummary)
	}
}

func TestRecordHookWorktreeCreateRemove(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	agentStore := store.NewFileAgentStore(filepath.Join(root, "managed-agents.json"))
	registry := runtime.NewRegistry(agentStore, store.NewFileEventStore(filepath.Join(root, "events.jsonl")))

	agent, err := registry.RegisterManaged(ctx, runtime.RegisterManagedInput{
		Provider: "claude", DisplayName: "test", ProjectPath: "/tmp",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := registry.RecordHookWorktreeCreate(ctx, agent.ID, "feature-branch", ""); err != nil {
		t.Fatalf("worktree create: %v", err)
	}
	snap, _ := registry.Snapshot(ctx)
	if !strings.Contains(snap.Agents[0].LastUserVisibleSummary, "Worktree created: feature-branch") {
		t.Fatalf("expected worktree created summary, got %q", snap.Agents[0].LastUserVisibleSummary)
	}

	if err := registry.RecordHookWorktreeRemove(ctx, agent.ID, "/tmp/wt", ""); err != nil {
		t.Fatalf("worktree remove: %v", err)
	}
	snap, _ = registry.Snapshot(ctx)
	if !strings.Contains(snap.Agents[0].LastUserVisibleSummary, "Worktree removed") {
		t.Fatalf("expected worktree removed summary, got %q", snap.Agents[0].LastUserVisibleSummary)
	}
}

func TestRecordHookCwdChanged(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	agentStore := store.NewFileAgentStore(filepath.Join(root, "managed-agents.json"))
	registry := runtime.NewRegistry(agentStore, store.NewFileEventStore(filepath.Join(root, "events.jsonl")))

	agent, err := registry.RegisterManaged(ctx, runtime.RegisterManagedInput{
		Provider: "claude", DisplayName: "test", ProjectPath: "/tmp/old",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := registry.RecordHookCwdChanged(ctx, agent.ID, "/tmp/old", "/tmp/new", ""); err != nil {
		t.Fatalf("cwd changed: %v", err)
	}
	snap, _ := registry.Snapshot(ctx)
	if snap.Agents[0].ProjectPath != "/tmp/new" {
		t.Fatalf("expected project path updated to /tmp/new, got %q", snap.Agents[0].ProjectPath)
	}
}

func TestRecordHookElicitation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	agentStore := store.NewFileAgentStore(filepath.Join(root, "managed-agents.json"))
	registry := runtime.NewRegistry(agentStore, store.NewFileEventStore(filepath.Join(root, "events.jsonl")))

	agent, err := registry.RegisterManaged(ctx, runtime.RegisterManagedInput{
		Provider: "claude", DisplayName: "test", ProjectPath: "/tmp",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := registry.RecordHookElicitation(ctx, agent.ID, ""); err != nil {
		t.Fatalf("elicitation: %v", err)
	}
	snap, _ := registry.Snapshot(ctx)
	if snap.Agents[0].Status != core.AgentStatusWaitingInput {
		t.Fatalf("expected waiting_input, got %q", snap.Agents[0].Status)
	}

	if err := registry.RecordHookElicitationResult(ctx, agent.ID, ""); err != nil {
		t.Fatalf("elicitation result: %v", err)
	}
	snap, _ = registry.Snapshot(ctx)
	if snap.Agents[0].Status != core.AgentStatusThinking {
		t.Fatalf("expected thinking after elicitation result, got %q", snap.Agents[0].Status)
	}
}
