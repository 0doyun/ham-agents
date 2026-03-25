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

func TestRefreshObservedAgentTreatsLatestContinuationLineAsThinking(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "observed.log")
	log := "waiting for input\ncontinuing automatically with fallback.\n"
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
	if updated.StatusReason != "Continuation-like output detected." {
		t.Fatalf("unexpected status reason %q", updated.StatusReason)
	}
	if updated.LastUserVisibleSummary != "Observed continuing output." {
		t.Fatalf("unexpected visible summary %q", updated.LastUserVisibleSummary)
	}
}

func TestRefreshObservedAgentRecentGenericOutputKeepsTimeBasedThinkingReason(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "observed.log")
	log := "plain heartbeat output\n"
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
	if updated.StatusReason != "Output changed 0s ago." {
		t.Fatalf("unexpected status reason %q", updated.StatusReason)
	}
}

func TestRefreshObservedAgentDetectsToolLikeOutput(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "observed.log")
	log := "Running tool apply_patch on tasks.md\n"
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
	if updated.Status != core.AgentStatusRunningTool {
		t.Fatalf("expected running_tool status, got %q", updated.Status)
	}
	if updated.StatusReason != "Tool-like output detected." {
		t.Fatalf("unexpected status reason %q", updated.StatusReason)
	}
}

func TestRefreshObservedAgentDetectsReadingLikeOutput(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "observed.log")
	log := "Reviewing architecture notes before next change\n"
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
	if updated.Status != core.AgentStatusReading {
		t.Fatalf("expected reading status, got %q", updated.Status)
	}
	if updated.StatusReason != "Reading-like output detected." {
		t.Fatalf("unexpected status reason %q", updated.StatusReason)
	}
}

func TestRefreshObservedAgentDetectsThinkingLikeOutput(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "observed.log")
	log := "Planning the next patch before editing\n"
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
	if updated.StatusReason != "Thinking-like output detected." {
		t.Fatalf("unexpected status reason %q", updated.StatusReason)
	}
	if updated.LastUserVisibleSummary != "Observed thinking-like activity." {
		t.Fatalf("unexpected summary %q", updated.LastUserVisibleSummary)
	}
}

func TestRefreshObservedAgentDetectsSleepingLikeOutput(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "observed.log")
	log := "Paused and waiting for changes until the next task\n"
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
	if updated.Status != core.AgentStatusSleeping {
		t.Fatalf("expected sleeping status, got %q", updated.Status)
	}
	if updated.StatusReason != "Sleeping-like output detected." {
		t.Fatalf("unexpected status reason %q", updated.StatusReason)
	}
	if updated.LastUserVisibleSummary != "Observed sleeping-like activity." {
		t.Fatalf("unexpected summary %q", updated.LastUserVisibleSummary)
	}
}

func TestRefreshObservedAgentDetectsBootingLikeOutput(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "observed.log")
	log := "Initializing workspace bootstrap before first step\n"
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
	if updated.Status != core.AgentStatusBooting {
		t.Fatalf("expected booting status, got %q", updated.Status)
	}
	if updated.StatusReason != "Booting-like output detected." {
		t.Fatalf("unexpected status reason %q", updated.StatusReason)
	}
	if updated.LastUserVisibleSummary != "Observed booting-like activity." {
		t.Fatalf("unexpected summary %q", updated.LastUserVisibleSummary)
	}
}

func TestRefreshObservedAgentDetectsIdleLikeOutput(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "observed.log")
	log := "Ready and standing by for the next task\n"
	if err := os.WriteFile(path, []byte(log), 0o644); err != nil {
		t.Fatalf("write observed log: %v", err)
	}

	agent := core.Agent{
		ID:               "agent-1",
		DisplayName:      "observer",
		Mode:             core.AgentModeObserved,
		SessionRef:       path,
		Status:           core.AgentStatusThinking,
		StatusConfidence: 0.35,
	}

	updated := inference.RefreshObservedAgent(agent, time.Now())
	if updated.Status != core.AgentStatusIdle {
		t.Fatalf("expected idle status, got %q", updated.Status)
	}
	if updated.StatusReason != "Idle-like output detected." {
		t.Fatalf("unexpected status reason %q", updated.StatusReason)
	}
	if updated.LastUserVisibleSummary != "Observed idle-like activity." {
		t.Fatalf("unexpected summary %q", updated.LastUserVisibleSummary)
	}
}

func TestRefreshObservedAgentDetectsDisconnectedLikeOutput(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "observed.log")
	log := "Session vanished and is now disconnected from the terminal\n"
	if err := os.WriteFile(path, []byte(log), 0o644); err != nil {
		t.Fatalf("write observed log: %v", err)
	}

	agent := core.Agent{
		ID:               "agent-1",
		DisplayName:      "observer",
		Mode:             core.AgentModeObserved,
		SessionRef:       path,
		Status:           core.AgentStatusThinking,
		StatusConfidence: 0.35,
	}

	updated := inference.RefreshObservedAgent(agent, time.Now())
	if updated.Status != core.AgentStatusDisconnected {
		t.Fatalf("expected disconnected status, got %q", updated.Status)
	}
	if updated.StatusReason != "Disconnected-like output detected." {
		t.Fatalf("unexpected status reason %q", updated.StatusReason)
	}
	if updated.LastUserVisibleSummary != "Observed disconnected-like output." {
		t.Fatalf("unexpected summary %q", updated.LastUserVisibleSummary)
	}
}
