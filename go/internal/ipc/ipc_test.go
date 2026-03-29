package ipc_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/core"
	"github.com/ham-agents/ham-agents/go/internal/ipc"
	"github.com/ham-agents/ham-agents/go/internal/runtime"
	"github.com/ham-agents/ham-agents/go/internal/store"
)

func TestClientServerRoundTripForManagedCommands(t *testing.T) {
	t.Parallel()

	root, err := os.MkdirTemp("/tmp", "hamd-ipc-")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })

	socketPath := filepath.Join(root, "s.sock")
	registry := runtime.NewRegistry(
		store.NewFileAgentStore(filepath.Join(root, "managed-agents.json")),
		store.NewFileEventStore(filepath.Join(root, "events.jsonl")),
	)
	managedService := runtime.NewManagedService(registry)
	settingsService := runtime.NewSettingsService(
		store.NewFileSettingsStore(filepath.Join(root, "settings.json")),
	)
	teamService := runtime.NewTeamService(
		store.NewFileTeamStore(filepath.Join(root, "teams.json")),
	)

	server := ipc.NewServer(socketPath, registry, managedService, settingsService, teamService, stubSessionLister{
		sessions: []core.AttachableSession{
			{ID: "abc", Title: "Claude", SessionRef: "iterm2://session/abc", IsActive: true},
		},
	}, stubSessionLister{
		sessions: []core.AttachableSession{
			{ID: "demo:1.0", Title: "ops", SessionRef: "tmux://demo:1.0", IsActive: true},
		},
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.Serve(ctx)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, err := os.Stat(socketPath); err == nil {
			break
		}
		select {
		case err := <-serverErrors:
			if err != nil && strings.Contains(err.Error(), "operation not permitted") {
				t.Skipf("unix socket binding unavailable in sandbox: %v", err)
			}
			t.Fatalf("server exited before socket became ready: %v", err)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatalf("socket did not appear: %s", socketPath)
		}
		time.Sleep(10 * time.Millisecond)
	}

	client := ipc.NewClient(socketPath)
	agent, err := client.RunManaged(context.Background(), runtime.RegisterManagedInput{
		Provider:    "claude",
		DisplayName: "builder",
		ProjectPath: "/tmp/project",
		Role:        "reviewer",
	})
	if err != nil {
		t.Fatalf("run managed via client: %v", err)
	}
	if agent.DisplayName != "builder" {
		t.Fatalf("unexpected display name %q", agent.DisplayName)
	}

	attached, err := client.AttachSession(context.Background(), runtime.RegisterAttachedInput{
		Provider:    "iterm2",
		DisplayName: "ops",
		ProjectPath: "/tmp/project",
		SessionRef:  "iterm2://session/abc",
	})
	if err != nil {
		t.Fatalf("attach session via client: %v", err)
	}
	if attached.Mode != "attached" {
		t.Fatalf("unexpected mode %q", attached.Mode)
	}

	observed, err := client.ObserveSource(context.Background(), runtime.RegisterObservedInput{
		Provider:    "log",
		DisplayName: "observer",
		ProjectPath: "/tmp/project",
		SessionRef:  "/tmp/project/transcript.log",
	})
	if err != nil {
		t.Fatalf("observe source via client: %v", err)
	}
	if observed.Mode != "observed" {
		t.Fatalf("unexpected mode %q", observed.Mode)
	}

	target, err := client.OpenTarget(context.Background(), attached.ID)
	if err != nil {
		t.Fatalf("open target via client: %v", err)
	}
	if target.Kind != "iterm_session" {
		t.Fatalf("unexpected open target kind %q", target.Kind)
	}
	if target.SessionID != "abc" {
		t.Fatalf("unexpected session id %q", target.SessionID)
	}

	sessions, err := client.ListItermSessions(context.Background())
	if err != nil {
		t.Fatalf("list iTerm sessions via client: %v", err)
	}
	if len(sessions) != 1 || sessions[0].ID != "abc" {
		t.Fatalf("unexpected attachable sessions %#v", sessions)
	}

	tmuxSessions, err := client.ListTmuxSessions(context.Background())
	if err != nil {
		t.Fatalf("list tmux sessions via client: %v", err)
	}
	if len(tmuxSessions) != 1 || tmuxSessions[0].ID != "demo:1.0" {
		t.Fatalf("unexpected tmux sessions %#v", tmuxSessions)
	}

	settings, err := client.Settings(context.Background())
	if err != nil {
		t.Fatalf("get settings via client: %v", err)
	}
	settings.Notifications.PreviewText = true
	settings.Notifications.QuietHoursStartHour = 21
	settings.Notifications.QuietHoursEndHour = 6
	settings.Appearance.Theme = "night"
	settings.Integrations.ItermEnabled = false
	updatedSettings, err := client.UpdateSettings(context.Background(), settings)
	if err != nil {
		t.Fatalf("update settings via client: %v", err)
	}
	if !updatedSettings.Notifications.PreviewText {
		t.Fatal("expected preview text to persist")
	}
	if updatedSettings.Notifications.QuietHoursStartHour != 21 || updatedSettings.Notifications.QuietHoursEndHour != 6 {
		t.Fatalf("expected quiet hours 21-6, got %d-%d", updatedSettings.Notifications.QuietHoursStartHour, updatedSettings.Notifications.QuietHoursEndHour)
	}
	if updatedSettings.Appearance.Theme != "night" {
		t.Fatalf("expected theme night, got %q", updatedSettings.Appearance.Theme)
	}
	if updatedSettings.Integrations.ItermEnabled {
		t.Fatal("expected iTerm integration to remain disabled")
	}

	agents, err := client.ListAgents(context.Background())
	if err != nil {
		t.Fatalf("list agents via client: %v", err)
	}
	if len(agents) != 3 {
		t.Fatalf("expected 3 agents, got %d", len(agents))
	}

	snapshot, err := client.Status(context.Background())
	if err != nil {
		t.Fatalf("status via client: %v", err)
	}
	if snapshot.TotalCount() != 3 {
		t.Fatalf("expected total count 3, got %d", snapshot.TotalCount())
	}

	events, err := client.Events(context.Background(), 20)
	if err != nil {
		t.Fatalf("events via client: %v", err)
	}
	registrationEvents := make([]core.Event, 0, 3)
	for _, e := range events {
		if e.Type == core.EventTypeAgentRegistered {
			registrationEvents = append(registrationEvents, e)
		}
	}
	if len(registrationEvents) != 3 {
		t.Fatalf("expected 3 registration events, got %d", len(registrationEvents))
	}
	if registrationEvents[0].AgentID != agent.ID {
		t.Fatalf("unexpected event agent id %q", registrationEvents[0].AgentID)
	}

	followedEvents, err := client.FollowEvents(context.Background(), registrationEvents[0].ID, 10, 0)
	if err != nil {
		t.Fatalf("follow events via client: %v", err)
	}
	followedRegistrationEvents := make([]core.Event, 0, 2)
	for _, e := range followedEvents {
		if e.Type == core.EventTypeAgentRegistered {
			followedRegistrationEvents = append(followedRegistrationEvents, e)
		}
	}
	if len(followedRegistrationEvents) != 2 {
		t.Fatalf("expected 2 newer registration events, got %d", len(followedRegistrationEvents))
	}

	updated, err := client.UpdateNotificationPolicy(context.Background(), agent.ID, core.NotificationPolicyMuted)
	if err != nil {
		t.Fatalf("update notification policy via client: %v", err)
	}
	if updated.NotificationPolicy != core.NotificationPolicyMuted {
		t.Fatalf("unexpected notification policy %q", updated.NotificationPolicy)
	}

	renamed, err := client.UpdateRole(context.Background(), agent.ID, "reviewer")
	if err != nil {
		t.Fatalf("update role via client: %v", err)
	}
	if renamed.Role != "reviewer" {
		t.Fatalf("unexpected role %q", renamed.Role)
	}

	if err := client.RemoveAgent(context.Background(), agent.ID); err != nil {
		t.Fatalf("remove agent via client: %v", err)
	}

	agentsAfterRemove, err := client.ListAgents(context.Background())
	if err != nil {
		t.Fatalf("list after remove: %v", err)
	}
	if len(agentsAfterRemove) != 2 {
		t.Fatalf("expected 2 remaining agents after remove, got %d", len(agentsAfterRemove))
	}
	for _, remaining := range agentsAfterRemove {
		if remaining.ID == agent.ID {
			t.Fatalf("managed agent %q should have been removed", agent.ID)
		}
	}

	cancel()
	if err := <-serverErrors; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("server shutdown: %v", err)
	}
}

type stubSessionLister struct {
	sessions []core.AttachableSession
}

func (s stubSessionLister) ListSessions() ([]core.AttachableSession, error) {
	return append([]core.AttachableSession(nil), s.sessions...), nil
}

func TestServerRejectsDirectoryAtSocketPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	socketPath := filepath.Join(root, "socket-dir")
	if err := os.MkdirAll(socketPath, 0o755); err != nil {
		t.Fatalf("create socket directory: %v", err)
	}

	registry := runtime.NewRegistry(
		store.NewFileAgentStore(filepath.Join(root, "managed-agents.json")),
		store.NewFileEventStore(filepath.Join(root, "events.jsonl")),
	)
	managedService := runtime.NewManagedService(registry)
	settingsService := runtime.NewSettingsService(
		store.NewFileSettingsStore(filepath.Join(root, "settings.json")),
	)
	teamService := runtime.NewTeamService(
		store.NewFileTeamStore(filepath.Join(root, "teams.json")),
	)

	server := ipc.NewServer(socketPath, registry, managedService, settingsService, teamService, nil, nil)
	err := server.Serve(context.Background())
	if err == nil {
		t.Fatal("expected server startup to fail when socket path is a directory")
	}
	if !strings.Contains(err.Error(), "not a unix socket") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientServerRoundTripForTeamCommands(t *testing.T) {
	t.Parallel()

	root, err := os.MkdirTemp("/tmp", "hamd-ipc-team-")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })

	socketPath := filepath.Join(root, "s.sock")
	registry := runtime.NewRegistry(
		store.NewFileAgentStore(filepath.Join(root, "managed-agents.json")),
		store.NewFileEventStore(filepath.Join(root, "events.jsonl")),
	)
	managedService := runtime.NewManagedService(registry)
	settingsService := runtime.NewSettingsService(
		store.NewFileSettingsStore(filepath.Join(root, "settings.json")),
	)
	teamService := runtime.NewTeamService(
		store.NewFileTeamStore(filepath.Join(root, "teams.json")),
	)

	server := ipc.NewServer(socketPath, registry, managedService, settingsService, teamService, nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.Serve(ctx)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, err := os.Stat(socketPath); err == nil {
			break
		}
		select {
		case err := <-serverErrors:
			if err != nil && strings.Contains(err.Error(), "operation not permitted") {
				t.Skipf("unix socket binding unavailable in sandbox: %v", err)
			}
			t.Fatalf("server exited before socket became ready: %v", err)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatalf("socket did not appear: %s", socketPath)
		}
		time.Sleep(10 * time.Millisecond)
	}

	client := ipc.NewClient(socketPath)
	agent, err := client.RunManaged(context.Background(), runtime.RegisterManagedInput{
		Provider:    "claude",
		DisplayName: "builder",
		ProjectPath: "/tmp/project",
	})
	if err != nil {
		t.Fatalf("run managed: %v", err)
	}

	team, err := client.CreateTeam(context.Background(), "frontend")
	if err != nil {
		t.Fatalf("create team: %v", err)
	}
	if team.DisplayName != "frontend" {
		t.Fatalf("unexpected team %#v", team)
	}

	updated, err := client.AddTeamMember(context.Background(), team.ID, agent.ID)
	if err != nil {
		t.Fatalf("add team member: %v", err)
	}
	if len(updated.MemberAgentIDs) != 1 || updated.MemberAgentIDs[0] != agent.ID {
		t.Fatalf("unexpected updated team %#v", updated)
	}

	listed, err := client.ListTeams(context.Background())
	if err != nil {
		t.Fatalf("list teams: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != team.ID {
		t.Fatalf("unexpected listed teams %#v", listed)
	}

	cancel()
	if err := <-serverErrors; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("server shutdown: %v", err)
	}
}

func TestClientServerStopManagedStopsProcess(t *testing.T) {
	root, err := os.MkdirTemp("/tmp", "hamd-ipc-stop-")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })

	t.Setenv("HAM_MANAGED_PROVIDER_LONGPROC_SHELL", "trap 'exit 0' TERM; while true; do sleep 1; done")

	socketPath := filepath.Join(root, "s.sock")
	registry := runtime.NewRegistry(
		store.NewFileAgentStore(filepath.Join(root, "managed-agents.json")),
		store.NewFileEventStore(filepath.Join(root, "events.jsonl")),
	)
	managedService := runtime.NewManagedService(registry)
	settingsService := runtime.NewSettingsService(
		store.NewFileSettingsStore(filepath.Join(root, "settings.json")),
	)
	teamService := runtime.NewTeamService(
		store.NewFileTeamStore(filepath.Join(root, "teams.json")),
	)

	server := ipc.NewServer(socketPath, registry, managedService, settingsService, teamService, nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.Serve(ctx)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, err := os.Stat(socketPath); err == nil {
			break
		}
		select {
		case err := <-serverErrors:
			if err != nil && strings.Contains(err.Error(), "operation not permitted") {
				t.Skipf("unix socket binding unavailable in sandbox: %v", err)
			}
			t.Fatalf("server exited before socket became ready: %v", err)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatalf("socket did not appear: %s", socketPath)
		}
		time.Sleep(10 * time.Millisecond)
	}

	client := ipc.NewClient(socketPath)
	agent, err := client.RunManaged(context.Background(), runtime.RegisterManagedInput{
		Provider:    "longproc",
		DisplayName: "builder",
		ProjectPath: root,
	})
	if err != nil {
		t.Fatalf("run managed: %v", err)
	}

	if err := client.StopManaged(context.Background(), agent.ID); err != nil {
		t.Fatalf("stop managed: %v", err)
	}

	stopDeadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(stopDeadline) {
		snapshot, err := registry.Snapshot(context.Background())
		if err != nil {
			t.Fatalf("snapshot: %v", err)
		}
		if len(snapshot.Agents) > 0 && snapshot.Agents[0].Status == core.AgentStatusDone && snapshot.Agents[0].StatusReason == "Managed process stopped." {
			cancel()
			if err := <-serverErrors; err != nil && !errors.Is(err, context.Canceled) {
				t.Fatalf("server shutdown: %v", err)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	t.Fatal("managed process did not transition to stopped state before timeout")
}

func TestClientServerRoundTripForHookCommands(t *testing.T) {
	t.Parallel()

	root, err := os.MkdirTemp("/tmp", "hamd-ipc-hook-")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })

	socketPath := filepath.Join(root, "s.sock")
	registry := runtime.NewRegistry(
		store.NewFileAgentStore(filepath.Join(root, "managed-agents.json")),
		store.NewFileEventStore(filepath.Join(root, "events.jsonl")),
	)
	managedService := runtime.NewManagedService(registry)
	settingsService := runtime.NewSettingsService(
		store.NewFileSettingsStore(filepath.Join(root, "settings.json")),
	)
	teamService := runtime.NewTeamService(
		store.NewFileTeamStore(filepath.Join(root, "teams.json")),
	)

	server := ipc.NewServer(socketPath, registry, managedService, settingsService, teamService, stubSessionLister{}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.Serve(ctx)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, err := os.Stat(socketPath); err == nil {
			break
		}
		select {
		case err := <-serverErrors:
			if err != nil && strings.Contains(err.Error(), "operation not permitted") {
				t.Skipf("unix socket binding unavailable in sandbox: %v", err)
			}
			t.Fatalf("server exited before socket became ready: %v", err)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatalf("socket did not appear: %s", socketPath)
		}
		time.Sleep(10 * time.Millisecond)
	}

	client := ipc.NewClient(socketPath)

	// Register a managed agent to use as hook target.
	agent, err := client.RunManaged(context.Background(), runtime.RegisterManagedInput{
		Provider:    "claude",
		DisplayName: "hook-test",
		ProjectPath: "/tmp/project",
	})
	if err != nil {
		t.Fatalf("register managed agent: %v", err)
	}

	// HookToolStart should transition agent to a tool-related status.
	if err := client.HookToolStart(context.Background(), agent.ID, "Read"); err != nil {
		t.Fatalf("hook tool-start: %v", err)
	}
	snapshot, err := client.Status(context.Background())
	if err != nil {
		t.Fatalf("status after tool-start: %v", err)
	}
	if len(snapshot.Agents) == 0 {
		t.Fatal("expected at least one agent after tool-start")
	}
	agentAfterToolStart := snapshot.Agents[0]
	if agentAfterToolStart.Status != core.AgentStatusReading {
		t.Fatalf("expected reading status after Read tool-start, got %q", agentAfterToolStart.Status)
	}
	if agentAfterToolStart.StatusConfidence != 1.0 {
		t.Fatalf("expected confidence 1.0, got %f", agentAfterToolStart.StatusConfidence)
	}

	// HookToolDone should transition back to thinking.
	if err := client.HookToolDone(context.Background(), agent.ID, "Read"); err != nil {
		t.Fatalf("hook tool-done: %v", err)
	}
	snapshot, err = client.Status(context.Background())
	if err != nil {
		t.Fatalf("status after tool-done: %v", err)
	}
	if snapshot.Agents[0].Status != core.AgentStatusThinking {
		t.Fatalf("expected thinking status after tool-done, got %q", snapshot.Agents[0].Status)
	}

	// HookAgentSpawned should increment SubAgentCount.
	if err := client.HookAgentSpawned(context.Background(), agent.ID, "sub-task"); err != nil {
		t.Fatalf("hook agent-spawned: %v", err)
	}
	snapshot, err = client.Status(context.Background())
	if err != nil {
		t.Fatalf("status after agent-spawned: %v", err)
	}
	if snapshot.Agents[0].SubAgentCount != 1 {
		t.Fatalf("expected SubAgentCount 1, got %d", snapshot.Agents[0].SubAgentCount)
	}

	// HookAgentFinished should decrement SubAgentCount.
	if err := client.HookAgentFinished(context.Background(), agent.ID); err != nil {
		t.Fatalf("hook agent-finished: %v", err)
	}
	snapshot, err = client.Status(context.Background())
	if err != nil {
		t.Fatalf("status after agent-finished: %v", err)
	}
	if snapshot.Agents[0].SubAgentCount != 0 {
		t.Fatalf("expected SubAgentCount 0, got %d", snapshot.Agents[0].SubAgentCount)
	}

	// HookSessionEnd should transition to done.
	if err := client.HookSessionEnd(context.Background(), agent.ID); err != nil {
		t.Fatalf("hook session-end: %v", err)
	}
	snapshot, err = client.Status(context.Background())
	if err != nil {
		t.Fatalf("status after session-end: %v", err)
	}
	if snapshot.Agents[0].Status != core.AgentStatusDone {
		t.Fatalf("expected done status after session-end, got %q", snapshot.Agents[0].Status)
	}

	cancel()
	if err := <-serverErrors; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("server shutdown: %v", err)
	}
}
