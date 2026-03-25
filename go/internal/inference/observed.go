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
	explicitDisconnectedSignals := []string{"disconnected", "connection lost", "offline", "session lost", "session vanished"}
	explicitDisconnectedNegations := []string{"not disconnected", "reconnected", "connection restored", "back online"}

	explicitDoneSignals := []string{"all tests passed", "finished successfully", "completed successfully", "task complete"}
	genericDoneSignals := []string{"done", "completed"}
	genericDoneNegations := []string{"not done", "not completed", "incomplete", "not yet done", "not yet completed", "task not complete", "not task complete"}

	explicitInputSignals := []string{"waiting for input", "needs input", "need input", "please confirm", "approval needed", "approve?"}
	genericInputNegations := []string{"no input needed", "don't need input", "doesn't need input", "input not needed", "approval not needed", "no approval needed", "do not need input"}

	explicitToolSignals := []string{"running tool", "tool call", "invoking tool", "executing command", "apply_patch"}
	explicitReadingSignals := []string{"reading ", "inspecting ", "analyzing ", "reviewing ", "searching "}
	explicitThinkingSignals := []string{"thinking", "planning", "investigating", "drafting", "reasoning"}
	explicitIdleSignals := []string{"idle", "ready", "standing by", "standing-by", "awaiting work"}
	explicitSleepingSignals := []string{"sleeping", "paused", "waiting for changes"}
	explicitBootingSignals := []string{"starting up", "initializing", "booting", "launching", "warming up"}

	kind, signalText := classifyObservedSignal(latestLine, content, explicitErrorSignals, genericErrorSignals, genericErrorNegations, explicitDisconnectedSignals, explicitDisconnectedNegations, explicitDoneSignals, genericDoneSignals, genericDoneNegations, explicitInputSignals, genericInputNegations, explicitToolSignals, explicitReadingSignals, explicitThinkingSignals, explicitIdleSignals, explicitSleepingSignals, explicitBootingSignals)
	continuationLine := latestLine != "" && indicatesObservedContinuation(latestLine)

	switch kind {
	case "booting":
		agent.Status = core.AgentStatusBooting
		agent.StatusConfidence = 0.38
		agent.StatusReason = "Booting-like output detected."
		agent.LastUserVisibleSummary = "Observed booting-like activity."
	case "disconnected":
		agent.Status = core.AgentStatusDisconnected
		agent.StatusConfidence = 0.4
		agent.StatusReason = "Disconnected-like output detected."
		agent.LastUserVisibleSummary = "Observed disconnected-like output."
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
	case "running_tool":
		agent.Status = core.AgentStatusRunningTool
		agent.StatusConfidence = 0.5
		agent.StatusReason = "Tool-like output detected."
		agent.LastUserVisibleSummary = "Observed tool-like activity."
	case "reading":
		agent.Status = core.AgentStatusReading
		agent.StatusConfidence = 0.45
		agent.StatusReason = "Reading-like output detected."
		agent.LastUserVisibleSummary = "Observed reading-like activity."
	case "thinking":
		agent.Status = core.AgentStatusThinking
		agent.StatusConfidence = 0.43
		agent.StatusReason = "Thinking-like output detected."
		agent.LastUserVisibleSummary = "Observed thinking-like activity."
	case "idle":
		agent.Status = core.AgentStatusIdle
		agent.StatusConfidence = 0.36
		agent.StatusReason = "Idle-like output detected."
		agent.LastUserVisibleSummary = "Observed idle-like activity."
	case "sleeping":
		agent.Status = core.AgentStatusSleeping
		agent.StatusConfidence = 0.34
		agent.StatusReason = "Sleeping-like output detected."
		agent.LastUserVisibleSummary = "Observed sleeping-like activity."
	default:
		if continuationLine {
			agent.Status = core.AgentStatusThinking
			agent.StatusConfidence = 0.42
			agent.StatusReason = "Continuation-like output detected."
			agent.LastUserVisibleSummary = "Observed continuing output."
		} else if age <= 2*time.Minute {
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
	explicitDisconnectedSignals []string,
	explicitDisconnectedNegations []string,
	explicitDoneSignals []string,
	genericDoneSignals []string,
	genericDoneNegations []string,
	explicitInputSignals []string,
	genericInputNegations []string,
	explicitToolSignals []string,
	explicitReadingSignals []string,
	explicitThinkingSignals []string,
	explicitIdleSignals []string,
	explicitSleepingSignals []string,
	explicitBootingSignals []string,
) (kind string, signalText string) {
	if latestLine != "" {
		if kind := classifyObservedText(latestLine, explicitErrorSignals, genericErrorSignals, genericErrorNegations, explicitDisconnectedSignals, explicitDisconnectedNegations, explicitDoneSignals, genericDoneSignals, genericDoneNegations, explicitInputSignals, genericInputNegations, explicitToolSignals, explicitReadingSignals, explicitThinkingSignals, explicitIdleSignals, explicitSleepingSignals, explicitBootingSignals); kind != "" {
			return kind, latestLine
		}
		if indicatesObservedContinuation(latestLine) {
			return "", latestLine
		}
	}
	return classifyObservedText(fullContent, explicitErrorSignals, genericErrorSignals, genericErrorNegations, explicitDisconnectedSignals, explicitDisconnectedNegations, explicitDoneSignals, genericDoneSignals, genericDoneNegations, explicitInputSignals, genericInputNegations, explicitToolSignals, explicitReadingSignals, explicitThinkingSignals, explicitIdleSignals, explicitSleepingSignals, explicitBootingSignals), fullContent
}

func classifyObservedText(
	text string,
	explicitErrorSignals []string,
	genericErrorSignals []string,
	genericErrorNegations []string,
	explicitDisconnectedSignals []string,
	explicitDisconnectedNegations []string,
	explicitDoneSignals []string,
	genericDoneSignals []string,
	genericDoneNegations []string,
	explicitInputSignals []string,
	genericInputNegations []string,
	explicitToolSignals []string,
	explicitReadingSignals []string,
	explicitThinkingSignals []string,
	explicitIdleSignals []string,
	explicitSleepingSignals []string,
	explicitBootingSignals []string,
) string {
	switch {
	case containsAny(text, explicitBootingSignals...):
		return "booting"
	case containsSignal(text, explicitDisconnectedSignals, explicitDisconnectedNegations):
		return "disconnected"
	case containsSignal(text, explicitErrorSignals, nil) || containsSignal(text, genericErrorSignals, genericErrorNegations):
		return "error"
	case containsSignal(text, explicitDoneSignals, genericDoneNegations) || containsSignal(text, genericDoneSignals, genericDoneNegations):
		return "done"
	case containsSignal(text, explicitInputSignals, genericInputNegations) || (strings.Contains(text, "?") && !containsAny(text, genericInputNegations...)):
		return "waiting_input"
	case containsAny(text, explicitToolSignals...):
		return "running_tool"
	case containsAny(text, explicitReadingSignals...):
		return "reading"
	case containsAny(text, explicitThinkingSignals...):
		return "thinking"
	case containsAny(text, explicitIdleSignals...):
		return "idle"
	case containsAny(text, explicitSleepingSignals...):
		return "sleeping"
	default:
		return ""
	}
}

func indicatesObservedContinuation(text string) bool {
	return containsAny(
		text,
		"continuing",
		"still working",
		"working on",
		"in progress",
		"processing",
		"retrying",
		"resuming",
	)
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
