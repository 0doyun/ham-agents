package core

import "strings"

// ModelPrice describes per-1M-token USD pricing for one Claude model. The
// fields mirror the columns of the Anthropic pricing page
// (https://platform.claude.com/docs/en/docs/about-claude/pricing) as fetched
// on 2026-04-08.
type ModelPrice struct {
	InputPer1M         float64
	CacheCreate5mPer1M float64
	CacheCreate1hPer1M float64
	CacheReadPer1M     float64
	OutputPer1M        float64
}

// modelPriceTable is a static snapshot of Anthropic's published pricing for
// the model IDs ham-agents tracks via Claude Code transcripts. Update this
// table when Anthropic publishes new prices or new model IDs. Source:
// https://platform.claude.com/docs/en/docs/about-claude/pricing (2026-04-08).
//
// Pricing tiers (per the same page):
//   - 5m cache write = 1.25x base input
//   - 1h cache write = 2.00x base input
//   - cache read     = 0.10x base input
var modelPriceTable = map[string]ModelPrice{
	// Opus 4.5 / 4.6 family — $5 input / $25 output
	"claude-opus-4-6": {InputPer1M: 5, CacheCreate5mPer1M: 6.25, CacheCreate1hPer1M: 10, CacheReadPer1M: 0.50, OutputPer1M: 25},
	"claude-opus-4-5": {InputPer1M: 5, CacheCreate5mPer1M: 6.25, CacheCreate1hPer1M: 10, CacheReadPer1M: 0.50, OutputPer1M: 25},
	// Opus 4 / 4.1 — $15 input / $75 output (legacy pricing)
	"claude-opus-4-1": {InputPer1M: 15, CacheCreate5mPer1M: 18.75, CacheCreate1hPer1M: 30, CacheReadPer1M: 1.50, OutputPer1M: 75},
	"claude-opus-4":   {InputPer1M: 15, CacheCreate5mPer1M: 18.75, CacheCreate1hPer1M: 30, CacheReadPer1M: 1.50, OutputPer1M: 75},
	// Sonnet 4 / 4.5 / 4.6 — $3 input / $15 output
	"claude-sonnet-4-6": {InputPer1M: 3, CacheCreate5mPer1M: 3.75, CacheCreate1hPer1M: 6, CacheReadPer1M: 0.30, OutputPer1M: 15},
	"claude-sonnet-4-5": {InputPer1M: 3, CacheCreate5mPer1M: 3.75, CacheCreate1hPer1M: 6, CacheReadPer1M: 0.30, OutputPer1M: 15},
	"claude-sonnet-4":   {InputPer1M: 3, CacheCreate5mPer1M: 3.75, CacheCreate1hPer1M: 6, CacheReadPer1M: 0.30, OutputPer1M: 15},
	// Sonnet 3.7 (deprecated) — same Sonnet pricing
	"claude-3-7-sonnet": {InputPer1M: 3, CacheCreate5mPer1M: 3.75, CacheCreate1hPer1M: 6, CacheReadPer1M: 0.30, OutputPer1M: 15},
	// Sonnet 3.5 — same Sonnet pricing per page (legacy entry)
	"claude-3-5-sonnet": {InputPer1M: 3, CacheCreate5mPer1M: 3.75, CacheCreate1hPer1M: 6, CacheReadPer1M: 0.30, OutputPer1M: 15},
	// Haiku 4.5 — $1 input / $5 output
	"claude-haiku-4-5": {InputPer1M: 1, CacheCreate5mPer1M: 1.25, CacheCreate1hPer1M: 2, CacheReadPer1M: 0.10, OutputPer1M: 5},
	// Haiku 3.5 — $0.80 / $4
	"claude-3-5-haiku": {InputPer1M: 0.80, CacheCreate5mPer1M: 1.0, CacheCreate1hPer1M: 1.6, CacheReadPer1M: 0.08, OutputPer1M: 4},
	// Haiku 3 — $0.25 / $1.25
	"claude-3-haiku": {InputPer1M: 0.25, CacheCreate5mPer1M: 0.30, CacheCreate1hPer1M: 0.50, CacheReadPer1M: 0.03, OutputPer1M: 1.25},
}

// LookupModelPrice returns the static price entry for the given model ID. The
// match is case-insensitive and tolerates the dated suffixes Anthropic
// sometimes appends to model IDs (e.g. "claude-opus-4-6-20260101"). When no
// match is found, the second return value is false and CalculateUSD callers
// should treat the cost as unknown.
func LookupModelPrice(model string) (ModelPrice, bool) {
	normalized := strings.ToLower(strings.TrimSpace(model))
	if normalized == "" {
		return ModelPrice{}, false
	}
	if price, ok := modelPriceTable[normalized]; ok {
		return price, true
	}
	// Try progressively shorter prefixes by stripping dated/version suffixes
	// like "-20260101" so future model IDs still resolve to the latest known
	// entry. We only walk segments separated by '-' to avoid spurious hits.
	parts := strings.Split(normalized, "-")
	for i := len(parts) - 1; i >= 2; i-- {
		candidate := strings.Join(parts[:i], "-")
		if price, ok := modelPriceTable[candidate]; ok {
			return price, true
		}
	}
	return ModelPrice{}, false
}
