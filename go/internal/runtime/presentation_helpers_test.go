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
