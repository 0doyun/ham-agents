package adapters

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTranscriptAdapterDiscoversSupportedFiles(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	for _, name := range []string{"a.log", "b.txt", "c.jsonl", "skip.bin"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("hi"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	adapter := NewTranscriptAdapter()
	sources, err := adapter.Discover([]string{root})
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(sources) != 3 {
		t.Fatalf("expected 3 transcript sources, got %#v", sources)
	}
}

func TestManagedProviderHintRecognizesClaudeToolUse(t *testing.T) {
	t.Parallel()
	status, reason, summary, ok := ManagedProviderHint("claude", `{"type":"tool_use"}`, false)
	if !ok || status == "" {
		t.Fatalf("expected provider hint, got ok=%v status=%q", ok, status)
	}
	if status != "running_tool" || reason == "" || summary == "" {
		t.Fatalf("unexpected hint values %q %q %q", status, reason, summary)
	}
}
