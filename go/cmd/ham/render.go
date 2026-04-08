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

// ANSI color helpers for terminal output.
const (
	ansiReset  = "\033[0m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiBlue   = "\033[34m"
	ansiGray   = "\033[90m"
)

func colorForStatus(status core.AgentStatus) string {
	switch status {
	case core.AgentStatusError:
		return ansiRed
	case core.AgentStatusWaitingInput:
		return ansiYellow
	case core.AgentStatusDisconnected:
		return ansiYellow
	case core.AgentStatusDone:
		return ansiGreen
	case core.AgentStatusRunningTool, core.AgentStatusReading, core.AgentStatusThinking,
		core.AgentStatusWriting, core.AgentStatusSearching, core.AgentStatusSpawning:
		return ansiBlue
	case core.AgentStatusBooting, core.AgentStatusIdle, core.AgentStatusSleeping:
		return ansiGray
	default:
		return ""
	}
}

func colorize(text string, color string) string {
	if color == "" {
		return text
	}
	return color + text + ansiReset
}

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
	statusLabel := formatAgentStatusLabel(agent)
	statusColor := colorForStatus(agent.Status)
	parts := []string{
		agent.ID,
		colorize(displayNameWithOmcMode(agent), statusColor),
		agent.Provider,
		string(agent.Mode),
		colorize(statusLabel, statusColor),
		formatConfidenceLabel(agent.StatusConfidence),
	}
	if reason := strings.TrimSpace(agent.StatusReason); reason != "" {
		parts = append(parts, reason)
	}
	if summary := strings.TrimSpace(agent.LastUserVisibleSummary); summary != "" {
		parts = append(parts, summary)
	}
	if agent.SubAgentCount > 0 {
		parts = append(parts, fmt.Sprintf("+%d sub", agent.SubAgentCount))
	}
	return strings.Join(parts, "\t")
}

func displayNameWithOmcMode(agent core.Agent) string {
	if mode := strings.TrimSpace(agent.OmcMode); mode != "" {
		return fmt.Sprintf("%s [%s]", agent.DisplayName, mode)
	}
	return agent.DisplayName
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
		prefix := colorize("!", colorForStatus(agent.Status))
		if _, err := fmt.Fprintf(out, "%s %s\n", prefix, formatAgentListLine(agent)); err != nil {
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

// renderSessionGraph prints the session tree to out.
// Each node is prefixed with indentation reflecting its depth.
// Format:
//
//	SessionGraph: N agents (M blocked)
//	  - alice [running] d0
//	    - bob [waiting_input] d1
//	      - carol [done] d2
//	  - dan [idle] d0
//
// The simple indented format is used (2 spaces per depth level) so that it is
// unambiguous and easy to test without requiring complex tree-drawing logic.
func renderSessionGraph(out io.Writer, graph core.SessionGraph) error {
	if _, err := fmt.Fprintf(out, "SessionGraph: %d agents (%d blocked)\n", graph.TotalCount, graph.BlockedCount); err != nil {
		return err
	}
	var printNode func(node core.SessionNode) error
	printNode = func(node core.SessionNode) error {
		indent := strings.Repeat("  ", node.Depth)
		name := node.Agent.DisplayName
		if name == "" {
			name = node.Agent.ID
		}
		status := string(node.Agent.Status)
		if _, err := fmt.Fprintf(out, "%s- %s [%s] d%d\n", indent, name, status, node.Depth); err != nil {
			return err
		}
		for _, child := range node.Children {
			if err := printNode(child); err != nil {
				return err
			}
		}
		return nil
	}
	for _, root := range graph.Roots {
		if err := printNode(root); err != nil {
			return err
		}
	}
	return nil
}

func inboxIconFor(itemType core.InboxItemType) string {
	switch itemType {
	case core.InboxItemPermissionRequest:
		return "!"
	case core.InboxItemNotification:
		return "i"
	case core.InboxItemTaskComplete:
		return "v"
	case core.InboxItemError:
		return "x"
	case core.InboxItemStop:
		return "."
	default:
		return "?"
	}
}

func inboxRelativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func renderInboxItems(out io.Writer, items []core.InboxItem) error {
	if len(items) == 0 {
		_, err := fmt.Fprintln(out, "no inbox items")
		return err
	}
	for _, item := range items {
		icon := inboxIconFor(item.Type)
		name := item.AgentName
		if name == "" {
			name = item.AgentID
		}
		_, err := fmt.Fprintf(out, "[%s] %s\t%s\t%s\n", icon, name, item.Summary, inboxRelativeTime(item.OccurredAt))
		if err != nil {
			return err
		}
	}
	return nil
}

// renderCostSummary writes the cost summary response in either text or JSON.
// The text format prints one row per group key with input/output token totals
// and the rolled-up USD value, followed by a Total USD line so the eye can
// verify the rollup matches the per-row sum.
func renderCostSummary(out io.Writer, response costSummaryView, asJSON bool) error {
	if asJSON {
		return writeJSONTo(out, response)
	}

	switch response.GroupBy {
	case "model":
		if _, err := fmt.Fprintln(out, "MODEL\tINPUT\tCACHE_READ\tOUTPUT\tUSD"); err != nil {
			return err
		}
	case "day":
		if _, err := fmt.Fprintln(out, "DAY\tRECORDS\tUSD"); err != nil {
			return err
		}
	case "agent":
		if _, err := fmt.Fprintln(out, "AGENT\tRECORDS\tUSD"); err != nil {
			return err
		}
	}

	for _, row := range response.Rows {
		switch response.GroupBy {
		case "model":
			if _, err := fmt.Fprintf(out, "%s\t%d\t%d\t%d\t$%.4f\n", row.Key, row.InputTokens, row.CacheReadTokens, row.OutputTokens, row.USD); err != nil {
				return err
			}
		default:
			if _, err := fmt.Fprintf(out, "%s\t%d\t$%.4f\n", row.Key, row.RecordCount, row.USD); err != nil {
				return err
			}
		}
	}
	if _, err := fmt.Fprintf(out, "TOTAL\t$%.4f (%d records)\n", response.TotalUSD, response.RecordCount); err != nil {
		return err
	}
	return nil
}

// costSummaryView is the renderer-friendly projection of an ipc.Response with
// the cost.summary fields populated. It is decoupled from ipc.Response so the
// CLI can hold the rendering logic without dragging in the IPC package in
// every test.
type costSummaryView struct {
	GroupBy     string         `json:"group_by"`
	TotalUSD    float64        `json:"total_usd"`
	RecordCount int            `json:"record_count"`
	Rows        []costRow      `json:"rows"`
}

type costRow struct {
	Key             string  `json:"key"`
	USD             float64 `json:"usd"`
	RecordCount     int     `json:"record_count,omitempty"`
	InputTokens     int64   `json:"input_tokens,omitempty"`
	CacheReadTokens int64   `json:"cache_read_tokens,omitempty"`
	OutputTokens    int64   `json:"output_tokens,omitempty"`
}

// buildFilteredGraph builds a SessionGraph from a filtered subset of agents.
func buildFilteredGraph(agents []core.Agent, generatedAt time.Time) core.SessionGraph {
	graph := core.BuildSessionGraph(agents)
	graph.GeneratedAt = generatedAt
	return graph
}
