package adapters

import (
	"errors"
	"testing"
)

func TestTmuxAdapterListsPanes(t *testing.T) {
	t.Parallel()

	adapter := NewTmuxAdapter(recordingOutputRunner{
		outputs: map[string][]byte{
			"tmux|list-sessions|-F|#{session_name}": []byte("demo\n"),
			"tmux|list-windows|-t|demo|-F|#{session_name}\t#{window_index}\t#{window_name}": []byte("demo\t1\teditor\n"),
			"tmux|list-panes|-t|demo:1|-F|#{session_name}\t#{window_index}\t#{pane_index}\t#{pane_active}\t#{pane_title}\t#{pane_current_command}\t#{pane_pid}\t#{pane_tty}": []byte("demo\t1\t0\t1\tops\tclaude\t4242\t/dev/ttys010\n"),
			"tmux|display-message|-p|#{session_name}:#{window_index}.#{pane_index}": []byte("demo:1.0\n"),
			"lsof|-a|-d|cwd|-p|4242|-Fn": []byte("p4242\nn/Users/User/projects/demo\n"),
		},
	})

	sessions, err := adapter.ListSessions()
	if err != nil {
		t.Fatalf("list tmux sessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 pane, got %#v", sessions)
	}
	if sessions[0].SessionRef != "tmux://demo:1.0" {
		t.Fatalf("unexpected session ref %q", sessions[0].SessionRef)
	}
	if !sessions[0].IsActive {
		t.Fatal("expected pane to be active")
	}
	if sessions[0].WorkingDirectory != "/Users/User/projects/demo" {
		t.Fatalf("unexpected working directory %q", sessions[0].WorkingDirectory)
	}
}

func TestTmuxAdapterCurrentPaneSessionRefRequiresTMUXEnv(t *testing.T) {
	t.Setenv("TMUX", "/tmp/tmux-1000/default,123,0")
	adapter := NewTmuxAdapter(recordingOutputRunner{
		outputs: map[string][]byte{
			"tmux|display-message|-p|#{session_name}:#{window_index}.#{pane_index}": []byte("demo:2.3\n"),
		},
	})

	if ref := adapter.CurrentPaneSessionRef(); ref != "tmux://demo:2.3" {
		t.Fatalf("unexpected current pane ref %q", ref)
	}
}

func TestTmuxAdapterListSessionsReturnsEmptyWhenProcessNotRunning(t *testing.T) {
	t.Parallel()

	adapter := NewTmuxAdapter(recordingOutputRunner{err: errors.New("boom")})
	sessions, err := adapter.ListSessions()
	if err != nil {
		t.Fatalf("expected nil error when process not running, got %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestParseTmuxSessionRefRejectsMalformedValues(t *testing.T) {
	t.Parallel()

	if _, err := ParseTmuxSessionRef("tmux://demo"); err == nil {
		t.Fatal("expected malformed ref to fail")
	}
}
