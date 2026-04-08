package store

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/core"
)

// transcriptRecord is the adapter type that mirrors the on-disk JSONL schema
// emitted by Claude Code at ~/.claude/projects/<encoded>/<session>.jsonl. It
// is intentionally permissive: unknown fields are ignored by encoding/json,
// and missing optional sub-objects (Message, Usage) collapse to zero values.
//
// When Claude Code rotates the schema, only this adapter needs to change.
type transcriptRecord struct {
	Type        string             `json:"type"`
	UUID        string             `json:"uuid"`
	SessionID   string             `json:"sessionId"`
	Timestamp   string             `json:"timestamp"`
	RequestID   string             `json:"requestId"`
	Cwd         string             `json:"cwd"`
	IsSidechain bool               `json:"isSidechain"`
	Message     *transcriptMessage `json:"message,omitempty"`
}

type transcriptMessage struct {
	ID    string             `json:"id"`
	Role  string             `json:"role"`
	Model string             `json:"model"`
	Usage *transcriptUsage   `json:"usage,omitempty"`
}

type transcriptUsage struct {
	InputTokens              int64                  `json:"input_tokens"`
	CacheCreationInputTokens int64                  `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int64                  `json:"cache_read_input_tokens"`
	OutputTokens             int64                  `json:"output_tokens"`
	ServiceTier              string                 `json:"service_tier"`
	CacheCreation            *transcriptCacheBlock  `json:"cache_creation,omitempty"`
	ServerToolUse            *transcriptServerTools `json:"server_tool_use,omitempty"`
}

type transcriptCacheBlock struct {
	Ephemeral5mInputTokens int64 `json:"ephemeral_5m_input_tokens"`
	Ephemeral1hInputTokens int64 `json:"ephemeral_1h_input_tokens"`
}

type transcriptServerTools struct {
	WebSearchRequests int64 `json:"web_search_requests"`
	WebFetchRequests  int64 `json:"web_fetch_requests"`
}

// ParseTranscriptLine decodes a single JSONL line into a CostRecord. The
// returned bool reports whether the line yielded a usage record:
//
//   - (record, true, nil)  — assistant message with usage block
//   - (nil, false, nil)    — non-assistant or no-usage record (skip silently)
//   - (nil, false, err)    — JSON decode failure (caller decides whether to skip)
//
// Unknown fields are tolerated. The function never returns a partially
// populated record on a soft skip.
func ParseTranscriptLine(line []byte) (*core.CostRecord, bool, error) {
	if len(line) == 0 {
		return nil, false, nil
	}
	var raw transcriptRecord
	if err := json.Unmarshal(line, &raw); err != nil {
		return nil, false, fmt.Errorf("decode transcript line: %w", err)
	}
	if raw.Type != "assistant" || raw.Message == nil || raw.Message.Usage == nil {
		return nil, false, nil
	}
	if raw.Message.Model == "" {
		log.Printf("transcript_parser: assistant record %q missing message.model — skipping", raw.UUID)
		return nil, false, nil
	}

	usage := raw.Message.Usage
	record := &core.CostRecord{
		SessionID:         raw.SessionID,
		ProjectPath:       raw.Cwd,
		Model:             raw.Message.Model,
		ServiceTier:       usage.ServiceTier,
		InputTokens:       usage.InputTokens,
		CacheReadTokens:   usage.CacheReadInputTokens,
		OutputTokens:      usage.OutputTokens,
		RequestID:         raw.RequestID,
		MessageID:         raw.Message.ID,
	}

	// Cache creation may be reported either as a flat
	// cache_creation_input_tokens count or as a structured cache_creation
	// block split into ephemeral_5m / ephemeral_1h. Prefer the structured
	// breakdown when present so we can apply the right per-tier price.
	if usage.CacheCreation != nil {
		record.CacheCreate5mTokens = usage.CacheCreation.Ephemeral5mInputTokens
		record.CacheCreate1hTokens = usage.CacheCreation.Ephemeral1hInputTokens
	} else if usage.CacheCreationInputTokens > 0 {
		// Unknown split — bucket into the cheaper 5m tier so we don't
		// over-bill, and log so operators notice the schema gap.
		record.CacheCreate5mTokens = usage.CacheCreationInputTokens
		log.Printf("transcript_parser: %s usage block missing cache_creation split — bucketing %d tokens as 5m", raw.UUID, usage.CacheCreationInputTokens)
	}

	if usage.ServerToolUse != nil {
		record.WebSearchRequests = usage.ServerToolUse.WebSearchRequests
		record.WebFetchRequests = usage.ServerToolUse.WebFetchRequests
	}

	if raw.Timestamp != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, raw.Timestamp); err == nil {
			record.RecordedAt = parsed.UTC()
		}
	}
	if raw.IsSidechain {
		record.Source = core.CostSourceSidechain
	} else {
		record.Source = core.CostSourceAssistant
	}
	if price, ok := core.LookupModelPrice(record.Model); ok {
		record.EstimatedUSD = core.CalculateUSD(*record, price)
	}
	return record, true, nil
}

// ParseTranscriptFile streams a Claude Code transcript JSONL file and returns
// the CostRecords for every assistant message that carried a usage block.
// Malformed lines are logged and skipped so a single corrupt entry cannot
// poison the whole file. Missing files return os.ErrNotExist unwrapped.
func ParseTranscriptFile(path string) ([]core.CostRecord, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		return nil, fmt.Errorf("open transcript %q: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Transcript lines can hold multi-KB tool input previews. Bump the
	// buffer to 4 MB to match the largest assistant turn we have observed
	// in ham-agents transcripts.
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	records := make([]core.CostRecord, 0, 64)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Bytes()
		record, ok, parseErr := ParseTranscriptLine(line)
		if parseErr != nil {
			log.Printf("transcript_parser: %s line %d: %v — skipping", path, lineNo, parseErr)
			continue
		}
		if !ok {
			continue
		}
		records = append(records, *record)
	}
	if err := scanner.Err(); err != nil {
		return records, fmt.Errorf("scan transcript %q: %w", path, err)
	}
	return records, nil
}
