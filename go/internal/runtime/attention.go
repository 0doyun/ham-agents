package runtime

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ham-agents/ham-agents/go/internal/core"
)

func snapshotAttentionBreakdown(agents []core.Agent) core.AttentionBreakdown {
	var breakdown core.AttentionBreakdown
	for _, agent := range agents {
		switch agent.Status {
		case core.AgentStatusError:
			breakdown.Error++
		case core.AgentStatusWaitingInput:
			breakdown.WaitingInput++
		case core.AgentStatusDisconnected:
			breakdown.Disconnected++
		}
	}
	return breakdown
}

func snapshotAttentionOrder(agents []core.Agent) []string {
	attentionAgents := make([]core.Agent, 0, len(agents))
	for _, agent := range agents {
		if core.RequiresAttention(agent.Status) {
			attentionAgents = append(attentionAgents, agent)
		}
	}

	sort.SliceStable(attentionAgents, func(i, j int) bool {
		left := attentionAgents[i]
		right := attentionAgents[j]

		leftSeverity := core.AttentionSeverity(left.Status)
		rightSeverity := core.AttentionSeverity(right.Status)
		if leftSeverity != rightSeverity {
			return leftSeverity < rightSeverity
		}
		if !left.LastEventAt.Equal(right.LastEventAt) {
			return left.LastEventAt.After(right.LastEventAt)
		}
		if left.DisplayName != right.DisplayName {
			return left.DisplayName < right.DisplayName
		}
		return left.ID < right.ID
	})

	orderedIDs := make([]string, 0, len(attentionAgents))
	for _, agent := range attentionAgents {
		orderedIDs = append(orderedIDs, agent.ID)
	}
	return orderedIDs
}

func snapshotAttentionSubtitles(agents []core.Agent) map[string]string {
	subtitles := map[string]string{}
	for _, agent := range agents {
		if core.RequiresAttention(agent.Status) {
			subtitles[agent.ID] = attentionSubtitle(agent)
		}
	}
	return subtitles
}

func attentionSubtitle(agent core.Agent) string {
	status := core.HumanAgentStatusLabel(agent.Status)
	if agent.StatusConfidence < 0.5 {
		status = "likely " + status
	}

	confidenceLevel := "low"
	switch {
	case agent.StatusConfidence >= 0.8:
		confidenceLevel = "high"
	case agent.StatusConfidence >= 0.5:
		confidenceLevel = "medium"
	}

	if trimmed := strings.TrimSpace(agent.StatusReason); trimmed != "" {
		return fmt.Sprintf("%s · %s confidence · %s", status, confidenceLevel, trimmed)
	}
	return fmt.Sprintf("%s · %s confidence", status, confidenceLevel)
}
