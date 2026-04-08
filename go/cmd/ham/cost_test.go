package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/core"
	"github.com/ham-agents/ham-agents/go/internal/ipc"
)

func TestParseCostInput_Defaults(t *testing.T) {
	t.Parallel()

	opts, err := parseCostInput(nil)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if opts.agentFilter != "" || opts.sinceDays != 0 || opts.groupBy != "model" || opts.asJSON {
		t.Fatalf("unexpected defaults: %+v", opts)
	}
}

func TestParseCostInput_Flags(t *testing.T) {
	t.Parallel()

	opts, err := parseCostInput([]string{"--agent", "agent-7", "--days", "14", "--by", "day", "--json"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if opts.agentFilter != "agent-7" || opts.sinceDays != 14 || opts.groupBy != "day" || !opts.asJSON {
		t.Fatalf("unexpected parse: %+v", opts)
	}
}

func TestParseCostInput_RejectsInvalidGroupBy(t *testing.T) {
	t.Parallel()

	if _, err := parseCostInput([]string{"--by", "bogus"}); err == nil {
		t.Fatalf("expected --by bogus to fail")
	}
}

func TestParseCostInput_RejectsNegativeDays(t *testing.T) {
	t.Parallel()

	if _, err := parseCostInput([]string{"--days", "-1"}); err == nil {
		t.Fatalf("expected negative days to fail")
	}
}

func sampleResponse() ipc.Response {
	t1 := time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 4, 8, 9, 30, 0, 0, time.UTC)
	return ipc.Response{
		CostRecords: []core.CostRecord{
			{AgentID: "a1", Model: "claude-opus-4-6", InputTokens: 100, CacheReadTokens: 50, OutputTokens: 200, EstimatedUSD: 1.5, RecordedAt: t1},
			{AgentID: "a2", Model: "claude-sonnet-4-6", InputTokens: 80, OutputTokens: 120, EstimatedUSD: 0.6, RecordedAt: t2},
			{AgentID: "", Model: "claude-haiku-4-5", InputTokens: 40, OutputTokens: 60, EstimatedUSD: 0.04, RecordedAt: t2},
		},
		TotalUSD: 2.14,
		ByModel: map[string]float64{
			"claude-opus-4-6":   1.5,
			"claude-sonnet-4-6": 0.6,
			"claude-haiku-4-5":  0.04,
		},
		ByDay: map[string]float64{
			"2026-04-07": 1.5,
			"2026-04-08": 0.64,
		},
		ByAgent: map[string]float64{
			"a1":       1.5,
			"a2":       0.6,
			"(orphan)": 0.04,
		},
	}
}

func TestBuildCostSummaryView_GroupByModel(t *testing.T) {
	t.Parallel()

	view := buildCostSummaryView(sampleResponse(), "model")
	if view.GroupBy != "model" || view.RecordCount != 3 {
		t.Fatalf("view header: %+v", view)
	}
	if len(view.Rows) != 3 {
		t.Fatalf("expected 3 model rows, got %d", len(view.Rows))
	}
	// Rows are sorted by key.
	keys := []string{view.Rows[0].Key, view.Rows[1].Key, view.Rows[2].Key}
	want := []string{"claude-haiku-4-5", "claude-opus-4-6", "claude-sonnet-4-6"}
	for i := range keys {
		if keys[i] != want[i] {
			t.Fatalf("row %d key: got %q want %q", i, keys[i], want[i])
		}
	}
	// Opus row should carry the cache_read tokens from the underlying record.
	for _, row := range view.Rows {
		if row.Key == "claude-opus-4-6" && row.CacheReadTokens != 50 {
			t.Fatalf("opus cache_read: got %d want 50", row.CacheReadTokens)
		}
	}
}

func TestBuildCostSummaryView_GroupByDay(t *testing.T) {
	t.Parallel()

	view := buildCostSummaryView(sampleResponse(), "day")
	if len(view.Rows) != 2 {
		t.Fatalf("day rows: %d", len(view.Rows))
	}
	if view.Rows[0].Key != "2026-04-07" || view.Rows[1].Key != "2026-04-08" {
		t.Fatalf("day order: %+v", view.Rows)
	}
	if view.Rows[1].RecordCount != 2 {
		t.Fatalf("2026-04-08 should have 2 records, got %d", view.Rows[1].RecordCount)
	}
}

func TestBuildCostSummaryView_GroupByAgent(t *testing.T) {
	t.Parallel()

	view := buildCostSummaryView(sampleResponse(), "agent")
	if len(view.Rows) != 3 {
		t.Fatalf("agent rows: %d", len(view.Rows))
	}
	// Find the orphan row and confirm it counts the empty-AgentID record.
	var orphan *costRow
	for i := range view.Rows {
		if view.Rows[i].Key == "(orphan)" {
			orphan = &view.Rows[i]
		}
	}
	if orphan == nil {
		t.Fatalf("orphan row missing: %+v", view.Rows)
	}
	if orphan.RecordCount != 1 {
		t.Fatalf("orphan record count: got %d want 1", orphan.RecordCount)
	}
}

func TestRenderCostSummary_TextFormat(t *testing.T) {
	t.Parallel()

	view := buildCostSummaryView(sampleResponse(), "model")
	var buf bytes.Buffer
	if err := renderCostSummary(&buf, view, false); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"MODEL", "INPUT", "CACHE_READ", "OUTPUT", "USD", "claude-opus-4-6", "TOTAL", "$2.1400"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestRenderCostSummary_JSON(t *testing.T) {
	t.Parallel()

	view := buildCostSummaryView(sampleResponse(), "model")
	var buf bytes.Buffer
	if err := renderCostSummary(&buf, view, true); err != nil {
		t.Fatalf("render json: %v", err)
	}
	var decoded costSummaryView
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("decode: %v\n%s", err, buf.String())
	}
	if decoded.GroupBy != "model" || decoded.RecordCount != 3 {
		t.Fatalf("decoded mismatch: %+v", decoded)
	}
}
