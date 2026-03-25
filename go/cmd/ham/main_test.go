package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
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

func TestResolveUICommandPrefersEnvironmentOverride(t *testing.T) {
	t.Parallel()

	target, printOnly, asJSON, err := resolveUICommand(
		[]string{"--json"},
		func() (string, error) { return "/tmp/ham", nil },
		func(key string) (string, bool) {
			if key == "HAM_UI_EXECUTABLE" {
				return "/tmp/custom/ham-menubar", true
			}
			return "", false
		},
		func() (string, error) { return "/tmp/project", nil },
		func(string) (string, error) { return "", fmt.Errorf("missing") },
	)
	if err != nil {
		t.Fatalf("resolve ui command: %v", err)
	}
	if target.Executable != "/tmp/custom/ham-menubar" {
		t.Fatalf("unexpected target %#v", target)
	}
	if !asJSON || printOnly {
		t.Fatalf("expected json true and print false")
	}
}

func TestResolveUICommandUsesBuildArtifactFallback(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	buildPath := filepath.Join(root, ".build", "arm64-apple-macosx", "debug")
	if err := os.MkdirAll(buildPath, 0o755); err != nil {
		t.Fatalf("mkdir build path: %v", err)
	}
	expectedPath := filepath.Join(buildPath, "ham-menubar")
	if err := os.WriteFile(expectedPath, []byte("binary"), 0o755); err != nil {
		t.Fatalf("write build artifact: %v", err)
	}

	target, printOnly, asJSON, err := resolveUICommand(
		[]string{"--print"},
		func() (string, error) { return "/tmp/ham", nil },
		func(string) (string, bool) { return "", false },
		func() (string, error) { return root, nil },
		func(string) (string, error) { return "", fmt.Errorf("missing") },
	)
	if err != nil {
		t.Fatalf("resolve ui command: %v", err)
	}
	if target.Executable != expectedPath {
		t.Fatalf("unexpected target %#v", target)
	}
	if !printOnly || asJSON {
		t.Fatalf("expected print true and json false")
	}
}

func TestResolveUICommandRejectsUnexpectedArgument(t *testing.T) {
	t.Parallel()

	if _, _, _, err := resolveUICommand(
		[]string{"unexpected"},
		func() (string, error) { return "/tmp/ham", nil },
		func(string) (string, bool) { return "", false },
		func() (string, error) { return "/tmp/project", nil },
		func(string) (string, error) { return "", fmt.Errorf("missing") },
	); err == nil {
		t.Fatalf("expected unexpected ui argument to fail")
	}
}

func TestRunDoctorRejectsUnexpectedArgument(t *testing.T) {
	t.Parallel()

	if err := runDoctor("/tmp/hamd.sock", []string{"unexpected"}); err == nil {
		t.Fatalf("expected unexpected doctor argument to fail")
	}
}

func TestGatherDoctorReportUsesEnvRootAndInspectsPaths(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HAM_AGENTS_HOME", root)

	socketPath := filepath.Join(root, "hamd.sock")
	statePath := filepath.Join(root, "managed-agents.json")
	eventPath := filepath.Join(root, "events.jsonl")
	if err := os.WriteFile(statePath, []byte("{}"), 0o644); err != nil {
		t.Fatalf("write state: %v", err)
	}
	if err := os.WriteFile(eventPath, []byte(""), 0o644); err != nil {
		t.Fatalf("write events: %v", err)
	}

	report, err := gatherDoctorReport(socketPath)
	if err != nil {
		t.Fatalf("gather doctor report: %v", err)
	}

	if report.RootSource != "env" {
		t.Fatalf("expected env root source, got %q", report.RootSource)
	}
	if report.HamAgentsHome != root {
		t.Fatalf("expected HAM_AGENTS_HOME %q, got %q", root, report.HamAgentsHome)
	}
	if report.Socket.Exists || report.Socket.Kind != "missing" {
		t.Fatalf("expected missing socket, got %#v", report.Socket)
	}
	if !report.State.Exists || report.State.Kind != "file" {
		t.Fatalf("expected state file, got %#v", report.State)
	}
	if !report.Events.Exists || report.Events.Kind != "file" {
		t.Fatalf("expected event file, got %#v", report.Events)
	}
	if report.Settings.Exists || report.Settings.Kind != "missing" {
		t.Fatalf("expected missing settings file, got %#v", report.Settings)
	}
}

func TestRenderDoctorReportHumanReadable(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	err := renderDoctorReport(&output, doctorReport{
		RootSource:    "env",
		HamAgentsHome: "/tmp/ham",
		ResolvedRoot:  "/tmp/ham",
		Socket:        doctorPathCheck{Path: "/tmp/ham/hamd.sock", Exists: true, Kind: "unix_socket", Reachable: true},
		State:         doctorPathCheck{Path: "/tmp/ham/managed-agents.json", Exists: true, Kind: "file"},
		Events:        doctorPathCheck{Path: "/tmp/ham/events.jsonl", Exists: false, Kind: "missing"},
		Settings:      doctorPathCheck{Path: "/tmp/ham/settings.json", Exists: true, Kind: "file"},
	}, false)
	if err != nil {
		t.Fatalf("render doctor report: %v", err)
	}

	rendered := output.String()
	if !strings.Contains(rendered, "ham-agents doctor") {
		t.Fatalf("expected header in output %q", rendered)
	}
	if !strings.Contains(rendered, "root_source: env") || !strings.Contains(rendered, "ham_agents_home: /tmp/ham") || !strings.Contains(rendered, "resolved_root: /tmp/ham") {
		t.Fatalf("expected root info in output %q", rendered)
	}
	if !strings.Contains(rendered, "socket: reachable_socket\t/tmp/ham/hamd.sock") {
		t.Fatalf("expected socket line in output %q", rendered)
	}
	if !strings.Contains(rendered, "events: missing\t/tmp/ham/events.jsonl") {
		t.Fatalf("expected missing events line in output %q", rendered)
	}
}

func TestRenderDoctorReportJSON(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	err := renderDoctorReport(&output, doctorReport{
		RootSource:   "default",
		ResolvedRoot: "/tmp/ham-agents",
		Socket:       doctorPathCheck{Path: "/tmp/hamd.sock", Exists: false, Kind: "missing"},
		State:        doctorPathCheck{Path: "/tmp/state.json", Exists: false, Kind: "missing"},
		Events:       doctorPathCheck{Path: "/tmp/events.jsonl", Exists: false, Kind: "missing"},
		Settings:     doctorPathCheck{Path: "/tmp/settings.json", Exists: false, Kind: "missing"},
	}, true)
	if err != nil {
		t.Fatalf("render doctor report json: %v", err)
	}

	payload := output.String()
	if !strings.Contains(payload, `"root_source": "default"`) || !strings.Contains(payload, `"kind": "missing"`) {
		t.Fatalf("expected doctor json fields in payload %q", payload)
	}
	if strings.Contains(payload, "ham-agents doctor") || strings.Contains(payload, "reachable_socket") {
		t.Fatalf("expected json output to avoid human wording, got %q", payload)
	}
}

func TestRenderDoctorReportHumanReadableDefaultRoot(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	err := renderDoctorReport(&output, doctorReport{
		RootSource:   "default",
		ResolvedRoot: "/Users/example/Library/Application Support/ham-agents",
		Socket:       doctorPathCheck{Path: "/tmp/hamd.sock", Exists: false, Kind: "missing"},
		State:        doctorPathCheck{Path: "/tmp/state.json", Exists: false, Kind: "missing"},
		Events:       doctorPathCheck{Path: "/tmp/events.jsonl", Exists: false, Kind: "missing"},
		Settings:     doctorPathCheck{Path: "/tmp/settings.json", Exists: false, Kind: "missing"},
	}, false)
	if err != nil {
		t.Fatalf("render doctor report: %v", err)
	}

	rendered := output.String()
	if !strings.Contains(rendered, "ham_agents_home: (unset)") || !strings.Contains(rendered, "resolved_root: /Users/example/Library/Application Support/ham-agents") {
		t.Fatalf("expected default root output, got %q", rendered)
	}
}

func TestRenderDoctorReportHumanReadableShowsSocketNotListening(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	err := renderDoctorReport(&output, doctorReport{
		RootSource:   "default",
		ResolvedRoot: "/tmp/ham",
		Socket:       doctorPathCheck{Path: "/tmp/ham/hamd.sock", Exists: true, Kind: "unix_socket", Reachable: false},
		State:        doctorPathCheck{Path: "/tmp/ham/managed-agents.json", Exists: false, Kind: "missing"},
		Events:       doctorPathCheck{Path: "/tmp/ham/events.jsonl", Exists: false, Kind: "missing"},
		Settings:     doctorPathCheck{Path: "/tmp/ham/settings.json", Exists: false, Kind: "missing"},
	}, false)
	if err != nil {
		t.Fatalf("render doctor report: %v", err)
	}

	if !strings.Contains(output.String(), "socket: socket_not_listening\t/tmp/ham/hamd.sock") {
		t.Fatalf("expected socket_not_listening output, got %q", output.String())
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
		{ID: "event-1", AgentID: "agent-1", PresentationLabel: "Managed", PresentationEmphasis: "info", PresentationSummary: "Managed session registered.", LifecycleStatus: "booting", LifecycleMode: "managed", LifecycleReason: "Managed launch requested."},
		{ID: "event-2", AgentID: "agent-2"},
		{ID: "event-3", AgentID: "agent-1", PresentationLabel: "Needs Input", PresentationEmphasis: "warning", PresentationSummary: "Needs confirmation.", LifecycleStatus: "waiting_input", LifecycleMode: "observed", LifecycleReason: "Question-like output detected.", LifecycleConfidence: 0.45},
		{ID: "event-4", AgentID: "agent-1", PresentationLabel: "Done", PresentationEmphasis: "positive", PresentationSummary: "Completed successfully.", LifecycleStatus: "done", LifecycleMode: "managed", LifecycleReason: "Completion-like output detected.", LifecycleConfidence: 0.9},
	}, "agent-1", 2)

	if len(filtered) != 2 {
		t.Fatalf("expected 2 events, got %d", len(filtered))
	}
	if filtered[0].ID != "event-3" || filtered[1].ID != "event-4" {
		t.Fatalf("unexpected filtered events %#v", filtered)
	}
	if filtered[0].PresentationLabel != "Needs Input" || filtered[0].PresentationEmphasis != "warning" {
		t.Fatalf("expected filtered event 0 to retain presentation hints %#v", filtered[0])
	}
	if filtered[0].PresentationSummary != "Needs confirmation." {
		t.Fatalf("expected filtered event 0 to retain presentation summary %#v", filtered[0])
	}
	if filtered[0].LifecycleStatus != "waiting_input" || filtered[0].LifecycleMode != "observed" {
		t.Fatalf("expected filtered event 0 to retain lifecycle metadata %#v", filtered[0])
	}
	if filtered[0].LifecycleReason != "Question-like output detected." {
		t.Fatalf("expected filtered event 0 to retain lifecycle reason %#v", filtered[0])
	}
	if filtered[0].LifecycleConfidence != 0.45 {
		t.Fatalf("expected filtered event 0 to retain lifecycle confidence %#v", filtered[0])
	}
	if filtered[1].PresentationLabel != "Done" || filtered[1].PresentationEmphasis != "positive" {
		t.Fatalf("expected filtered event 1 to retain presentation hints %#v", filtered[1])
	}
	if filtered[1].PresentationSummary != "Completed successfully." {
		t.Fatalf("expected filtered event 1 to retain presentation summary %#v", filtered[1])
	}
	if filtered[1].LifecycleStatus != "done" || filtered[1].LifecycleMode != "managed" {
		t.Fatalf("expected filtered event 1 to retain lifecycle metadata %#v", filtered[1])
	}
	if filtered[1].LifecycleReason != "Completion-like output detected." {
		t.Fatalf("expected filtered event 1 to retain lifecycle reason %#v", filtered[1])
	}
	if filtered[1].LifecycleConfidence != 0.9 {
		t.Fatalf("expected filtered event 1 to retain lifecycle confidence %#v", filtered[1])
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
			ID:                   "event-1",
			AgentID:              "agent-1",
			Type:                 core.EventTypeAgentRegistered,
			Summary:              "Registered.",
			OccurredAt:           time.Unix(1, 0).UTC(),
			PresentationLabel:    "Managed",
			PresentationEmphasis: "info",
			PresentationSummary:  "Managed session registered.",
			LifecycleStatus:      "booting",
			LifecycleMode:        "managed",
			LifecycleReason:      "Managed launch requested.",
			LifecycleConfidence:  1,
		},
	}

	if err := printEvents(&output, events, true); err != nil {
		t.Fatalf("print events: %v", err)
	}
	if !strings.Contains(output.String(), `"id":"event-1"`) {
		t.Fatalf("expected json line output, got %q", output.String())
	}
	if !strings.Contains(output.String(), `"presentation_label":"Managed"`) || !strings.Contains(output.String(), `"presentation_emphasis":"info"`) {
		t.Fatalf("expected presentation hints in json line output, got %q", output.String())
	}
	if !strings.Contains(output.String(), `"presentation_summary":"Managed session registered."`) {
		t.Fatalf("expected presentation summary in json line output, got %q", output.String())
	}
	if !strings.Contains(output.String(), `"lifecycle_status":"booting"`) || !strings.Contains(output.String(), `"lifecycle_mode":"managed"`) {
		t.Fatalf("expected lifecycle metadata in json line output, got %q", output.String())
	}
	if !strings.Contains(output.String(), `"lifecycle_reason":"Managed launch requested."`) {
		t.Fatalf("expected lifecycle reason in json line output, got %q", output.String())
	}
	if !strings.Contains(output.String(), `"lifecycle_confidence":1`) {
		t.Fatalf("expected lifecycle confidence in json line output, got %q", output.String())
	}
}

func TestPrintEventsWritesEmptyJSONArrayToProvidedWriter(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	if err := printEvents(&output, []core.Event{}, true); err != nil {
		t.Fatalf("print empty events json: %v", err)
	}
	if strings.TrimSpace(output.String()) != "[]" {
		t.Fatalf("expected empty json array on provided writer, got %q", output.String())
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
		AttentionCount:     1,
		AttentionBreakdown: core.AttentionBreakdown{Error: 0, WaitingInput: 1, Disconnected: 0},
		AttentionOrder:     []string{"agent-2"},
		AttentionSubtitles: map[string]string{"agent-2": "waiting_input · high confidence · Needs confirmation."},
		GeneratedAt:        time.Unix(10, 0).UTC(),
	}, true)
	if err != nil {
		t.Fatalf("render status json: %v", err)
	}

	payload := output.String()
	if !strings.Contains(payload, `"total": 3`) || !strings.Contains(payload, `"running": 1`) || !strings.Contains(payload, `"waiting": 1`) || !strings.Contains(payload, `"done": 1`) {
		t.Fatalf("expected machine-readable counts in payload %q", payload)
	}
	if !strings.Contains(payload, `"attention_count": 1`) || !strings.Contains(payload, `"waiting_input": 1`) || !strings.Contains(payload, `"attention_order": [`) || !strings.Contains(payload, `"attention_subtitles": {`) {
		t.Fatalf("expected attention fields in payload %q", payload)
	}
	if !strings.Contains(payload, `"agent-2": "waiting_input · high confidence · Needs confirmation."`) {
		t.Fatalf("expected attention subtitles in payload %q", payload)
	}
	if strings.Contains(payload, "attention=") || strings.Contains(payload, "\n!") {
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
