package adapters

import (
	"errors"
	"testing"
)

func TestIterm2AdapterListsSessions(t *testing.T) {
	t.Parallel()

	adapter := NewIterm2Adapter(recordingOutputRunner{
		outputs: map[string][]byte{
			"osascript":                   []byte("abc\ttrue\tClaude Review\tttys001\nxyz\tfalse\tShell\t\n"),
			"ps|-ax|-o|tty=,pid=,comm=":   []byte("ttys001 12345 /usr/local/bin/claude\n"),
			"lsof|-a|-d|cwd|-p|12345|-Fn": []byte("p12345\nn/Users/User/projects/demo\n"),
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
	if sessions[0].WorkingDirectory != "/Users/User/projects/demo" {
		t.Fatalf("unexpected working directory %q", sessions[0].WorkingDirectory)
	}
}

func TestIterm2AdapterListSessionsReturnsRunnerError(t *testing.T) {
	t.Parallel()

	adapter := NewIterm2Adapter(recordingOutputRunner{err: errors.New("boom")})
	if _, err := adapter.ListSessions(); err == nil {
		t.Fatal("expected list sessions error")
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
