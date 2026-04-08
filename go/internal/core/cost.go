package core

import "time"

// CostSource identifies where a CostRecord was extracted from.
const (
	// CostSourceAssistant marks records that came from a top-level assistant
	// message in the Claude Code transcript.
	CostSourceAssistant = "assistant"
	// CostSourceSidechain marks records that came from a sidechain assistant
	// message (e.g. Agent tool inner conversation).
	CostSourceSidechain = "sidechain"
)

// CostRecord captures the token + USD usage of a single Claude API response,
// reconstructed from a Claude Code transcript JSONL line. See ADR-3 for the
// data source rationale.
//
// All token counts are non-negative. EstimatedUSD is computed by
// CalculateUSD using LookupModelPrice.
type CostRecord struct {
	AgentID             string    `json:"agent_id,omitempty"`
	SessionID           string    `json:"session_id,omitempty"`
	ProjectPath         string    `json:"project_path,omitempty"`
	Model               string    `json:"model"`
	ServiceTier         string    `json:"service_tier,omitempty"`
	InputTokens         int64     `json:"input_tokens"`
	CacheCreate5mTokens int64     `json:"cache_create_5m_tokens,omitempty"`
	CacheCreate1hTokens int64     `json:"cache_create_1h_tokens,omitempty"`
	CacheReadTokens     int64     `json:"cache_read_tokens,omitempty"`
	OutputTokens        int64     `json:"output_tokens"`
	WebSearchRequests   int64     `json:"web_search_requests,omitempty"`
	WebFetchRequests    int64     `json:"web_fetch_requests,omitempty"`
	EstimatedUSD        float64   `json:"estimated_usd"`
	RecordedAt          time.Time `json:"recorded_at"`
	Source              string    `json:"source"`
	RequestID           string    `json:"request_id,omitempty"`
	MessageID           string    `json:"message_id,omitempty"`
}

// DedupKey returns a stable identifier for deduplicating cost records across
// repeated transcript reads. RequestID is preferred when present because the
// Anthropic API guarantees its uniqueness; MessageID is the fallback for
// older records that omit requestId.
func (r CostRecord) DedupKey() string {
	if r.RequestID != "" {
		return "req:" + r.RequestID
	}
	if r.MessageID != "" {
		return "msg:" + r.MessageID
	}
	return ""
}

// CalculateUSD returns the estimated USD cost for the record using the given
// price table. The result is the sum of base input, both cache-write tiers,
// cache-read, output token costs, and per-request web search charges
// ($10 per 1k searches per Anthropic pricing page). Web fetch is free.
func CalculateUSD(record CostRecord, price ModelPrice) float64 {
	const perMillion = 1_000_000.0
	tokenCost := float64(record.InputTokens)*price.InputPer1M/perMillion +
		float64(record.CacheCreate5mTokens)*price.CacheCreate5mPer1M/perMillion +
		float64(record.CacheCreate1hTokens)*price.CacheCreate1hPer1M/perMillion +
		float64(record.CacheReadTokens)*price.CacheReadPer1M/perMillion +
		float64(record.OutputTokens)*price.OutputPer1M/perMillion
	webSearchCost := float64(record.WebSearchRequests) * 10.0 / 1000.0
	return tokenCost + webSearchCost
}
