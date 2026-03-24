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
	settingsService := runtime.NewSettingsService(
		store.NewFileSettingsStore(filepath.Join(root, "settings.json")),
	)

	server := ipc.NewServer(socketPath, registry, settingsService, stubSessionLister{
		sessions: []core.AttachableSession{
			{ID: "abc", Title: "Claude", SessionRef: "iterm2://session/abc", IsActive: true},
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

	events, err := client.Events(context.Background(), 10)
	if err != nil {
		t.Fatalf("events via client: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if events[0].AgentID != agent.ID {
		t.Fatalf("unexpected event agent id %q", events[0].AgentID)
	}

	followedEvents, err := client.FollowEvents(context.Background(), events[0].ID, 10, 0)
	if err != nil {
		t.Fatalf("follow events via client: %v", err)
	}
	if len(followedEvents) != 2 {
		t.Fatalf("expected 2 newer events, got %d", len(followedEvents))
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
	settingsService := runtime.NewSettingsService(
		store.NewFileSettingsStore(filepath.Join(root, "settings.json")),
	)

	server := ipc.NewServer(socketPath, registry, settingsService, nil)
	err := server.Serve(context.Background())
	if err == nil {
		t.Fatal("expected server startup to fail when socket path is a directory")
	}
	if !strings.Contains(err.Error(), "not a unix socket") {
		t.Fatalf("unexpected error: %v", err)
	}
}
