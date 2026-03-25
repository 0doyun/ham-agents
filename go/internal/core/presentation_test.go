package core

import "testing"

func TestHumanAgentStatusLabelHumanizesSpecialStatuses(t *testing.T) {
	t.Parallel()

	if got := HumanAgentStatusLabel(AgentStatusWaitingInput); got != "needs input" {
		t.Fatalf("unexpected waiting label %q", got)
	}
	if got := HumanAgentStatusLabel(AgentStatusRunningTool); got != "running tool" {
		t.Fatalf("unexpected running tool label %q", got)
	}
	if got := HumanAgentStatusLabel(AgentStatusDisconnected); got != "disconnected" {
		t.Fatalf("unexpected default label %q", got)
	}
}

func TestRequiresAttentionRecognizesUrgentStatuses(t *testing.T) {
	t.Parallel()

	if !RequiresAttention(AgentStatusError) || !RequiresAttention(AgentStatusWaitingInput) || !RequiresAttention(AgentStatusDisconnected) {
		t.Fatal("expected urgent statuses to require attention")
	}
	if RequiresAttention(AgentStatusThinking) {
		t.Fatal("expected non-urgent status to avoid attention")
	}
}

func TestAttentionSeverityOrdersUrgentStatuses(t *testing.T) {
	t.Parallel()

	if got := AttentionSeverity(AgentStatusError); got != 0 {
		t.Fatalf("unexpected error severity %d", got)
	}
	if got := AttentionSeverity(AgentStatusWaitingInput); got != 1 {
		t.Fatalf("unexpected waiting severity %d", got)
	}
	if got := AttentionSeverity(AgentStatusDisconnected); got != 2 {
		t.Fatalf("unexpected disconnected severity %d", got)
	}
	if got := AttentionSeverity(AgentStatusThinking); got != 3 {
		t.Fatalf("unexpected default severity %d", got)
	}
}

func TestEventsAfterIDFiltersAndLimits(t *testing.T) {
	t.Parallel()

	events := []Event{{ID: "event-1"}, {ID: "event-2"}, {ID: "event-3"}}
	filtered := EventsAfterID(events, "event-1", 10)
	if len(filtered) != 2 || filtered[0].ID != "event-2" || filtered[1].ID != "event-3" {
		t.Fatalf("unexpected filtered events %#v", filtered)
	}

	limited := EventsAfterID(events, "", 2)
	if len(limited) != 2 || limited[0].ID != "event-2" || limited[1].ID != "event-3" {
		t.Fatalf("unexpected limited events %#v", limited)
	}
}
