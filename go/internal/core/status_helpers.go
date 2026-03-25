package core

import "strings"

func IsRunningStatus(status AgentStatus) bool {
	switch status {
	case AgentStatusBooting, AgentStatusThinking, AgentStatusReading, AgentStatusRunningTool:
		return true
	default:
		return false
	}
}

func RequiresAttention(status AgentStatus) bool {
	switch status {
	case AgentStatusError, AgentStatusWaitingInput, AgentStatusDisconnected:
		return true
	default:
		return false
	}
}

func AttentionSeverity(status AgentStatus) int {
	switch status {
	case AgentStatusError:
		return 0
	case AgentStatusWaitingInput:
		return 1
	case AgentStatusDisconnected:
		return 2
	default:
		return 3
	}
}

func HumanAgentStatusLabel(status AgentStatus) string {
	switch status {
	case AgentStatusWaitingInput:
		return "needs input"
	case AgentStatusRunningTool:
		return "running tool"
	default:
		return strings.ReplaceAll(string(status), "_", " ")
	}
}

func EventsAfterID(events []Event, afterEventID string, limit int) []Event {
	if afterEventID == "" {
		if limit <= 0 || len(events) <= limit {
			return events
		}
		return events[len(events)-limit:]
	}

	start := -1
	for index, event := range events {
		if event.ID == afterEventID {
			start = index + 1
			break
		}
	}
	if start == -1 {
		start = 0
	}
	if start >= len(events) {
		return []Event{}
	}

	followed := events[start:]
	if limit > 0 && len(followed) > limit {
		return followed[len(followed)-limit:]
	}
	return followed
}
