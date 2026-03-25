package runtime

import (
	"testing"

	"github.com/ham-agents/ham-agents/go/internal/core"
)

func TestRemovalPresentationSummaryHumanizesWaitingInput(t *testing.T) {
	t.Parallel()

	summary := removalPresentationSummary(core.Event{
		Type:            core.EventTypeAgentRemoved,
		Summary:         "Tracking stopped.",
		LifecycleStatus: "waiting_input",
		LifecycleReason: "Needs confirmation.",
	})

	if summary != "Stopped tracking while waiting for input. Needs confirmation." {
		t.Fatalf("unexpected removal presentation summary %q", summary)
	}
}

func TestEventPresentationHintMapsRunningToolStatus(t *testing.T) {
	t.Parallel()

	label, emphasis, summary := eventPresentationHint(core.Event{
		Type:    core.EventTypeAgentStatusUpdated,
		Summary: "Status changed to running_tool. Observed tool-like activity.",
	})

	if label != "Running Tool" || emphasis != "info" || summary != "Observed tool-like activity." {
		t.Fatalf("unexpected presentation hint %q %q %q", label, emphasis, summary)
	}
}

func TestEventPresentationHintMapsReadingStatus(t *testing.T) {
	t.Parallel()

	label, emphasis, summary := eventPresentationHint(core.Event{
		Type:    core.EventTypeAgentStatusUpdated,
		Summary: "Status changed to reading. Observed reading-like activity.",
	})

	if label != "Reading" || emphasis != "info" || summary != "Observed reading-like activity." {
		t.Fatalf("unexpected presentation hint %q %q %q", label, emphasis, summary)
	}
}

func TestEventPresentationHintMapsBootingStatus(t *testing.T) {
	t.Parallel()

	label, emphasis, summary := eventPresentationHint(core.Event{
		Type:    core.EventTypeAgentStatusUpdated,
		Summary: "Status changed to booting. Observed booting-like activity.",
	})

	if label != "Booting" || emphasis != "info" || summary != "Observed booting-like activity." {
		t.Fatalf("unexpected presentation hint %q %q %q", label, emphasis, summary)
	}
}

func TestEventPresentationHintMapsObservedReconnectionStatus(t *testing.T) {
	t.Parallel()

	label, emphasis, summary := eventPresentationHint(core.Event{
		Type:    core.EventTypeAgentStatusUpdated,
		Summary: "Status changed to idle. Observed connection restored.",
	})

	if label != "Reconnected" || emphasis != "positive" || summary != "Observed connection restored." {
		t.Fatalf("unexpected presentation hint %q %q %q", label, emphasis, summary)
	}
}

func TestEventPresentationHintMapsThinkingStatus(t *testing.T) {
	t.Parallel()

	label, emphasis, summary := eventPresentationHint(core.Event{
		Type:    core.EventTypeAgentStatusUpdated,
		Summary: "Status changed to thinking. Observed recent output.",
	})

	if label != "Thinking" || emphasis != "info" || summary != "Observed recent output." {
		t.Fatalf("unexpected presentation hint %q %q %q", label, emphasis, summary)
	}
}

func TestEventPresentationHintMapsSleepingStatus(t *testing.T) {
	t.Parallel()

	label, emphasis, summary := eventPresentationHint(core.Event{
		Type:    core.EventTypeAgentStatusUpdated,
		Summary: "Status changed to sleeping. Observed source idle for 10m.",
	})

	if label != "Sleeping" || emphasis != "neutral" || summary != "Observed source idle for 10m." {
		t.Fatalf("unexpected presentation hint %q %q %q", label, emphasis, summary)
	}
}

func TestAttentionSubtitleHumanizesWaitingInputStatus(t *testing.T) {
	t.Parallel()

	subtitle := attentionSubtitle(core.Agent{
		Status:           core.AgentStatusWaitingInput,
		StatusConfidence: 0.85,
		StatusReason:     "Needs confirmation.",
	})

	if subtitle != "needs input · high confidence · Needs confirmation." {
		t.Fatalf("unexpected attention subtitle %q", subtitle)
	}
}
