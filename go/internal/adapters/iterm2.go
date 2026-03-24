package adapters

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
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
	sessions, err := parseAttachableSessions(payload)
	if err != nil {
		return nil, err
	}
	return enrichAttachableSessions(sessions, a.runner), nil
}

func parseAttachableSessions(payload []byte) ([]core.AttachableSession, error) {
	trimmed := strings.TrimSpace(string(payload))
	if trimmed == "" {
		return []core.AttachableSession{}, nil
	}

	lines := strings.Split(trimmed, "\n")
	sessions := make([]core.AttachableSession, 0, len(lines))

	for _, line := range lines {
		fields := strings.SplitN(strings.TrimSpace(line), "\t", 4)
		if len(fields) < 2 {
			return nil, fmt.Errorf("invalid iTerm2 session row %q", line)
		}

		sessionID := strings.TrimSpace(fields[0])
		activeValue := strings.TrimSpace(fields[1])
		title := sessionID
		if len(fields) >= 3 && strings.TrimSpace(fields[2]) != "" {
			title = strings.TrimSpace(fields[2])
		}
		tty := ""
		if len(fields) == 4 {
			tty = strings.TrimSpace(fields[3])
		}

		if sessionID == "" {
			return nil, fmt.Errorf("invalid iTerm2 session row %q", line)
		}

		sessions = append(sessions, core.AttachableSession{
			ID:         sessionID,
			Title:      title,
			SessionRef: "iterm2://session/" + sessionID,
			IsActive:   activeValue == "true",
			TTY:        tty,
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
	script.WriteString(`                set sessionTTY to ""` + "\n")
	script.WriteString(`                try` + "\n")
	script.WriteString(`                    set sessionTTY to (tty of aSession) as string` + "\n")
	script.WriteString(`                end try` + "\n")
	script.WriteString(`                set sessionOutput to sessionOutput & sessionID & tab & activeFlag & tab & sessionName & tab & sessionTTY & linefeed` + "\n")
	script.WriteString(`            end repeat` + "\n")
	script.WriteString(`        end repeat` + "\n")
	script.WriteString(`    end repeat` + "\n")
	script.WriteString(`    return sessionOutput` + "\n")
	script.WriteString(`end tell`)
	return script.String()
}

func enrichAttachableSessions(sessions []core.AttachableSession, runner ScriptOutputRunner) []core.AttachableSession {
	if runner == nil {
		runner = ExecOutputRunner{}
	}

	for index, session := range sessions {
		if strings.TrimSpace(session.TTY) == "" {
			continue
		}
		activity, pid := sessionActivityForTTY(runner, session.TTY)
		if activity != "" {
			sessions[index].Activity = activity
		}
		if pid != "" {
			sessions[index].WorkingDirectory = workingDirectoryForPID(runner, pid)
		}
	}

	return sessions
}

func sessionActivityForTTY(runner ScriptOutputRunner, tty string) (activity string, pid string) {
	output, err := runner.Output("ps", "-ax", "-o", "tty=,pid=,comm=")
	if err != nil {
		return "", ""
	}

	normalizedTTY := strings.TrimPrefix(strings.TrimSpace(tty), "/dev/")
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 3 || fields[0] != normalizedTTY {
			continue
		}
		pid = fields[1]
		activity = filepath.Base(strings.Join(fields[2:], " "))
	}

	return activity, pid
}

func workingDirectoryForPID(runner ScriptOutputRunner, pid string) string {
	output, err := runner.Output("lsof", "-a", "-d", "cwd", "-p", pid, "-Fn")
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if strings.HasPrefix(line, "n") && len(line) > 1 {
			return strings.TrimSpace(strings.TrimPrefix(line, "n"))
		}
	}

	return ""
}
