package main

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/core"
)

func renderStopResult(out io.Writer, agentID string, asJSON bool) error {
	if asJSON {
		return writeJSONTo(out, map[string]any{
			"stopped": agentID,
		})
	}

	_, err := fmt.Fprintf(out, "stopped %s\n", agentID)
	return err
}

func renderDetachResult(out io.Writer, agentID string, asJSON bool) error {
	if asJSON {
		return writeJSONTo(out, map[string]any{
			"detached": agentID,
		})
	}

	_, err := fmt.Fprintf(out, "detached %s\n", agentID)
	return err
}

func agentLogFetchLimit(limit int) int {
	fetchLimit := limit * 10
	if fetchLimit < 100 {
		return 100
	}
	return fetchLimit
}

func renderAgents(out io.Writer, agents []core.Agent, asJSON bool) error {
	if asJSON {
		return writeJSONTo(out, agents)
	}

	agents = sortAgentsForHumanList(agents)

	if len(agents) == 0 {
		_, err := fmt.Fprintln(out, "no tracked agents")
		return err
	}
	if _, err := fmt.Fprintf(out, "%s\n", formatAgentListSummary(agents)); err != nil {
		return err
	}

	for _, agent := range agents {
		if _, err := fmt.Fprintln(out, formatAgentListLine(agent)); err != nil {
			return err
		}
	}
	return nil
}

func formatAgentListSummary(agents []core.Agent) string {
	managedCount, attachedCount, observedCount := modeBreakdown(agents)
	return fmt.Sprintf(
		"summary total=%d attention=%d managed=%d attached=%d observed=%d",
		len(agents),
		countAttentionAgents(agents),
		managedCount,
		attachedCount,
		observedCount,
	)
}

func formatAgentListLine(agent core.Agent) string {
	parts := []string{
		agent.ID,
		agent.DisplayName,
		agent.Provider,
		string(agent.Mode),
		formatAgentStatusLabel(agent),
		formatConfidenceLabel(agent.StatusConfidence),
	}
	if reason := strings.TrimSpace(agent.StatusReason); reason != "" {
		parts = append(parts, reason)
	}
	if agent.SubAgentCount > 0 {
		parts = append(parts, fmt.Sprintf("+%d sub", agent.SubAgentCount))
	}
	return strings.Join(parts, "\t")
}

func formatAgentStatusLabel(agent core.Agent) string {
	label := humanizeAgentStatus(agent.Status)
	if agent.StatusConfidence < 0.5 {
		return "likely " + label
	}
	return label
}

func humanizeAgentStatus(status core.AgentStatus) string {
	return core.HumanAgentStatusLabel(status)
}

func formatConfidenceLabel(confidence float64) string {
	percent := int((confidence * 100) + 0.5)
	level := "low"
	switch {
	case confidence >= 0.8:
		level = "high"
	case confidence >= 0.5:
		level = "medium"
	}
	return fmt.Sprintf("%s %d%%", level, percent)
}

func countAttentionAgents(agents []core.Agent) int {
	count := 0
	for _, agent := range agents {
		switch agent.Status {
		case core.AgentStatusError, core.AgentStatusWaitingInput, core.AgentStatusDisconnected:
			count++
		}
	}
	return count
}

func attentionBreakdown(agents []core.Agent) (errorCount, waitingCount, disconnectedCount int) {
	for _, agent := range agents {
		switch agent.Status {
		case core.AgentStatusError:
			errorCount++
		case core.AgentStatusWaitingInput:
			waitingCount++
		case core.AgentStatusDisconnected:
			disconnectedCount++
		}
	}
	return errorCount, waitingCount, disconnectedCount
}

func modeBreakdown(agents []core.Agent) (managedCount, attachedCount, observedCount int) {
	for _, agent := range agents {
		switch agent.Mode {
		case core.AgentModeManaged:
			managedCount++
		case core.AgentModeAttached:
			attachedCount++
		case core.AgentModeObserved:
			observedCount++
		}
	}
	return managedCount, attachedCount, observedCount
}

func buildFilteredSnapshot(agents []core.Agent, generatedAt time.Time) core.RuntimeSnapshot {
	errorCount, waitingCount, disconnectedCount := attentionBreakdown(agents)

	return core.RuntimeSnapshot{
		Agents:         agents,
		GeneratedAt:    generatedAt,
		AttentionCount: countAttentionAgents(agents),
		AttentionBreakdown: core.AttentionBreakdown{
			Error:        errorCount,
			WaitingInput: waitingCount,
			Disconnected: disconnectedCount,
		},
		AttentionOrder:     buildAttentionOrder(agents),
		AttentionSubtitles: buildAttentionSubtitles(agents),
	}
}

func buildAttentionOrder(agents []core.Agent) []string {
	urgent := urgentAgents(agents)
	ordered := make([]string, 0, len(urgent))
	for _, agent := range urgent {
		ordered = append(ordered, agent.ID)
	}
	return ordered
}

func buildAttentionSubtitles(agents []core.Agent) map[string]string {
	subtitles := make(map[string]string)
	for _, agent := range agents {
		if !core.RequiresAttention(agent.Status) {
			continue
		}
		subtitles[agent.ID] = buildAttentionSubtitle(agent)
	}
	return subtitles
}

func buildAttentionSubtitle(agent core.Agent) string {
	status := formatAgentStatusLabel(agent)
	confidence := strings.SplitN(formatConfidenceLabel(agent.StatusConfidence), " ", 2)[0]
	if reason := strings.TrimSpace(agent.StatusReason); reason != "" {
		return fmt.Sprintf("%s · %s confidence · %s", status, confidence, reason)
	}
	return fmt.Sprintf("%s · %s confidence", status, confidence)
}

func renderStatus(out io.Writer, snapshot core.RuntimeSnapshot, asJSON bool) error {
	if asJSON {
		return writeJSONTo(out, map[string]any{
			"total":               snapshot.TotalCount(),
			"running":             snapshot.RunningCount(),
			"waiting":             snapshot.WaitingCount(),
			"done":                snapshot.DoneCount(),
			"attention_count":     snapshot.AttentionCount,
			"attention_breakdown": snapshot.AttentionBreakdown,
			"attention_order":     snapshot.AttentionOrder,
			"attention_subtitles": snapshot.AttentionSubtitles,
			"generatedAt":         snapshot.GeneratedAt,
		})
	}

	attentionCount := countAttentionAgents(snapshot.Agents)
	if _, err := fmt.Fprintf(
		out,
		"total=%d running=%d waiting=%d done=%d attention=%d\n",
		snapshot.TotalCount(),
		snapshot.RunningCount(),
		snapshot.WaitingCount(),
		snapshot.DoneCount(),
		attentionCount,
	); err != nil {
		return err
	}
	if attentionCount > 0 {
		errorCount, waitingCount, disconnectedCount := attentionBreakdown(snapshot.Agents)
		if _, err := fmt.Fprintf(
			out,
			"attention_breakdown error=%d needs_input=%d disconnected=%d\n",
			errorCount,
			waitingCount,
			disconnectedCount,
		); err != nil {
			return err
		}
	}

	for _, agent := range urgentAgents(snapshot.Agents) {
		if _, err := fmt.Fprintf(out, "! %s\n", formatAgentListLine(agent)); err != nil {
			return err
		}
	}
	return nil
}

func urgentAgents(agents []core.Agent) []core.Agent {
	filtered := make([]core.Agent, 0, len(agents))
	for _, agent := range agents {
		if !requiresAttention(agent.Status) {
			continue
		}
		filtered = append(filtered, agent)
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		left := filtered[i]
		right := filtered[j]

		leftSeverity := attentionSeverity(left.Status)
		rightSeverity := attentionSeverity(right.Status)
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

	return filtered
}

func sortAgentsForHumanList(agents []core.Agent) []core.Agent {
	sorted := append([]core.Agent(nil), agents...)
	sort.SliceStable(sorted, func(i, j int) bool {
		left := sorted[i]
		right := sorted[j]

		leftAttention := requiresAttention(left.Status)
		rightAttention := requiresAttention(right.Status)
		if leftAttention != rightAttention {
			return leftAttention
		}

		leftSeverity := attentionSeverity(left.Status)
		rightSeverity := attentionSeverity(right.Status)
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
	return sorted
}

func requiresAttention(status core.AgentStatus) bool {
	return core.RequiresAttention(status)
}

func attentionSeverity(status core.AgentStatus) int {
	return core.AttentionSeverity(status)
}

func printEvents(out io.Writer, events []core.Event, asJSON bool) error {
	if asJSON {
		if len(events) == 0 {
			return writeJSONTo(out, []core.Event{})
		}
		for _, event := range events {
			payload, err := json.Marshal(event)
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintf(out, "%s\n", payload); err != nil {
				return err
			}
		}
		return nil
	}

	if len(events) == 0 {
		_, err := fmt.Fprintln(out, "no events")
		return err
	}
	for _, event := range events {
		if _, err := fmt.Fprintf(out, "%s\t%s\t%s\t%s\n", event.OccurredAt.Format(time.RFC3339), event.Type, event.AgentID, eventDisplaySummary(event)); err != nil {
			return err
		}
	}
	return nil
}

func eventDisplaySummary(event core.Event) string {
	if summary := strings.TrimSpace(event.PresentationSummary); summary != "" {
		return sanitizeSensitiveText(summary)
	}
	if reason := strings.TrimSpace(event.LifecycleReason); reason != "" {
		if event.LifecycleConfidence > 0 && event.LifecycleConfidence < 0.5 {
			return sanitizeSensitiveText(reason) + " (low confidence)"
		}
		return sanitizeSensitiveText(reason)
	}
	return sanitizeSensitiveText(event.Summary)
}

func eventsForAgent(events []core.Event, agentID string, limit int) []core.Event {
	filtered := make([]core.Event, 0, len(events))
	for _, event := range events {
		if event.AgentID != agentID {
			continue
		}
		filtered = append(filtered, event)
	}
	if len(filtered) <= limit {
		return filtered
	}
	return filtered[len(filtered)-limit:]
}

func eventsAfterIDForDisplay(events []core.Event, afterEventID string, limit int) []core.Event {
	return core.EventsAfterID(events, afterEventID, limit)
}

var secretAssignmentPattern = regexp.MustCompile(`(?i)\b([A-Z0-9_]*(token|secret|password|api[_-]?key)[A-Z0-9_]*)=([^\s]+)`)
var homePathPattern = regexp.MustCompile(`/Users/[^/\s]+`)

func sanitizeSensitiveText(value string) string {
	sanitized := secretAssignmentPattern.ReplaceAllString(value, `$1=***`)
	sanitized = homePathPattern.ReplaceAllString(sanitized, `/Users/***`)
	return sanitized
}
