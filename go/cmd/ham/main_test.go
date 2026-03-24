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

	line := output.String()
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
	if strings.Contains(payload, "likely waiting_input") || strings.Contains(payload, "low 45%") {
		t.Fatalf("expected json output to avoid human wording, got %q", payload)
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
	if strings.Contains(payload, "attention=") {
		t.Fatalf("expected json payload to avoid human summary wording, got %q", payload)
	}
}
