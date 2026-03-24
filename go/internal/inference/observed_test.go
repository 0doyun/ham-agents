package inference_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/core"
	"github.com/ham-agents/ham-agents/go/internal/inference"
)

func TestRefreshObservedAgentDetectsErrorLikeOutput(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "observed.log")
	if err := os.WriteFile(path, []byte("task failed with error"), 0o644); err != nil {
		t.Fatalf("write observed log: %v", err)
	}

	agent := core.Agent{
		ID:               "agent-1",
		DisplayName:      "observer",
		Mode:             core.AgentModeObserved,
		SessionRef:       path,
		Status:           core.AgentStatusIdle,
		StatusConfidence: 0.35,
	}

	updated := inference.RefreshObservedAgent(agent, time.Now())
	if updated.Status != core.AgentStatusError {
		t.Fatalf("expected error status, got %q", updated.Status)
	}
}

func TestRefreshObservedAgentFallsBackToSleepingForStaleLog(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "observed.log")
	if err := os.WriteFile(path, []byte("still watching"), 0o644); err != nil {
		t.Fatalf("write observed log: %v", err)
	}

	old := time.Now().Add(-10 * time.Minute)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatalf("set old modtime: %v", err)
	}

	agent := core.Agent{
		ID:               "agent-1",
		DisplayName:      "observer",
		Mode:             core.AgentModeObserved,
		SessionRef:       path,
		Status:           core.AgentStatusIdle,
		StatusConfidence: 0.35,
	}

	updated := inference.RefreshObservedAgent(agent, time.Now())
	if updated.Status != core.AgentStatusSleeping {
		t.Fatalf("expected sleeping status, got %q", updated.Status)
	}
}
