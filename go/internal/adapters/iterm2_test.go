package adapters

import (
	"errors"
	"testing"

	"github.com/ham-agents/ham-agents/go/internal/core"
)

func TestIterm2AdapterListsSessions(t *testing.T) {
	t.Parallel()

	adapter := NewIterm2Adapter(recordingOutputRunner{
		outputs: map[string][]byte{
			"osascript":                    []byte("abc\ttrue\tClaude Review\tttys001\t1\t2\nxyz\tfalse\tShell\t\t2\t1\n"),
			"ps|-ax|-o|tty=,pid=,command=": []byte("ttys001 12345 /usr/local/bin/claude\n"),
			"lsof|-a|-d|cwd|-p|12345|-Fn":  []byte("p12345\nn/Users/User/projects/demo\n"),
		},
	})

	sessions, err := adapter.ListSessions()
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
	if sessions[0].SessionRef != "iterm2://session/abc" {
		t.Fatalf("unexpected session ref %q", sessions[0].SessionRef)
	}
	if !sessions[0].IsActive {
		t.Fatal("expected first session to be active")
	}
	if sessions[0].Activity != "claude" {
		t.Fatalf("expected activity claude, got %q", sessions[0].Activity)
	}
	if sessions[0].ProcessID != 12345 {
		t.Fatalf("expected process id 12345, got %d", sessions[0].ProcessID)
	}
	if sessions[0].Command != "/usr/local/bin/claude" {
		t.Fatalf("unexpected command %q", sessions[0].Command)
	}
	if sessions[0].WorkingDirectory != "/Users/User/projects/demo" {
		t.Fatalf("unexpected working directory %q", sessions[0].WorkingDirectory)
	}
	if sessions[0].WindowIndex != 1 || sessions[0].TabIndex != 2 {
		t.Fatalf("expected layout indices 1/2, got %d/%d", sessions[0].WindowIndex, sessions[0].TabIndex)
	}
}

func TestIterm2AdapterListSessionsReturnsRunnerError(t *testing.T) {
	t.Parallel()

	adapter := NewIterm2Adapter(recordingOutputRunner{err: errors.New("boom")})
	if _, err := adapter.ListSessions(); err == nil {
		t.Fatal("expected list sessions error")
	}
}

func TestIterm2AdapterPrefersForegroundToolOverShellNoise(t *testing.T) {
	t.Parallel()

	adapter := NewIterm2Adapter(recordingOutputRunner{
		outputs: map[string][]byte{
			"osascript":                    []byte("abc\ttrue\tClaude Review\tttys001\n"),
			"ps|-ax|-o|tty=,pid=,command=": []byte("ttys001 1200 -zsh\nttys001 1300 /usr/local/bin/claude chat\n"),
			"lsof|-a|-d|cwd|-p|1300|-Fn":   []byte("p1300\nn/Users/User/projects/demo\n"),
		},
	})

	sessions, err := adapter.ListSessions()
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}

	if sessions[0].ProcessID != 1300 {
		t.Fatalf("expected foreground tool pid 1300, got %d", sessions[0].ProcessID)
	}
	if sessions[0].Activity != "claude" {
		t.Fatalf("expected foreground activity claude, got %q", sessions[0].Activity)
	}
	if sessions[0].Command != "/usr/local/bin/claude chat" {
		t.Fatalf("unexpected command %q", sessions[0].Command)
	}
}

func TestSessionActivityFallsBackToShellLabelWhenOnlyShellPresent(t *testing.T) {
	t.Parallel()

	activity, pid, command := sessionActivityForTTY(recordingOutputRunner{
		outputs: map[string][]byte{
			"ps|-ax|-o|tty=,pid=,command=": []byte("ttys001 1200 /bin/zsh -l\n"),
		},
	}, "ttys001")

	if pid != 1200 {
		t.Fatalf("expected shell pid 1200, got %d", pid)
	}
	if activity != "shell" {
		t.Fatalf("expected shell activity label, got %q", activity)
	}
	if command != "/bin/zsh -l" {
		t.Fatalf("unexpected shell command %q", command)
	}
}

func TestNormalizeAttachableSessionDropsShellCommandNoise(t *testing.T) {
	t.Parallel()

	session := normalizeAttachableSession(core.AttachableSession{
		ID:         "abc",
		Title:      "Shell",
		SessionRef: "iterm2://session/abc",
		TTY:        "ttys001",
		Activity:   "shell",
		Command:    "/bin/zsh -l",
	})

	if session.Command != "" {
		t.Fatalf("expected shell command to be hidden, got %q", session.Command)
	}
	if session.TTY == "" {
		t.Fatal("expected tty to remain when shell context still exists")
	}
}

func TestParseAttachableSessionsRejectsMalformedRows(t *testing.T) {
	t.Parallel()

	if _, err := parseAttachableSessions([]byte("bad-row")); err == nil {
		t.Fatal("expected malformed rows to fail")
	}
}

type recordingOutputRunner struct {
	outputs map[string][]byte
	err     error
}

func (r recordingOutputRunner) Output(name string, args ...string) ([]byte, error) {
	if r.err != nil {
		return nil, r.err
	}
	if name == "osascript" {
		if payload, ok := r.outputs["osascript"]; ok {
			return payload, nil
		}
	}
	key := name
	for _, arg := range args {
		key += "|" + arg
	}
	return r.outputs[key], nil
}
