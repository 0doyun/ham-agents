package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ham-agents/ham-agents/go/internal/core"
	"github.com/ham-agents/ham-agents/go/internal/store"
)

const sampleAssistantLine = `{"type":"assistant","uuid":"u1","sessionId":"sess-abc","timestamp":"2026-04-08T12:34:56.000Z","requestId":"req_abc","cwd":"/tmp/proj","isSidechain":false,"message":{"id":"msg_001","role":"assistant","model":"claude-opus-4-6","usage":{"input_tokens":3,"cache_creation_input_tokens":20525,"cache_read_input_tokens":0,"output_tokens":84,"service_tier":"standard","cache_creation":{"ephemeral_5m_input_tokens":0,"ephemeral_1h_input_tokens":20525},"server_tool_use":{"web_search_requests":0,"web_fetch_requests":0}}}}`

const sampleSidechainLine = `{"type":"assistant","uuid":"u2","sessionId":"sess-side","timestamp":"2026-04-08T12:35:00.000Z","requestId":"req_side","isSidechain":true,"message":{"id":"msg_002","role":"assistant","model":"claude-haiku-4-5","usage":{"input_tokens":10,"output_tokens":20,"service_tier":"standard"}}}`

const sampleUserLine = `{"type":"user","uuid":"u3","sessionId":"sess-abc","message":{"role":"user","content":"hi"}}`

const sampleAssistantNoUsage = `{"type":"assistant","uuid":"u4","sessionId":"sess-abc","message":{"id":"msg_003","role":"assistant","model":"claude-opus-4-6"}}`

const sampleAssistantNoModel = `{"type":"assistant","uuid":"u5","sessionId":"sess-abc","message":{"id":"msg_004","role":"assistant","usage":{"input_tokens":1,"output_tokens":1}}}`

const sampleSonnet = `{"type":"assistant","uuid":"u6","sessionId":"sess-x","timestamp":"2026-04-08T13:00:00.000Z","requestId":"req_sonnet","message":{"id":"msg_sonnet","role":"assistant","model":"claude-sonnet-4-6","usage":{"input_tokens":1000,"cache_read_input_tokens":500,"output_tokens":2000,"service_tier":"standard"}}}`

func TestParseTranscriptLine_AssistantWithUsage(t *testing.T) {
	t.Parallel()

	rec, ok, err := store.ParseTranscriptLine([]byte(sampleAssistantLine))
	if err != nil {
		t.Fatalf("parse line: %v", err)
	}
	if !ok || rec == nil {
		t.Fatalf("expected ok=true and non-nil record")
	}
	if rec.Model != "claude-opus-4-6" {
		t.Fatalf("model: got %q want claude-opus-4-6", rec.Model)
	}
	if rec.SessionID != "sess-abc" {
		t.Fatalf("sessionId: got %q", rec.SessionID)
	}
	if rec.RequestID != "req_abc" || rec.MessageID != "msg_001" {
		t.Fatalf("ids: req=%q msg=%q", rec.RequestID, rec.MessageID)
	}
	if rec.InputTokens != 3 || rec.OutputTokens != 84 {
		t.Fatalf("token counts: in=%d out=%d", rec.InputTokens, rec.OutputTokens)
	}
	if rec.CacheCreate1hTokens != 20525 || rec.CacheCreate5mTokens != 0 {
		t.Fatalf("cache split: 5m=%d 1h=%d", rec.CacheCreate5mTokens, rec.CacheCreate1hTokens)
	}
	if rec.Source != core.CostSourceAssistant {
		t.Fatalf("source: got %q", rec.Source)
	}
	if rec.EstimatedUSD <= 0 {
		t.Fatalf("estimatedUSD should be positive, got %.6f", rec.EstimatedUSD)
	}
	if rec.RecordedAt.IsZero() {
		t.Fatalf("recordedAt should be parsed")
	}
}

func TestParseTranscriptLine_Sidechain(t *testing.T) {
	t.Parallel()

	rec, ok, err := store.ParseTranscriptLine([]byte(sampleSidechainLine))
	if err != nil || !ok || rec == nil {
		t.Fatalf("expected sidechain to parse: ok=%v err=%v", ok, err)
	}
	if rec.Source != core.CostSourceSidechain {
		t.Fatalf("source: got %q want sidechain", rec.Source)
	}
}

func TestParseTranscriptLine_SkipsUserRecord(t *testing.T) {
	t.Parallel()

	rec, ok, err := store.ParseTranscriptLine([]byte(sampleUserLine))
	if err != nil {
		t.Fatalf("user line should not error: %v", err)
	}
	if ok || rec != nil {
		t.Fatalf("user line should be skipped (ok=%v rec=%v)", ok, rec)
	}
}

func TestParseTranscriptLine_SkipsMissingUsage(t *testing.T) {
	t.Parallel()

	rec, ok, _ := store.ParseTranscriptLine([]byte(sampleAssistantNoUsage))
	if ok || rec != nil {
		t.Fatalf("assistant without usage should be skipped")
	}
}

func TestParseTranscriptLine_SkipsMissingModel(t *testing.T) {
	t.Parallel()

	rec, ok, _ := store.ParseTranscriptLine([]byte(sampleAssistantNoModel))
	if ok || rec != nil {
		t.Fatalf("assistant without model should be skipped")
	}
}

func TestParseTranscriptLine_RejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	_, ok, err := store.ParseTranscriptLine([]byte("{not json"))
	if err == nil {
		t.Fatalf("expected JSON decode error")
	}
	if ok {
		t.Fatalf("invalid JSON should not be ok")
	}
}

func TestParseTranscriptFile_RoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	contents := sampleAssistantLine + "\n" +
		sampleUserLine + "\n" +
		sampleSidechainLine + "\n" +
		sampleSonnet + "\n"
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	records, err := store.ParseTranscriptFile(path)
	if err != nil {
		t.Fatalf("parse file: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("expected 3 cost records, got %d", len(records))
	}
	if records[0].Model != "claude-opus-4-6" {
		t.Fatalf("first record model: %q", records[0].Model)
	}
	if records[1].Source != core.CostSourceSidechain {
		t.Fatalf("second record should be sidechain, got %q", records[1].Source)
	}
	if records[2].Model != "claude-sonnet-4-6" || records[2].CacheReadTokens != 500 {
		t.Fatalf("third record sanity: %+v", records[2])
	}
}

func TestParseTranscriptFile_EmptyFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "empty.jsonl")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatalf("write empty fixture: %v", err)
	}
	records, err := store.ParseTranscriptFile(path)
	if err != nil {
		t.Fatalf("empty file should not error: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected 0 records, got %d", len(records))
	}
}

func TestParseTranscriptFile_MalformedLineSkipped(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "mixed.jsonl")
	contents := "{not json\n" + sampleAssistantLine + "\n"
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	records, err := store.ParseTranscriptFile(path)
	if err != nil {
		t.Fatalf("malformed-line tolerated: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record after skipping malformed line, got %d", len(records))
	}
}

func TestParseTranscriptFile_MissingFile(t *testing.T) {
	t.Parallel()

	_, err := store.ParseTranscriptFile(filepath.Join(t.TempDir(), "nope.jsonl"))
	if !os.IsNotExist(err) {
		t.Fatalf("expected os.ErrNotExist, got %v", err)
	}
}
