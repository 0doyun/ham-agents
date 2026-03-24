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

	server := ipc.NewServer(socketPath, registry)
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

	agents, err := client.ListAgents(context.Background())
	if err != nil {
		t.Fatalf("list agents via client: %v", err)
	}
	if len(agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(agents))
	}

	snapshot, err := client.Status(context.Background())
	if err != nil {
		t.Fatalf("status via client: %v", err)
	}
	if snapshot.TotalCount() != 2 {
		t.Fatalf("expected total count 2, got %d", snapshot.TotalCount())
	}

	events, err := client.Events(context.Background(), 10)
	if err != nil {
		t.Fatalf("events via client: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].AgentID != agent.ID {
		t.Fatalf("unexpected event agent id %q", events[0].AgentID)
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
	if len(agentsAfterRemove) != 0 {
		t.Fatalf("expected empty registry after remove, got %d", len(agentsAfterRemove))
	}

	cancel()
	if err := <-serverErrors; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("server shutdown: %v", err)
	}
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

	server := ipc.NewServer(socketPath, registry)
	err := server.Serve(context.Background())
	if err == nil {
		t.Fatal("expected server startup to fail when socket path is a directory")
	}
	if !strings.Contains(err.Error(), "not a unix socket") {
		t.Fatalf("unexpected error: %v", err)
	}
}
