package store_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/core"
	"github.com/ham-agents/ham-agents/go/internal/store"
)

// TestFileArtifactStore_SaveAndLoad_RoundTrip saves 50 KB and loads via the
// returned ref, asserting bytes are identical.
func TestFileArtifactStore_SaveAndLoad_RoundTrip(t *testing.T) {
	t.Parallel()
	as := store.NewFileArtifactStore(t.TempDir())

	data := bytes.Repeat([]byte("x"), 50*1024)
	ref, err := as.Save("agent-1", "event-1", data)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := as.Load(ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !bytes.Equal(data, got) {
		t.Errorf("round-trip mismatch: saved %d bytes, loaded %d bytes", len(data), len(got))
	}
}

// TestFileArtifactStore_Save_CreatesParentDirs verifies the agent subdirectory
// is created automatically.
func TestFileArtifactStore_Save_CreatesParentDirs(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	as := store.NewFileArtifactStore(root)

	_, err := as.Save("agent-abc", "event-xyz", []byte("hello"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	agentDir := filepath.Join(root, "agent-abc")
	if _, err := os.Stat(agentDir); errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected agent dir %q to exist", agentDir)
	}
}

// TestFileArtifactStore_Load_NotFound expects an error when ref does not exist.
func TestFileArtifactStore_Load_NotFound(t *testing.T) {
	t.Parallel()
	as := store.NewFileArtifactStore(t.TempDir())

	_, err := as.Load("/nonexistent/path/that/does/not/exist.bin")
	if err == nil {
		t.Fatal("expected error loading nonexistent ref, got nil")
	}
}

// TestFileArtifactStore_Prune_LRU saves 5 files with staggered mtimes and
// calls Prune with a budget that fits only the 2 newest. Asserts the 3 oldest
// are removed and the 2 newest remain.
func TestFileArtifactStore_Prune_LRU(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	as := store.NewFileArtifactStore(root)

	const fileSize = 1024 // 1 KB each
	data := bytes.Repeat([]byte("a"), fileSize)

	type saved struct {
		ref   string
		mtime time.Time
	}
	files := make([]saved, 5)

	base := time.Now().Add(-10 * time.Hour)
	for i := 0; i < 5; i++ {
		ref, err := as.Save("agent-prune", "event-prune-"+string(rune('0'+i)), data)
		if err != nil {
			t.Fatalf("Save %d: %v", i, err)
		}
		mtime := base.Add(time.Duration(i) * time.Hour)
		if err := os.Chtimes(ref, mtime, mtime); err != nil {
			t.Fatalf("Chtimes %d: %v", i, err)
		}
		files[i] = saved{ref: ref, mtime: mtime}
	}

	// Budget: keep only 2 newest files (2 KB), prune the 3 oldest.
	budget := int64(2 * fileSize)
	if err := as.Prune(budget); err != nil {
		t.Fatalf("Prune: %v", err)
	}

	// 3 oldest should be gone.
	for i := 0; i < 3; i++ {
		if _, err := os.Stat(files[i].ref); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("file %d (oldest) should be pruned, stat err: %v", i, err)
		}
	}
	// 2 newest should remain.
	for i := 3; i < 5; i++ {
		if _, err := os.Stat(files[i].ref); err != nil {
			t.Errorf("file %d (newest) should remain, stat err: %v", i, err)
		}
	}
}

// TestFileEventStore_Append_OffloadsLargeArtifact appends an event with 50 KB
// ArtifactData, then reloads and asserts ArtifactData is empty and ArtifactRef
// points to the saved bytes.
func TestFileEventStore_Append_OffloadsLargeArtifact(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	as := store.NewFileArtifactStore(filepath.Join(dir, "artifacts"))
	es := store.NewFileEventStore(filepath.Join(dir, "events.jsonl")).WithArtifactStore(as)

	originalData := string(bytes.Repeat([]byte("z"), 50*1024))
	ev := core.Event{
		ID:           "ev-large",
		AgentID:      "agent-A",
		Type:         core.EventTypeAgentProcessOutput,
		ArtifactType: "text",
		ArtifactData: originalData,
	}

	ctx := context.Background()
	if err := es.Append(ctx, ev); err != nil {
		t.Fatalf("Append: %v", err)
	}

	events, err := es.Load(ctx)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	loaded := events[0]
	if loaded.ArtifactData != "" {
		t.Errorf("expected ArtifactData to be empty after offload, got %d chars", len(loaded.ArtifactData))
	}
	if loaded.ArtifactRef == "" {
		t.Error("expected non-empty ArtifactRef after offload")
	}

	// The artifact file must contain the original bytes.
	got, err := as.Load(loaded.ArtifactRef)
	if err != nil {
		t.Fatalf("artifact Load: %v", err)
	}
	if string(got) != originalData {
		t.Errorf("artifact content mismatch: got %d bytes, want %d", len(got), len(originalData))
	}
}

// TestFileEventStore_Append_InlineSmallArtifact ensures small ArtifactData
// (1 KB) stays inline and ArtifactRef remains empty.
func TestFileEventStore_Append_InlineSmallArtifact(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	as := store.NewFileArtifactStore(filepath.Join(dir, "artifacts"))
	es := store.NewFileEventStore(filepath.Join(dir, "events.jsonl")).WithArtifactStore(as)

	smallData := string(bytes.Repeat([]byte("s"), 1024))
	ev := core.Event{
		ID:           "ev-small",
		AgentID:      "agent-B",
		Type:         core.EventTypeAgentProcessOutput,
		ArtifactType: "text",
		ArtifactData: smallData,
	}

	ctx := context.Background()
	if err := es.Append(ctx, ev); err != nil {
		t.Fatalf("Append: %v", err)
	}

	events, err := es.Load(ctx)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	loaded := events[0]
	if loaded.ArtifactData != smallData {
		t.Errorf("expected ArtifactData inline, got %d chars", len(loaded.ArtifactData))
	}
	if loaded.ArtifactRef != "" {
		t.Errorf("expected empty ArtifactRef for small artifact, got %q", loaded.ArtifactRef)
	}
}

// TestFileEventStore_Append_TruncatesOverMax verifies that 2 MB ArtifactData
// is truncated to exactly 1 MB and ArtifactType is marked with ":truncated".
func TestFileEventStore_Append_TruncatesOverMax(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	as := store.NewFileArtifactStore(filepath.Join(dir, "artifacts"))
	es := store.NewFileEventStore(filepath.Join(dir, "events.jsonl")).WithArtifactStore(as)

	bigData := string(bytes.Repeat([]byte("b"), 2*1024*1024))
	ev := core.Event{
		ID:           "ev-big",
		AgentID:      "agent-C",
		Type:         core.EventTypeAgentProcessOutput,
		ArtifactType: "binary",
		ArtifactData: bigData,
	}

	ctx := context.Background()
	if err := es.Append(ctx, ev); err != nil {
		t.Fatalf("Append: %v", err)
	}

	events, err := es.Load(ctx)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	loaded := events[0]
	if !strings.HasSuffix(loaded.ArtifactType, ":truncated") {
		t.Errorf("expected ArtifactType to end with :truncated, got %q", loaded.ArtifactType)
	}

	got, err := as.Load(loaded.ArtifactRef)
	if err != nil {
		t.Fatalf("artifact Load: %v", err)
	}
	const wantSize = 1 * 1024 * 1024
	if len(got) != wantSize {
		t.Errorf("expected truncated artifact to be %d bytes, got %d", wantSize, len(got))
	}
}
