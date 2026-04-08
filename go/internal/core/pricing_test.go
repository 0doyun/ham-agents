package core_test

import (
	"testing"

	"github.com/ham-agents/ham-agents/go/internal/core"
)

func TestLookupModelPrice_KnownModels(t *testing.T) {
	t.Parallel()

	cases := []struct {
		model string
		input float64
	}{
		{"claude-opus-4-6", 5},
		{"claude-sonnet-4-6", 3},
		{"claude-haiku-4-5", 1},
		{"claude-opus-4-5", 5},
		{"claude-sonnet-4-5", 3},
		{"claude-3-7-sonnet", 3},
		{"claude-3-5-haiku", 0.80},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.model, func(t *testing.T) {
			t.Parallel()
			price, ok := core.LookupModelPrice(tc.model)
			if !ok {
				t.Fatalf("model %q should be known", tc.model)
			}
			if price.InputPer1M != tc.input {
				t.Fatalf("model %q input price: got %.4f want %.4f", tc.model, price.InputPer1M, tc.input)
			}
		})
	}
}

func TestLookupModelPrice_UnknownReturnsFalse(t *testing.T) {
	t.Parallel()

	cases := []string{"", "  ", "gpt-4", "claude-imaginary-99", "totally-bogus"}
	for _, model := range cases {
		model := model
		t.Run(model, func(t *testing.T) {
			t.Parallel()
			if _, ok := core.LookupModelPrice(model); ok {
				t.Fatalf("model %q should be unknown", model)
			}
		})
	}
}

func TestLookupModelPrice_StripsDatedSuffix(t *testing.T) {
	t.Parallel()

	// Anthropic sometimes appends a date suffix; ensure we still resolve to
	// the base model entry.
	price, ok := core.LookupModelPrice("claude-opus-4-6-20260101")
	if !ok {
		t.Fatal("dated suffix should resolve to claude-opus-4-6")
	}
	if price.InputPer1M != 5 {
		t.Fatalf("expected $5 input price, got %.4f", price.InputPer1M)
	}
}

func TestPricing_OpusCostlierThanSonnetCostlierThanHaiku(t *testing.T) {
	t.Parallel()

	opus, _ := core.LookupModelPrice("claude-opus-4-6")
	sonnet, _ := core.LookupModelPrice("claude-sonnet-4-6")
	haiku, _ := core.LookupModelPrice("claude-haiku-4-5")

	if !(opus.InputPer1M > sonnet.InputPer1M) {
		t.Fatalf("opus input %.2f should exceed sonnet %.2f", opus.InputPer1M, sonnet.InputPer1M)
	}
	if !(sonnet.InputPer1M > haiku.InputPer1M) {
		t.Fatalf("sonnet input %.2f should exceed haiku %.2f", sonnet.InputPer1M, haiku.InputPer1M)
	}
	if !(opus.OutputPer1M > sonnet.OutputPer1M) {
		t.Fatalf("opus output should exceed sonnet")
	}
	if !(sonnet.OutputPer1M > haiku.OutputPer1M) {
		t.Fatalf("sonnet output should exceed haiku")
	}
}
