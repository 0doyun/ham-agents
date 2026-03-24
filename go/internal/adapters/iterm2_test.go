package adapters

import (
	"errors"
	"testing"
)

func TestIterm2AdapterListsSessions(t *testing.T) {
	t.Parallel()

	adapter := NewIterm2Adapter(staticOutputRunner{
		payload: []byte("abc\ttrue\tClaude Review\nxyz\tfalse\tShell\n"),
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
}

func TestIterm2AdapterListSessionsReturnsRunnerError(t *testing.T) {
	t.Parallel()

	adapter := NewIterm2Adapter(staticOutputRunner{err: errors.New("boom")})
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

type staticOutputRunner struct {
	payload []byte
	err     error
}

func (r staticOutputRunner) Output(name string, args ...string) ([]byte, error) {
	_ = name
	_ = args
	if r.err != nil {
		return nil, r.err
	}
	return r.payload, nil
}
