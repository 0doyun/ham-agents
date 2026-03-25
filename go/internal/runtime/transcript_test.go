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

func TestEnsureObservedTranscriptsRegistersAndRefreshesSources(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	transcript := filepath.Join(root, "agent.log")
	if err := os.WriteFile(transcript, []byte("Need input?\n"), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}

	registry := runtime.NewRegistry(
		store.NewFileAgentStore(filepath.Join(root, "managed-agents.json")),
		store.NewFileEventStore(filepath.Join(root, "events.jsonl")),
	)

	if err := registry.EnsureObservedTranscripts(ctx, []string{transcript}); err != nil {
		t.Fatalf("ensure observed transcripts: %v", err)
	}

	agents, err := registry.List(ctx)
	if err != nil {
		t.Fatalf("list agents: %v", err)
	}
	if len(agents) != 1 {
		t.Fatalf("expected 1 observed agent, got %#v", agents)
	}
	if agents[0].Provider != "transcript" || agents[0].Status != core.AgentStatusWaitingInput {
		t.Fatalf("unexpected observed agent %#v", agents[0])
	}
}
