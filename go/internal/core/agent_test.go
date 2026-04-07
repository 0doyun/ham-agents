package core

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestEvent_JSONRoundTrip_WithNewFields(t *testing.T) {
	original := Event{
		ID:                   "evt-001",
		AgentID:              "agent-abc",
		Type:                 EventTypeAgentStatusUpdated,
		Summary:              "Status updated",
		OccurredAt:           time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
		PresentationLabel:    "Status",
		PresentationEmphasis: "info",
		PresentationSummary:  "Agent status changed",
		LifecycleStatus:      "thinking",
		LifecycleMode:        "managed",
		LifecycleReason:      "Processing user request",
		LifecycleConfidence:  0.95,
		SessionID:            "sess-xyz",
		ParentAgentID:        "parent-001",
		TaskName:             "code-review",
		TaskDesc:             "Review the pull request for correctness",
		ArtifactType:         "file",
		ArtifactRef:          "src/main.go",
		ArtifactData:         "package main\n",
		ToolName:             "Bash",
		ToolInput:            "go build ./...",
		ToolType:             "bash",
		ToolDuration:         1234,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded Event
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if !reflect.DeepEqual(original, decoded) {
		t.Errorf("round-trip mismatch\ngot:  %+v\nwant: %+v", decoded, original)
	}
}

func TestEvent_JSONRoundTrip_OmitemptyWorks(t *testing.T) {
	original := Event{
		ID:         "evt-002",
		AgentID:    "agent-abc",
		Type:       EventTypeAgentRegistered,
		Summary:    "Agent registered",
		OccurredAt: time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	jsonStr := string(data)
	newFields := []string{
		"session_id", "parent_agent_id", "task_name", "task_desc",
		"artifact_type", "artifact_ref", "artifact_data",
		"tool_name", "tool_input", "tool_type", "tool_duration_ms",
	}
	for _, field := range newFields {
		if strings.Contains(jsonStr, `"`+field+`"`) {
			t.Errorf("expected field %q to be omitted when zero, but found in JSON: %s", field, jsonStr)
		}
	}
}

func TestEvent_JSONRoundTrip_LegacyFormat(t *testing.T) {
	legacy := `{
		"id": "evt-legacy",
		"agent_id": "agent-old",
		"type": "agent.registered",
		"summary": "Old agent registered",
		"occurred_at": "2026-01-01T00:00:00Z",
		"presentation_label": "Registered",
		"lifecycle_status": "booting",
		"lifecycle_confidence": 1.0
	}`

	var e Event
	if err := json.Unmarshal([]byte(legacy), &e); err != nil {
		t.Fatalf("unmarshal legacy JSON failed: %v", err)
	}

	// Existing fields populated correctly
	if e.ID != "evt-legacy" {
		t.Errorf("ID: got %q, want %q", e.ID, "evt-legacy")
	}
	if e.AgentID != "agent-old" {
		t.Errorf("AgentID: got %q, want %q", e.AgentID, "agent-old")
	}
	if e.PresentationLabel != "Registered" {
		t.Errorf("PresentationLabel: got %q, want %q", e.PresentationLabel, "Registered")
	}
	if e.LifecycleStatus != "booting" {
		t.Errorf("LifecycleStatus: got %q, want %q", e.LifecycleStatus, "booting")
	}
	if e.LifecycleConfidence != 1.0 {
		t.Errorf("LifecycleConfidence: got %v, want 1.0", e.LifecycleConfidence)
	}

	// New fields must be zero-value
	if e.SessionID != "" {
		t.Errorf("SessionID should be empty, got %q", e.SessionID)
	}
	if e.ParentAgentID != "" {
		t.Errorf("ParentAgentID should be empty, got %q", e.ParentAgentID)
	}
	if e.TaskName != "" {
		t.Errorf("TaskName should be empty, got %q", e.TaskName)
	}
	if e.TaskDesc != "" {
		t.Errorf("TaskDesc should be empty, got %q", e.TaskDesc)
	}
	if e.ArtifactType != "" {
		t.Errorf("ArtifactType should be empty, got %q", e.ArtifactType)
	}
	if e.ArtifactRef != "" {
		t.Errorf("ArtifactRef should be empty, got %q", e.ArtifactRef)
	}
	if e.ArtifactData != "" {
		t.Errorf("ArtifactData should be empty, got %q", e.ArtifactData)
	}
	if e.ToolName != "" {
		t.Errorf("ToolName should be empty, got %q", e.ToolName)
	}
	if e.ToolInput != "" {
		t.Errorf("ToolInput should be empty, got %q", e.ToolInput)
	}
	if e.ToolType != "" {
		t.Errorf("ToolType should be empty, got %q", e.ToolType)
	}
	if e.ToolDuration != 0 {
		t.Errorf("ToolDuration should be 0, got %d", e.ToolDuration)
	}
}
