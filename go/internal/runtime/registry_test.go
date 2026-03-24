package runtime_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ham-agents/ham-agents/go/internal/runtime"
	"github.com/ham-agents/ham-agents/go/internal/store"
)

func TestRegisterManagedPersistsAndBuildsSnapshot(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	statePath := filepath.Join(t.TempDir(), "managed-agents.json")
	registry := runtime.NewRegistry(store.NewFileAgentStore(statePath))

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
}
