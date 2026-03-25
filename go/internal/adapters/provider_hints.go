package adapters

import (
	"encoding/json"
	"strings"

	"github.com/ham-agents/ham-agents/go/internal/core"
)

func ManagedProviderHint(provider string, line string, isStderr bool) (core.AgentStatus, string, string, bool) {
	if strings.EqualFold(strings.TrimSpace(provider), "claude") {
		var payload map[string]any
		if json.Unmarshal([]byte(line), &payload) == nil {
			typeValue, _ := payload["type"].(string)
			switch strings.TrimSpace(typeValue) {
			case "tool_use":
				return core.AgentStatusRunningTool, "Claude reported tool use.", "Claude is running a tool.", true
			case "assistant", "message":
				return core.AgentStatusThinking, "Claude reported assistant output.", strings.TrimSpace(line), true
			case "error":
				return core.AgentStatusError, "Claude reported an error.", strings.TrimSpace(line), true
			}
		}
	}
	if isStderr && (strings.Contains(strings.ToLower(line), "error") || strings.Contains(strings.ToLower(line), "failed")) {
		return core.AgentStatusError, "Managed process emitted error output.", strings.TrimSpace(line), true
	}
	return "", "", "", false
}
