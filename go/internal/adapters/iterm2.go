package adapters

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/ham-agents/ham-agents/go/internal/core"
)

type SessionLocator struct {
	Host        string
	Application string
	SessionID   string
}

type FocusRequest struct {
	Locator SessionLocator
}

type FocusResult struct {
	Supported bool
	Reason    string
}

type FocusAdapter interface {
	Focus(request FocusRequest) (FocusResult, error)
}

type SessionListing interface {
	ListSessions() ([]core.AttachableSession, error)
}

type ScriptOutputRunner interface {
	Output(name string, args ...string) ([]byte, error)
}

type ExecOutputRunner struct{}

func (ExecOutputRunner) Output(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

type Iterm2Adapter struct {
	runner ScriptOutputRunner
}

func NewIterm2Adapter(runner ScriptOutputRunner) Iterm2Adapter {
	if runner == nil {
		runner = ExecOutputRunner{}
	}
	return Iterm2Adapter{runner: runner}
}

func (a Iterm2Adapter) Focus(request FocusRequest) (FocusResult, error) {
	_ = request
	return FocusResult{
		Supported: false,
		Reason:    "iTerm2 focus automation is deferred; adapter boundary is bootstrapped.",
	}, nil
}

func (a Iterm2Adapter) ListSessions() ([]core.AttachableSession, error) {
	payload, err := a.runner.Output("osascript", "-e", listSessionsAppleScript())
	if err != nil {
		return nil, fmt.Errorf("list iTerm2 sessions: %w", err)
	}
	return parseAttachableSessions(payload)
}

func parseAttachableSessions(payload []byte) ([]core.AttachableSession, error) {
	trimmed := strings.TrimSpace(string(payload))
	if trimmed == "" {
		return []core.AttachableSession{}, nil
	}

	lines := strings.Split(trimmed, "\n")
	sessions := make([]core.AttachableSession, 0, len(lines))

	for _, line := range lines {
		fields := strings.SplitN(strings.TrimSpace(line), "\t", 3)
		if len(fields) < 2 {
			return nil, fmt.Errorf("invalid iTerm2 session row %q", line)
		}

		sessionID := strings.TrimSpace(fields[0])
		activeValue := strings.TrimSpace(fields[1])
		title := sessionID
		if len(fields) == 3 && strings.TrimSpace(fields[2]) != "" {
			title = strings.TrimSpace(fields[2])
		}

		if sessionID == "" {
			return nil, fmt.Errorf("invalid iTerm2 session row %q", line)
		}

		sessions = append(sessions, core.AttachableSession{
			ID:         sessionID,
			Title:      title,
			SessionRef: "iterm2://session/" + sessionID,
			IsActive:   activeValue == "true",
		})
	}

	return sessions, nil
}

func listSessionsAppleScript() string {
	var script bytes.Buffer
	script.WriteString(`tell application "iTerm"` + "\n")
	script.WriteString(`    set currentSessionID to ""` + "\n")
	script.WriteString(`    try` + "\n")
	script.WriteString(`        set currentSessionID to (id of current session of current window) as string` + "\n")
	script.WriteString(`    end try` + "\n")
	script.WriteString(`    set sessionOutput to ""` + "\n")
	script.WriteString(`    repeat with aWindow in windows` + "\n")
	script.WriteString(`        repeat with aTab in tabs of aWindow` + "\n")
	script.WriteString(`            repeat with aSession in sessions of aTab` + "\n")
	script.WriteString(`                set sessionID to (id of aSession) as string` + "\n")
	script.WriteString(`                set sessionName to (name of aSession) as string` + "\n")
	script.WriteString(`                set activeFlag to "false"` + "\n")
	script.WriteString(`                if currentSessionID is sessionID then set activeFlag to "true"` + "\n")
	script.WriteString(`                set sessionOutput to sessionOutput & sessionID & tab & activeFlag & tab & sessionName & linefeed` + "\n")
	script.WriteString(`            end repeat` + "\n")
	script.WriteString(`        end repeat` + "\n")
	script.WriteString(`    end repeat` + "\n")
	script.WriteString(`    return sessionOutput` + "\n")
	script.WriteString(`end tell`)
	return script.String()
}
