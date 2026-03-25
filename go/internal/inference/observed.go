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
	latestLine := latestNonEmptyObservedLine(content)
	modifiedAt := info.ModTime().UTC()
	age := now.UTC().Sub(modifiedAt)

	explicitErrorSignals := []string{"traceback", "panic:", "fatal:", "exception"}
	genericErrorSignals := []string{"error", "failed"}
	genericErrorNegations := []string{"no error", "without error", "0 failed", "zero failed", "not failed"}

	explicitDoneSignals := []string{"all tests passed", "finished successfully", "completed successfully", "task complete"}
	genericDoneSignals := []string{"done", "completed"}
	genericDoneNegations := []string{"not done", "not completed", "incomplete", "not yet done", "not yet completed", "task not complete", "not task complete"}

	explicitInputSignals := []string{"waiting for input", "needs input", "need input", "please confirm", "approval needed", "approve?"}
	genericInputNegations := []string{"no input needed", "don't need input", "doesn't need input", "input not needed", "approval not needed", "no approval needed", "do not need input"}

	kind, signalText := classifyObservedSignal(latestLine, content, explicitErrorSignals, genericErrorSignals, genericErrorNegations, explicitDoneSignals, genericDoneSignals, genericDoneNegations, explicitInputSignals, genericInputNegations)

	switch kind {
	case "error":
		agent.Status = core.AgentStatusError
		agent.StatusConfidence = observedSignalConfidence(signalText, 0.65, 0.55, explicitErrorSignals...)
		agent.StatusReason = observedSignalReason(signalText, "Explicit error-like output detected.", "Error-like output detected.", explicitErrorSignals...)
		agent.LastUserVisibleSummary = observedSignalSummary(signalText, "Observed explicit error-like output.", "Observed error-like output.", explicitErrorSignals...)
	case "done":
		agent.Status = core.AgentStatusDone
		agent.StatusConfidence = observedSignalConfidence(signalText, 0.65, 0.5, explicitDoneSignals...)
		agent.StatusReason = observedSignalReason(signalText, "Explicit completion-like output detected.", "Completion-like output detected.", explicitDoneSignals...)
		agent.LastUserVisibleSummary = observedSignalSummary(signalText, "Observed explicit completion-like output.", "Observed completion-like output.", explicitDoneSignals...)
	case "waiting_input":
		agent.Status = core.AgentStatusWaitingInput
		agent.StatusConfidence = observedSignalConfidence(signalText, 0.65, 0.45, explicitInputSignals...)
		agent.StatusReason = observedSignalReason(signalText, "Explicit input request detected.", "Question-like output detected.", explicitInputSignals...)
		agent.LastUserVisibleSummary = observedSignalSummary(signalText, "Observed explicit input request.", "Observed question-like output.", explicitInputSignals...)
	default:
		if age <= 2*time.Minute {
			agent.Status = core.AgentStatusThinking
			agent.StatusConfidence = 0.4
			agent.StatusReason = fmt.Sprintf("Output changed %s ago.", age.Round(time.Second))
			agent.LastUserVisibleSummary = fmt.Sprintf("Observed recent output (%s ago).", age.Round(time.Second))
		} else {
			agent.Status = core.AgentStatusSleeping
			agent.StatusConfidence = 0.3
			agent.StatusReason = fmt.Sprintf("No fresh output for %s.", age.Round(time.Second))
			agent.LastUserVisibleSummary = fmt.Sprintf("Observed source idle for %s.", age.Round(time.Second))
		}
	}

	agent.LastEventAt = modifiedAt
	return agent
}

func containsAny(content string, patterns ...string) bool {
	for _, pattern := range patterns {
		if strings.Contains(content, pattern) {
			return true
		}
	}
	return false
}

func containsSignal(content string, positive []string, negative []string) bool {
	if !containsAny(content, positive...) {
		return false
	}
	if len(negative) == 0 {
		return true
	}
	return !containsAny(content, negative...)
}

func latestNonEmptyObservedLine(content string) string {
	lines := strings.Split(content, "\n")
	for index := len(lines) - 1; index >= 0; index-- {
		line := strings.TrimSpace(lines[index])
		if line != "" {
			return line
		}
	}
	return ""
}

func classifyObservedSignal(
	latestLine string,
	fullContent string,
	explicitErrorSignals []string,
	genericErrorSignals []string,
	genericErrorNegations []string,
	explicitDoneSignals []string,
	genericDoneSignals []string,
	genericDoneNegations []string,
	explicitInputSignals []string,
	genericInputNegations []string,
) (kind string, signalText string) {
	if latestLine != "" {
		if kind := classifyObservedText(latestLine, explicitErrorSignals, genericErrorSignals, genericErrorNegations, explicitDoneSignals, genericDoneSignals, genericDoneNegations, explicitInputSignals, genericInputNegations); kind != "" {
			return kind, latestLine
		}
	}
	return classifyObservedText(fullContent, explicitErrorSignals, genericErrorSignals, genericErrorNegations, explicitDoneSignals, genericDoneSignals, genericDoneNegations, explicitInputSignals, genericInputNegations), fullContent
}

func classifyObservedText(
	text string,
	explicitErrorSignals []string,
	genericErrorSignals []string,
	genericErrorNegations []string,
	explicitDoneSignals []string,
	genericDoneSignals []string,
	genericDoneNegations []string,
	explicitInputSignals []string,
	genericInputNegations []string,
) string {
	switch {
	case containsSignal(text, explicitErrorSignals, nil) || containsSignal(text, genericErrorSignals, genericErrorNegations):
		return "error"
	case containsSignal(text, explicitDoneSignals, genericDoneNegations) || containsSignal(text, genericDoneSignals, genericDoneNegations):
		return "done"
	case containsSignal(text, explicitInputSignals, genericInputNegations) || (strings.Contains(text, "?") && !containsAny(text, genericInputNegations...)):
		return "waiting_input"
	default:
		return ""
	}
}

func observedSignalConfidence(content string, explicitConfidence float64, genericConfidence float64, explicitPatterns ...string) float64 {
	if containsAny(content, explicitPatterns...) {
		return explicitConfidence
	}
	return genericConfidence
}

func observedSignalReason(content string, explicitReason string, genericReason string, explicitPatterns ...string) string {
	if containsAny(content, explicitPatterns...) {
		return explicitReason
	}
	return genericReason
}

func observedSignalSummary(content string, explicitSummary string, genericSummary string, explicitPatterns ...string) string {
	if containsAny(content, explicitPatterns...) {
		return explicitSummary
	}
	return genericSummary
}
