package inference

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/core"
)

func RefreshObservedAgent(agent core.Agent, now time.Time) core.Agent {
	path := strings.TrimSpace(agent.SessionRef)
	if path == "" {
		agent.Status = core.AgentStatusDisconnected
		agent.StatusConfidence = 0.2
		agent.StatusReason = "Observed source path missing."
		agent.LastUserVisibleSummary = "Observed source is missing."
		agent.LastEventAt = now.UTC()
		return agent
	}

	info, err := os.Stat(path)
	if err != nil {
		agent.Status = core.AgentStatusDisconnected
		agent.StatusConfidence = 0.25
		agent.StatusReason = "Observed source unavailable."
		agent.LastUserVisibleSummary = "Observed source is unavailable."
		agent.LastEventAt = now.UTC()
		return agent
	}

	payload, err := os.ReadFile(path)
	if err != nil {
		agent.Status = core.AgentStatusDisconnected
		agent.StatusConfidence = 0.25
		agent.StatusReason = "Observed source unreadable."
		agent.LastUserVisibleSummary = "Observed source could not be read."
		agent.LastEventAt = now.UTC()
		return agent
	}

	content := strings.ToLower(string(payload))
	modifiedAt := info.ModTime().UTC()
	age := now.UTC().Sub(modifiedAt)

	switch {
	case strings.Contains(content, "error") || strings.Contains(content, "failed"):
		agent.Status = core.AgentStatusError
		agent.StatusConfidence = 0.55
		agent.StatusReason = "Error-like output detected."
		agent.LastUserVisibleSummary = "Observed error-like output."
	case strings.Contains(content, "done") || strings.Contains(content, "completed"):
		agent.Status = core.AgentStatusDone
		agent.StatusConfidence = 0.5
		agent.StatusReason = "Completion-like output detected."
		agent.LastUserVisibleSummary = "Observed completion-like output."
	case strings.Contains(content, "?"):
		agent.Status = core.AgentStatusWaitingInput
		agent.StatusConfidence = 0.45
		agent.StatusReason = "Question-like output detected."
		agent.LastUserVisibleSummary = "Observed question-like output."
	case age <= 2*time.Minute:
		agent.Status = core.AgentStatusThinking
		agent.StatusConfidence = 0.4
		agent.StatusReason = fmt.Sprintf("Output changed %s ago.", age.Round(time.Second))
		agent.LastUserVisibleSummary = fmt.Sprintf("Observed recent output (%s ago).", age.Round(time.Second))
	default:
		agent.Status = core.AgentStatusSleeping
		agent.StatusConfidence = 0.3
		agent.StatusReason = fmt.Sprintf("No fresh output for %s.", age.Round(time.Second))
		agent.LastUserVisibleSummary = fmt.Sprintf("Observed source idle for %s.", age.Round(time.Second))
	}

	agent.LastEventAt = modifiedAt
	return agent
}
