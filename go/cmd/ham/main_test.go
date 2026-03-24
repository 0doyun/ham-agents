package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/core"
)

func TestParseHourFlagAcceptsValidQuietHours(t *testing.T) {
	t.Parallel()

	hour, err := parseHourFlag("--quiet-start-hour=22", "--quiet-start-hour=")
	if err != nil {
		t.Fatalf("parse hour flag: %v", err)
	}
	if hour != 22 {
		t.Fatalf("expected hour 22, got %d", hour)
	}
}

func TestParseHourFlagRejectsInvalidQuietHours(t *testing.T) {
	t.Parallel()

	testCases := []string{
		"--quiet-start-hour=-1",
		"--quiet-start-hour=24",
		"--quiet-start-hour=nope",
	}

	for _, argument := range testCases {
		argument := argument
		t.Run(argument, func(t *testing.T) {
			t.Parallel()

			if _, err := parseHourFlag(argument, "--quiet-start-hour="); err == nil {
				t.Fatalf("expected %q to fail", argument)
			}
		})
	}
}

func TestParseStopInputAcceptsAgentIDAndJSONFlag(t *testing.T) {
	t.Parallel()

	agentID, asJSON, err := parseStopInput([]string{"agent-1", "--json"})
	if err != nil {
		t.Fatalf("parse stop input: %v", err)
	}
	if agentID != "agent-1" {
		t.Fatalf("expected agent-1, got %q", agentID)
	}
	if !asJSON {
		t.Fatalf("expected json flag to be true")
	}
}

func TestParseStopInputRejectsMissingAgentID(t *testing.T) {
	t.Parallel()

	if _, _, err := parseStopInput([]string{"--json"}); err == nil {
		t.Fatalf("expected missing agent id to fail")
	}
}

func TestParseLogsInputAcceptsAgentIDLimitAndJSON(t *testing.T) {
	t.Parallel()

	agentID, limit, asJSON, err := parseLogsInput([]string{"--json", "--limit", "7", "agent-1"})
	if err != nil {
		t.Fatalf("parse logs input: %v", err)
	}
	if agentID != "agent-1" {
		t.Fatalf("expected agent-1, got %q", agentID)
	}
	if limit != 7 {
		t.Fatalf("expected limit 7, got %d", limit)
	}
	if !asJSON {
		t.Fatalf("expected json flag to be true")
	}
}

func TestParseLogsInputRejectsMissingAgentID(t *testing.T) {
	t.Parallel()

	if _, _, _, err := parseLogsInput([]string{"--limit", "5"}); err == nil {
		t.Fatalf("expected missing agent id to fail")
	}
}

func TestParseLogsInputRejectsNonPositiveLimit(t *testing.T) {
	t.Parallel()

	if _, _, _, err := parseLogsInput([]string{"--limit", "0", "agent-1"}); err == nil {
		t.Fatalf("expected zero limit to fail")
	}
}

func TestChooseAttachableSessionReturnsOnlySessionWithoutPrompt(t *testing.T) {
	t.Parallel()

	session, err := chooseAttachableSession(strings.NewReader(""), &strings.Builder{}, []core.AttachableSession{
		{ID: "abc", Title: "Claude", SessionRef: "iterm2://session/abc"},
	})
	if err != nil {
		t.Fatalf("choose attachable session: %v", err)
	}
	if session.ID != "abc" {
		t.Fatalf("expected abc, got %q", session.ID)
	}
}

func TestChooseAttachableSessionReadsNumericSelection(t *testing.T) {
	t.Parallel()

	var output strings.Builder
	session, err := chooseAttachableSession(strings.NewReader("2\n"), &output, []core.AttachableSession{
		{ID: "abc", Title: "Claude", SessionRef: "iterm2://session/abc", IsActive: true},
		{ID: "xyz", Title: "Shell", SessionRef: "iterm2://session/xyz"},
	})
	if err != nil {
		t.Fatalf("choose attachable session: %v", err)
	}
	if session.ID != "xyz" {
		t.Fatalf("expected xyz, got %q", session.ID)
	}
	if !strings.Contains(output.String(), "Select iTerm session") {
		t.Fatalf("expected prompt output, got %q", output.String())
	}
}

func TestEventsAfterIDForDisplayFiltersOlderEvents(t *testing.T) {
	t.Parallel()

	events := []core.Event{
		{ID: "event-1"},
		{ID: "event-2"},
		{ID: "event-3"},
	}

	filtered := eventsAfterIDForDisplay(events, "event-1", 20)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 events, got %d", len(filtered))
	}
	if filtered[0].ID != "event-2" || filtered[1].ID != "event-3" {
		t.Fatalf("unexpected filtered events %#v", filtered)
	}
}

func TestEventsForAgentFiltersAndLimitsToMostRecentMatches(t *testing.T) {
	t.Parallel()

	filtered := eventsForAgent([]core.Event{
		{ID: "event-1", AgentID: "agent-1"},
		{ID: "event-2", AgentID: "agent-2"},
		{ID: "event-3", AgentID: "agent-1"},
		{ID: "event-4", AgentID: "agent-1"},
	}, "agent-1", 2)

	if len(filtered) != 2 {
		t.Fatalf("expected 2 events, got %d", len(filtered))
	}
	if filtered[0].ID != "event-3" || filtered[1].ID != "event-4" {
		t.Fatalf("unexpected filtered events %#v", filtered)
	}
}

func TestAgentLogFetchLimitHasFloor(t *testing.T) {
	t.Parallel()

	if limit := agentLogFetchLimit(2); limit != 100 {
		t.Fatalf("expected floor 100, got %d", limit)
	}
	if limit := agentLogFetchLimit(15); limit != 150 {
		t.Fatalf("expected scaled limit 150, got %d", limit)
	}
}

func TestPrintEventsWritesJSONLinesWhenRequested(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	events := []core.Event{
		{
			ID:         "event-1",
			AgentID:    "agent-1",
			Type:       core.EventTypeAgentRegistered,
			Summary:    "Registered.",
			OccurredAt: time.Unix(1, 0).UTC(),
		},
	}

	if err := printEvents(&output, events, true); err != nil {
		t.Fatalf("print events: %v", err)
	}
	if !strings.Contains(output.String(), `"id":"event-1"`) {
		t.Fatalf("expected json line output, got %q", output.String())
	}
}

func TestFormatAgentListLineIncludesConfidenceAndReason(t *testing.T) {
	t.Parallel()

	line := formatAgentListLine(core.Agent{
		ID:               "agent-1",
		DisplayName:      "observer",
		Provider:         "log",
		Mode:             core.AgentModeObserved,
		Status:           core.AgentStatusWaitingInput,
		StatusConfidence: 0.45,
		StatusReason:     "Question-like output detected.",
	})

	if !strings.Contains(line, "likely waiting_input") {
		t.Fatalf("expected softened status in line %q", line)
	}
	if !strings.Contains(line, "low 45%") {
		t.Fatalf("expected confidence label in line %q", line)
	}
	if !strings.Contains(line, "Question-like output detected.") {
		t.Fatalf("expected reason in line %q", line)
	}
}

func TestRenderAgentsHumanReadableIncludesConfidenceAndReason(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	err := renderAgents(&output, []core.Agent{
		{
			ID:               "agent-1",
			DisplayName:      "observer",
			Provider:         "log",
			Mode:             core.AgentModeObserved,
			Status:           core.AgentStatusWaitingInput,
			StatusConfidence: 0.45,
			StatusReason:     "Question-like output detected.",
		},
	}, false)
	if err != nil {
		t.Fatalf("render agents: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected summary plus one agent line, got %d from %q", len(lines), output.String())
	}
	if !strings.Contains(lines[0], "summary total=1 attention=1 managed=0 attached=0 observed=1") {
		t.Fatalf("expected summary line in output %q", output.String())
	}
	line := lines[1]
	if !strings.Contains(line, "likely waiting_input") {
		t.Fatalf("expected softened status in line %q", output.String())
	}
	if !strings.Contains(line, "low 45%") {
		t.Fatalf("expected confidence label in line %q", output.String())
	}
	if !strings.Contains(line, "Question-like output detected.") {
		t.Fatalf("expected reason in line %q", output.String())
	}
}

func TestRenderAgentsJSONKeepsMachineReadableFields(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	err := renderAgents(&output, []core.Agent{
		{
			ID:               "agent-1",
			DisplayName:      "observer",
			Provider:         "log",
			Mode:             core.AgentModeObserved,
			Status:           core.AgentStatusWaitingInput,
			StatusConfidence: 0.45,
			StatusReason:     "Question-like output detected.",
		},
		{
			ID:               "agent-2",
			DisplayName:      "broken",
			Status:           core.AgentStatusError,
			StatusConfidence: 0.95,
			StatusReason:     "Tool failed.",
		},
	}, true)
	if err != nil {
		t.Fatalf("render agents json: %v", err)
	}

	payload := output.String()
	if !strings.Contains(payload, `"status": "waiting_input"`) {
		t.Fatalf("expected raw status field in payload %q", payload)
	}
	if !strings.Contains(payload, `"status_confidence": 0.45`) {
		t.Fatalf("expected raw confidence field in payload %q", payload)
	}
	firstIndex := strings.Index(payload, `"id": "agent-1"`)
	secondIndex := strings.Index(payload, `"id": "agent-2"`)
	if firstIndex == -1 || secondIndex == -1 || firstIndex > secondIndex {
		t.Fatalf("expected json output to preserve input order, got %q", payload)
	}
	if strings.Contains(payload, "likely waiting_input") || strings.Contains(payload, "low 45%") {
		t.Fatalf("expected json output to avoid human wording, got %q", payload)
	}
	if strings.Contains(payload, "summary total=") {
		t.Fatalf("expected json output to avoid human summary wording, got %q", payload)
	}
}

func TestRenderAgentsHumanReadablePrioritizesAttentionAgents(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	err := renderAgents(&output, []core.Agent{
		{
			ID:          "agent-1",
			DisplayName: "calm",
			Status:      core.AgentStatusThinking,
			LastEventAt: time.Unix(1, 0).UTC(),
		},
		{
			ID:               "agent-2",
			DisplayName:      "waiting",
			Status:           core.AgentStatusWaitingInput,
			StatusConfidence: 0.65,
			StatusReason:     "Needs approval.",
			LastEventAt:      time.Unix(2, 0).UTC(),
		},
		{
			ID:               "agent-3",
			DisplayName:      "broken",
			Status:           core.AgentStatusError,
			StatusConfidence: 0.9,
			StatusReason:     "Tool failed.",
			LastEventAt:      time.Unix(3, 0).UTC(),
		},
	}, false)
	if err != nil {
		t.Fatalf("render agents: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected summary plus 3 lines, got %d from %q", len(lines), output.String())
	}
	if !strings.Contains(lines[0], "summary total=3 attention=2 managed=0 attached=0 observed=0") {
		t.Fatalf("expected summary line, got %q", output.String())
	}
	if !strings.Contains(lines[1], "broken") || !strings.Contains(lines[2], "waiting") || !strings.Contains(lines[3], "calm") {
		t.Fatalf("expected attention-first ordering, got %q", output.String())
	}
}

func TestRenderAgentsHumanReadableUsesRecencyWithinSameSeverity(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	err := renderAgents(&output, []core.Agent{
		{
			ID:          "agent-1",
			DisplayName: "older",
			Status:      core.AgentStatusWaitingInput,
			LastEventAt: time.Unix(1, 0).UTC(),
		},
		{
			ID:          "agent-2",
			DisplayName: "newer",
			Status:      core.AgentStatusWaitingInput,
			LastEventAt: time.Unix(2, 0).UTC(),
		},
	}, false)
	if err != nil {
		t.Fatalf("render agents: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected summary plus 2 lines, got %d from %q", len(lines), output.String())
	}
	if !strings.Contains(lines[1], "newer") || !strings.Contains(lines[2], "older") {
		t.Fatalf("expected newer same-severity urgent agent first, got %q", output.String())
	}
}

func TestFormatAgentListSummaryIncludesModeAndAttentionBreakdown(t *testing.T) {
	t.Parallel()

	summary := formatAgentListSummary([]core.Agent{
		{Mode: core.AgentModeManaged, Status: core.AgentStatusThinking},
		{Mode: core.AgentModeAttached, Status: core.AgentStatusError},
		{Mode: core.AgentModeObserved, Status: core.AgentStatusWaitingInput},
	})

	if summary != "summary total=3 attention=2 managed=1 attached=1 observed=1" {
		t.Fatalf("unexpected summary %q", summary)
	}
}

func TestCountAttentionAgentsCountsWaitingErrorDisconnected(t *testing.T) {
	t.Parallel()

	count := countAttentionAgents([]core.Agent{
		{Status: core.AgentStatusThinking},
		{Status: core.AgentStatusWaitingInput},
		{Status: core.AgentStatusError},
		{Status: core.AgentStatusDisconnected},
	})

	if count != 3 {
		t.Fatalf("expected attention count 3, got %d", count)
	}
}

func TestRenderStatusHumanReadableIncludesAttentionSummary(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	err := renderStatus(&output, core.RuntimeSnapshot{
		Agents: []core.Agent{
			{Status: core.AgentStatusThinking},
			{Status: core.AgentStatusWaitingInput},
			{Status: core.AgentStatusDone},
			{Status: core.AgentStatusError},
		},
	}, false)
	if err != nil {
		t.Fatalf("render status: %v", err)
	}

	line := output.String()
	if !strings.Contains(line, "total=4") || !strings.Contains(line, "running=1") || !strings.Contains(line, "waiting=1") || !strings.Contains(line, "done=1") {
		t.Fatalf("expected count summary in line %q", line)
	}
	if !strings.Contains(line, "attention=2") {
		t.Fatalf("expected attention summary in line %q", line)
	}
	if !strings.Contains(line, "attention_breakdown error=1 waiting_input=1 disconnected=0") {
		t.Fatalf("expected attention breakdown in output %q", line)
	}
}

func TestRenderStatusHumanReadableIncludesUrgentAgentDetails(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	err := renderStatus(&output, core.RuntimeSnapshot{
		Agents: []core.Agent{
			{
				ID:               "agent-1",
				DisplayName:      "disconnected",
				Status:           core.AgentStatusDisconnected,
				StatusConfidence: 0.8,
				StatusReason:     "Session vanished.",
				LastEventAt:      time.Unix(1, 0).UTC(),
			},
			{
				ID:               "agent-2",
				DisplayName:      "waiting",
				Status:           core.AgentStatusWaitingInput,
				StatusConfidence: 0.55,
				StatusReason:     "Needs approval.",
				LastEventAt:      time.Unix(2, 0).UTC(),
			},
			{
				ID:               "agent-3",
				DisplayName:      "erroring",
				Status:           core.AgentStatusError,
				StatusConfidence: 0.95,
				StatusReason:     "Tool failed.",
				LastEventAt:      time.Unix(3, 0).UTC(),
			},
		},
	}, false)
	if err != nil {
		t.Fatalf("render status: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 5 {
		t.Fatalf("expected summary, breakdown, and 3 urgent lines, got %d from %q", len(lines), output.String())
	}
	if !strings.Contains(lines[1], "attention_breakdown error=1 waiting_input=1 disconnected=1") {
		t.Fatalf("expected breakdown line, got %q", output.String())
	}
	if !strings.Contains(lines[2], "erroring") || !strings.Contains(lines[3], "waiting") || !strings.Contains(lines[4], "disconnected") {
		t.Fatalf("expected severity-ordered urgent details, got %q", output.String())
	}
	if !strings.Contains(lines[2], "Tool failed.") || !strings.Contains(lines[3], "Needs approval.") {
		t.Fatalf("expected reasons in urgent details, got %q", output.String())
	}
}

func TestRenderStatusHumanReadableUsesRecencyWithinSameSeverity(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	err := renderStatus(&output, core.RuntimeSnapshot{
		Agents: []core.Agent{
			{
				ID:          "agent-1",
				DisplayName: "older",
				Status:      core.AgentStatusWaitingInput,
				LastEventAt: time.Unix(1, 0).UTC(),
			},
			{
				ID:          "agent-2",
				DisplayName: "newer",
				Status:      core.AgentStatusWaitingInput,
				LastEventAt: time.Unix(2, 0).UTC(),
			},
		},
	}, false)
	if err != nil {
		t.Fatalf("render status: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected summary, breakdown, and 2 urgent lines, got %d from %q", len(lines), output.String())
	}
	if !strings.Contains(lines[2], "newer") || !strings.Contains(lines[3], "older") {
		t.Fatalf("expected newer same-severity urgent detail first, got %q", output.String())
	}
}

func TestRenderStatusJSONKeepsMachineReadableShape(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	err := renderStatus(&output, core.RuntimeSnapshot{
		Agents: []core.Agent{
			{Status: core.AgentStatusThinking},
			{Status: core.AgentStatusWaitingInput},
			{Status: core.AgentStatusDone},
		},
		GeneratedAt: time.Unix(10, 0).UTC(),
	}, true)
	if err != nil {
		t.Fatalf("render status json: %v", err)
	}

	payload := output.String()
	if !strings.Contains(payload, `"total": 3`) || !strings.Contains(payload, `"running": 1`) || !strings.Contains(payload, `"waiting": 1`) || !strings.Contains(payload, `"done": 1`) {
		t.Fatalf("expected machine-readable counts in payload %q", payload)
	}
	if strings.Contains(payload, "attention=") || strings.Contains(payload, "attention_breakdown") || strings.Contains(payload, "\n!") {
		t.Fatalf("expected json payload to avoid human summary wording, got %q", payload)
	}
}

func TestRenderStopResultHumanReadable(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	if err := renderStopResult(&output, "agent-1", false); err != nil {
		t.Fatalf("render stop result: %v", err)
	}

	if got := output.String(); got != "stopped tracking agent-1\n" {
		t.Fatalf("unexpected human stop output %q", got)
	}
}

func TestRenderStopResultJSON(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	if err := renderStopResult(&output, "agent-1", true); err != nil {
		t.Fatalf("render stop result json: %v", err)
	}

	payload := output.String()
	if !strings.Contains(payload, `"removed": "agent-1"`) {
		t.Fatalf("expected removed field in payload %q", payload)
	}
	if strings.Contains(payload, "stopped tracking") {
		t.Fatalf("expected json stop output to avoid human wording, got %q", payload)
	}
}
