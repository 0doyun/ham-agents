package core_test

import (
	"math"
	"testing"

	"github.com/ham-agents/ham-agents/go/internal/core"
)

func TestCalculateUSD_BasicMath(t *testing.T) {
	t.Parallel()

	price := core.ModelPrice{
		InputPer1M:         5,
		CacheCreate5mPer1M: 6.25,
		CacheCreate1hPer1M: 10,
		CacheReadPer1M:     0.50,
		OutputPer1M:        25,
	}
	record := core.CostRecord{
		Model:               "claude-opus-4-6",
		InputTokens:         1_000_000,
		CacheCreate5mTokens: 1_000_000,
		CacheCreate1hTokens: 1_000_000,
		CacheReadTokens:     1_000_000,
		OutputTokens:        1_000_000,
		WebSearchRequests:   100,
	}
	got := core.CalculateUSD(record, price)
	// 5 + 6.25 + 10 + 0.50 + 25 = 46.75 + web search 100*$10/1k = $1
	want := 46.75 + 1.0
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("CalculateUSD: got %.6f want %.6f", got, want)
	}
}

func TestCalculateUSD_CacheReadCheaperThanInput(t *testing.T) {
	t.Parallel()

	price, ok := core.LookupModelPrice("claude-opus-4-6")
	if !ok {
		t.Fatal("expected claude-opus-4-6 to be known")
	}
	inputOnly := core.CalculateUSD(core.CostRecord{InputTokens: 1_000_000}, price)
	cacheReadOnly := core.CalculateUSD(core.CostRecord{CacheReadTokens: 1_000_000}, price)
	if !(cacheReadOnly < inputOnly) {
		t.Fatalf("cache_read should be cheaper than input: cacheRead=%.4f input=%.4f", cacheReadOnly, inputOnly)
	}
}

func TestCostRecord_DedupKeyPrefersRequestID(t *testing.T) {
	t.Parallel()

	withReq := core.CostRecord{RequestID: "req_1", MessageID: "msg_1"}
	if got := withReq.DedupKey(); got != "req:req_1" {
		t.Fatalf("DedupKey with both IDs: got %q want req:req_1", got)
	}
	withMsgOnly := core.CostRecord{MessageID: "msg_2"}
	if got := withMsgOnly.DedupKey(); got != "msg:msg_2" {
		t.Fatalf("DedupKey msg only: got %q want msg:msg_2", got)
	}
	empty := core.CostRecord{}
	if got := empty.DedupKey(); got != "" {
		t.Fatalf("DedupKey empty: got %q want empty", got)
	}
}
