package ipc_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/core"
	"github.com/ham-agents/ham-agents/go/internal/ipc"
	"github.com/ham-agents/ham-agents/go/internal/store"
)

// stubLister gives the existing test helpers an empty session source so we
// can construct an ipc.Server without iTerm/tmux adapters.
type stubLister struct{}

func (stubLister) ListSessions() ([]core.AttachableSession, error) { return nil, nil }

// dispatchCostSummary builds a server with a pre-seeded FileCostStore and
// drives one cost.summary request through it. We avoid spinning up a real
// unix socket because the dispatch path is the only behavior under test.
func dispatchCostSummary(t *testing.T, request ipc.Request) ipc.Response {
	t.Helper()
	dir := t.TempDir()
	costPath := filepath.Join(dir, "cost.jsonl")
	costStore := store.NewFileCostStore(costPath)
	now := time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)
	records := []core.CostRecord{
		{AgentID: "a1", Model: "claude-opus-4-6", InputTokens: 100, OutputTokens: 200, EstimatedUSD: 1.5, RecordedAt: now, RequestID: "r1"},
		{AgentID: "a2", Model: "claude-sonnet-4-6", InputTokens: 50, OutputTokens: 80, EstimatedUSD: 0.6, RecordedAt: now.Add(24 * time.Hour), RequestID: "r2"},
		{AgentID: "", Model: "claude-haiku-4-5", InputTokens: 10, OutputTokens: 20, EstimatedUSD: 0.04, RecordedAt: now.Add(24 * time.Hour), RequestID: "r3"},
	}
	for _, record := range records {
		if err := costStore.Append(context.Background(), record); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	server := ipc.NewServer("/dev/null", nil, nil, nil, nil, nil, stubLister{}, stubLister{})
	server.SetCostStore(costStore)
	response, err := dispatchForTest(server, request)
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	return response
}

// dispatchForTest reaches into the unexported dispatch by reusing the public
// CostSummary client→server contract. Since dispatch is unexported, we drive
// the server through its public Request handler indirectly via a temporary
// in-process loopback. For this test we instead call the package-private
// helper exposed in cost_summary_export_test.go.
func dispatchForTest(server *ipc.Server, request ipc.Request) (ipc.Response, error) {
	return ipc.DispatchForTest(server, context.Background(), request)
}

func TestHandleCostSummary_RawGroupByReturnsRecordsAndAllRollups(t *testing.T) {
	t.Parallel()

	response := dispatchCostSummary(t, ipc.Request{Command: ipc.CommandCostSummary})
	if len(response.CostRecords) != 3 {
		t.Fatalf("raw groupBy should include all records, got %d", len(response.CostRecords))
	}
	if response.ByModel == nil || response.ByDay == nil || response.ByAgent == nil {
		t.Fatalf("raw groupBy should populate every rollup map")
	}
	if response.ByModel["claude-opus-4-6"] != 1.5 {
		t.Fatalf("opus rollup: %.4f", response.ByModel["claude-opus-4-6"])
	}
	if response.ByAgent["(orphan)"] != 0.04 {
		t.Fatalf("orphan rollup: %.4f", response.ByAgent["(orphan)"])
	}
}

func TestHandleCostSummary_GroupByModelOmitsRecordsAndOtherMaps(t *testing.T) {
	t.Parallel()

	response := dispatchCostSummary(t, ipc.Request{
		Command: ipc.CommandCostSummary,
		GroupBy: ipc.CostGroupByModel,
	})
	if len(response.CostRecords) != 0 {
		t.Fatalf("model groupBy should drop CostRecords, got %d", len(response.CostRecords))
	}
	if response.ByModel == nil || response.ByModel["claude-opus-4-6"] != 1.5 {
		t.Fatalf("model rollup missing or wrong: %v", response.ByModel)
	}
	if response.ByDay != nil {
		t.Fatalf("model groupBy should leave ByDay nil, got %v", response.ByDay)
	}
	if response.ByAgent != nil {
		t.Fatalf("model groupBy should leave ByAgent nil, got %v", response.ByAgent)
	}
	if response.TotalUSD <= 0 {
		t.Fatalf("TotalUSD should still aggregate, got %.4f", response.TotalUSD)
	}
}

func TestHandleCostSummary_GroupByDayOmitsModelAndAgent(t *testing.T) {
	t.Parallel()

	response := dispatchCostSummary(t, ipc.Request{
		Command: ipc.CommandCostSummary,
		GroupBy: ipc.CostGroupByDay,
	})
	if response.ByDay == nil || len(response.ByDay) != 2 {
		t.Fatalf("day rollup: %v", response.ByDay)
	}
	if response.ByModel != nil || response.ByAgent != nil {
		t.Fatalf("day groupBy should leave ByModel/ByAgent nil")
	}
}

func TestHandleCostSummary_GroupByAgentScopedByFilter(t *testing.T) {
	t.Parallel()

	response := dispatchCostSummary(t, ipc.Request{
		Command:       ipc.CommandCostSummary,
		AgentIDFilter: "a1",
		GroupBy:       ipc.CostGroupByAgent,
	})
	if response.ByAgent == nil || len(response.ByAgent) != 1 {
		t.Fatalf("agent rollup with filter should yield single key, got %v", response.ByAgent)
	}
	if response.ByAgent["a1"] != 1.5 {
		t.Fatalf("a1 rollup: %.4f", response.ByAgent["a1"])
	}
}
