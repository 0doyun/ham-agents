package adapters

import (
	"testing"

	"github.com/ham-agents/ham-agents/go/internal/core"
)

func TestInferManagedOutputUsesClaudeStructuredPayloadWhenAvailable(t *testing.T) {
	t.Parallel()

	inferred := InferManagedOutput("claude", `{"status":"waiting_input","reason":"Needs confirmation.","summary":"Approve the patch?"}`, false)

	if inferred.Status != core.AgentStatusWaitingInput {
		t.Fatalf("expected waiting_input, got %q", inferred.Status)
	}
	if inferred.Reason != "Needs confirmation." || inferred.Summary != "Approve the patch?" {
		t.Fatalf("unexpected inference %#v", inferred)
	}
}

func TestClassifyProcessExitMapsNilToDoneAndErrorToError(t *testing.T) {
	t.Parallel()

	if got := ClassifyProcessExit(nil); got.Status != core.AgentStatusDone {
		t.Fatalf("expected done for nil exit, got %#v", got)
	}
	if got := ClassifyProcessExit(assertionError("boom")); got.Status != core.AgentStatusError {
		t.Fatalf("expected error for non-nil exit, got %#v", got)
	}
}

type assertionError string

func (e assertionError) Error() string { return string(e) }
