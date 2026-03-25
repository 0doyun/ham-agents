package adapters

import (
	"encoding/json"
	"strings"

	"github.com/ham-agents/ham-agents/go/internal/core"
)

type ManagedOutputSummary struct {
	Status     core.AgentStatus
	Confidence float64
	Reason     string
	Summary    string
}

func InferManagedOutput(provider string, line string, isStderr bool) ManagedOutputSummary {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return ManagedOutputSummary{
			Status:     core.AgentStatusThinking,
			Confidence: 1,
			Reason:     "Managed process emitted output.",
			Summary:    "",
		}
	}

	if strings.EqualFold(strings.TrimSpace(provider), "claude") {
		if inferred, ok := inferClaudeStructuredOutput(trimmed); ok {
			return inferred
		}
	}

	lower := strings.ToLower(trimmed)
	switch {
	case strings.Contains(lower, "need input"), strings.Contains(lower, "needs input"), strings.Contains(lower, "approval"), strings.Contains(lower, "?"):
		return ManagedOutputSummary{
			Status:     core.AgentStatusWaitingInput,
			Confidence: 1,
			Reason:     "Managed process is waiting for input.",
			Summary:    trimmed,
		}
	case strings.Contains(lower, "done"), strings.Contains(lower, "complete"), strings.Contains(lower, "finished successfully"):
		return ManagedOutputSummary{
			Status:     core.AgentStatusDone,
			Confidence: 1,
			Reason:     "Managed process reported completion.",
			Summary:    trimmed,
		}
	case isStderr && (strings.Contains(lower, "error") || strings.Contains(lower, "failed")):
		return ManagedOutputSummary{
			Status:     core.AgentStatusError,
			Confidence: 1,
			Reason:     "Managed process emitted error output.",
			Summary:    trimmed,
		}
	default:
		return ManagedOutputSummary{
			Status:     core.AgentStatusThinking,
			Confidence: 1,
			Reason:     "Managed process emitted output.",
			Summary:    trimmed,
		}
	}
}

func inferClaudeStructuredOutput(line string) (ManagedOutputSummary, bool) {
	var payload struct {
		Status  string `json:"status"`
		Reason  string `json:"reason"`
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal([]byte(line), &payload); err != nil {
		return ManagedOutputSummary{}, false
	}

	status := core.AgentStatus(strings.TrimSpace(payload.Status))
	switch status {
	case core.AgentStatusThinking, core.AgentStatusWaitingInput, core.AgentStatusDone, core.AgentStatusError, core.AgentStatusReading, core.AgentStatusRunningTool, core.AgentStatusBooting:
	default:
		return ManagedOutputSummary{}, false
	}

	reason := strings.TrimSpace(payload.Reason)
	if reason == "" {
		reason = "Claude structured output detected."
	}
	summary := strings.TrimSpace(payload.Summary)
	if summary == "" {
		summary = line
	}

	return ManagedOutputSummary{
		Status:     status,
		Confidence: 1,
		Reason:     reason,
		Summary:    summary,
	}, true
}
