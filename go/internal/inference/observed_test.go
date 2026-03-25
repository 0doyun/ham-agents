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

func TestRefreshObservedAgentDetectsExplicitInputRequest(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "observed.log")
	if err := os.WriteFile(path, []byte("Waiting for input before proceeding."), 0o644); err != nil {
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
	if updated.Status != core.AgentStatusWaitingInput {
		t.Fatalf("expected waiting_input status, got %q", updated.Status)
	}
	if updated.StatusConfidence != 0.65 {
		t.Fatalf("expected elevated explicit-signal confidence, got %v", updated.StatusConfidence)
	}
	if updated.StatusReason != "Explicit input request detected." {
		t.Fatalf("unexpected status reason %q", updated.StatusReason)
	}
}

func TestRefreshObservedAgentDetectsExplicitCompletionSignal(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "observed.log")
	if err := os.WriteFile(path, []byte("All tests passed; task complete."), 0o644); err != nil {
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
	if updated.Status != core.AgentStatusDone {
		t.Fatalf("expected done status, got %q", updated.Status)
	}
	if updated.StatusConfidence != 0.65 {
		t.Fatalf("expected elevated explicit-signal confidence, got %v", updated.StatusConfidence)
	}
	if updated.StatusReason != "Explicit completion-like output detected." {
		t.Fatalf("unexpected status reason %q", updated.StatusReason)
	}
}

func TestRefreshObservedAgentDoesNotTreatZeroFailedAsError(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "observed.log")
	if err := os.WriteFile(path, []byte("All tests passed with 0 failed checks."), 0o644); err != nil {
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
	if updated.Status != core.AgentStatusDone {
		t.Fatalf("expected done status, got %q", updated.Status)
	}
	if updated.StatusReason != "Explicit completion-like output detected." {
		t.Fatalf("unexpected status reason %q", updated.StatusReason)
	}
}

func TestRefreshObservedAgentDoesNotTreatNotCompletedAsDone(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "observed.log")
	if err := os.WriteFile(path, []byte("Task not completed yet; still working."), 0o644); err != nil {
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
	if updated.Status != core.AgentStatusThinking {
		t.Fatalf("expected thinking status, got %q", updated.Status)
	}
	if updated.StatusReason == "Completion-like output detected." || updated.StatusReason == "Explicit completion-like output detected." {
		t.Fatalf("expected negated completion text to avoid done inference, got %q", updated.StatusReason)
	}
}

func TestRefreshObservedAgentDoesNotTreatNoErrorAsError(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "observed.log")
	if err := os.WriteFile(path, []byte("Build finished with no error and still streaming output."), 0o644); err != nil {
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
	if updated.Status != core.AgentStatusThinking {
		t.Fatalf("expected thinking status, got %q", updated.Status)
	}
	if updated.StatusReason == "Error-like output detected." || updated.StatusReason == "Explicit error-like output detected." {
		t.Fatalf("expected negated error text to avoid error inference, got %q", updated.StatusReason)
	}
}

func TestRefreshObservedAgentDoesNotTreatNegatedInputAsWaiting(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "observed.log")
	if err := os.WriteFile(path, []byte("We don't need input anymore; continuing automatically."), 0o644); err != nil {
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
	if updated.Status != core.AgentStatusThinking {
		t.Fatalf("expected thinking status, got %q", updated.Status)
	}
	if updated.StatusReason == "Question-like output detected." || updated.StatusReason == "Explicit input request detected." {
		t.Fatalf("expected negated input text to avoid waiting inference, got %q", updated.StatusReason)
	}
}

func TestRefreshObservedAgentPrefersLatestCompletionLineOverEarlierError(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "observed.log")
	log := "previous step failed with error\nall tests passed\n"
	if err := os.WriteFile(path, []byte(log), 0o644); err != nil {
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
	if updated.Status != core.AgentStatusDone {
		t.Fatalf("expected done status, got %q", updated.Status)
	}
	if updated.StatusReason != "Explicit completion-like output detected." {
		t.Fatalf("unexpected status reason %q", updated.StatusReason)
	}
}

func TestRefreshObservedAgentPrefersLatestNegatedInputLineOverEarlierWaitingLine(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "observed.log")
	log := "waiting for input\nwe don't need input anymore; continuing automatically.\n"
	if err := os.WriteFile(path, []byte(log), 0o644); err != nil {
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
	if updated.Status != core.AgentStatusThinking {
		t.Fatalf("expected thinking status, got %q", updated.Status)
	}
}
