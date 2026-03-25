package adapters

import (
	"strings"

	"github.com/ham-agents/ham-agents/go/internal/core"
)

type ProcessExitSummary struct {
	Status  core.AgentStatus
	Reason  string
	Summary string
}

func ClassifyProcessExit(err error) ProcessExitSummary {
	if err == nil {
		return ProcessExitSummary{
			Status:  core.AgentStatusDone,
			Reason:  "Managed process exited successfully.",
			Summary: "Managed process exited successfully.",
		}
	}

	summary := strings.TrimSpace(err.Error())
	if summary == "" {
		summary = "Managed process exited with an error."
	}
	return ProcessExitSummary{
		Status:  core.AgentStatusError,
		Reason:  "Managed process exited with an error.",
		Summary: summary,
	}
}
