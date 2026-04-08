package adapters

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ham-agents/ham-agents/go/internal/core"
)

const tmuxSessionRefPrefix = "tmux://"

type TmuxPaneRef struct {
	SessionName string
	WindowIndex int
	PaneIndex   int
}

func (r TmuxPaneRef) WindowTarget() string {
	return fmt.Sprintf("%s:%d", r.SessionName, r.WindowIndex)
}

func (r TmuxPaneRef) PaneTarget() string {
	return fmt.Sprintf("%s.%d", r.WindowTarget(), r.PaneIndex)
}

func (r TmuxPaneRef) SessionRef() string {
	return tmuxSessionRefPrefix + r.PaneTarget()
}

func ParseTmuxSessionRef(sessionRef string) (TmuxPaneRef, error) {
	trimmed := strings.TrimSpace(sessionRef)
	if !strings.HasPrefix(trimmed, tmuxSessionRefPrefix) {
		return TmuxPaneRef{}, fmt.Errorf("tmux session ref must start with %q", tmuxSessionRefPrefix)
	}

	target := strings.TrimPrefix(trimmed, tmuxSessionRefPrefix)
	dot := strings.LastIndex(target, ".")
	if dot <= 0 || dot == len(target)-1 {
		return TmuxPaneRef{}, fmt.Errorf("invalid tmux session ref %q", sessionRef)
	}

	colon := strings.LastIndex(target[:dot], ":")
	if colon <= 0 || colon == dot-1 {
		return TmuxPaneRef{}, fmt.Errorf("invalid tmux session ref %q", sessionRef)
	}

	windowIndex, err := strconv.Atoi(target[colon+1 : dot])
	if err != nil {
		return TmuxPaneRef{}, fmt.Errorf("parse tmux window index: %w", err)
	}
	paneIndex, err := strconv.Atoi(target[dot+1:])
	if err != nil {
		return TmuxPaneRef{}, fmt.Errorf("parse tmux pane index: %w", err)
	}

	return TmuxPaneRef{
		SessionName: strings.TrimSpace(target[:colon]),
		WindowIndex: windowIndex,
		PaneIndex:   paneIndex,
	}, nil
}

type TmuxAdapter struct {
	runner ScriptOutputRunner
}

func NewTmuxAdapter(runner ScriptOutputRunner) TmuxAdapter {
	if runner == nil {
		runner = ExecOutputRunner{}
	}
	return TmuxAdapter{runner: runner}
}

func (a TmuxAdapter) ListSessions() ([]core.AttachableSession, error) {
	if !processRunning(a.runner, "tmux") {
		return []core.AttachableSession{}, nil
	}
	sessionNames, err := a.sessionNames()
	if err != nil {
		return nil, err
	}

	currentTarget := ""
	if detected, err := a.currentPaneTarget(); err == nil {
		currentTarget = detected
	}

	sessions := make([]core.AttachableSession, 0)
	for _, sessionName := range sessionNames {
		windows, err := a.windows(sessionName)
		if err != nil {
			return nil, err
		}

		for _, window := range windows {
			panes, err := a.panes(window.target())
			if err != nil {
				return nil, err
			}

			for _, pane := range panes {
				ref := TmuxPaneRef{
					SessionName: pane.SessionName,
					WindowIndex: pane.WindowIndex,
					PaneIndex:   pane.PaneIndex,
				}
				title := pane.Title
				if title == "" {
					title = ref.PaneTarget()
				}
				if window.Name != "" && window.Name != title {
					title = fmt.Sprintf("%s — %s", title, window.Name)
				}

				session := core.AttachableSession{
					ID:               ref.PaneTarget(),
					Title:            title,
					SessionRef:       ref.SessionRef(),
					IsActive:         pane.Active || currentTarget == ref.PaneTarget(),
					TTY:              pane.TTY,
					Activity:         activityLabel(pane.CurrentCommand),
					ProcessID:        pane.PID,
					Command:          strings.TrimSpace(pane.CurrentCommand),
					WorkingDirectory: workingDirectoryForPID(a.runner, strconv.Itoa(pane.PID)),
				}
				sessions = append(sessions, normalizeAttachableSession(session))
			}
		}
	}

	return sessions, nil
}

func (a TmuxAdapter) CurrentPaneSessionRef() string {
	if strings.TrimSpace(os.Getenv("TMUX")) == "" {
		return ""
	}
	target, err := a.currentPaneTarget()
	if err != nil || target == "" {
		return ""
	}
	ref, err := ParseTmuxSessionRef(tmuxSessionRefPrefix + target)
	if err != nil {
		return ""
	}
	return ref.SessionRef()
}

type tmuxWindow struct {
	SessionName string
	Index       int
	Name        string
}

func (w tmuxWindow) target() string {
	return fmt.Sprintf("%s:%d", w.SessionName, w.Index)
}

type tmuxPane struct {
	SessionName    string
	WindowIndex    int
	PaneIndex      int
	Title          string
	CurrentCommand string
	PID            int
	TTY            string
	Active         bool
}

func (a TmuxAdapter) sessionNames() ([]string, error) {
	output, err := a.runner.Output("tmux", "list-sessions", "-F", "#{session_name}")
	if err != nil {
		return nil, fmt.Errorf("list tmux sessions: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	sessions := make([]string, 0, len(lines))
	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name != "" {
			sessions = append(sessions, name)
		}
	}
	return sessions, nil
}

func (a TmuxAdapter) windows(sessionName string) ([]tmuxWindow, error) {
	output, err := a.runner.Output("tmux", "list-windows", "-t", sessionName, "-F", "#{session_name}\t#{window_index}\t#{window_name}")
	if err != nil {
		return nil, fmt.Errorf("list tmux windows for %s: %w", sessionName, err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	windows := make([]tmuxWindow, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.SplitN(strings.TrimSpace(line), "\t", 3)
		if len(fields) < 2 {
			return nil, fmt.Errorf("invalid tmux window row %q", line)
		}
		index, err := strconv.Atoi(strings.TrimSpace(fields[1]))
		if err != nil {
			return nil, fmt.Errorf("parse tmux window row %q: %w", line, err)
		}
		name := ""
		if len(fields) >= 3 {
			name = strings.TrimSpace(fields[2])
		}
		windows = append(windows, tmuxWindow{
			SessionName: strings.TrimSpace(fields[0]),
			Index:       index,
			Name:        name,
		})
	}
	return windows, nil
}

func (a TmuxAdapter) panes(windowTarget string) ([]tmuxPane, error) {
	output, err := a.runner.Output("tmux", "list-panes", "-t", windowTarget, "-F", "#{session_name}\t#{window_index}\t#{pane_index}\t#{pane_active}\t#{pane_title}\t#{pane_current_command}\t#{pane_pid}\t#{pane_tty}")
	if err != nil {
		return nil, fmt.Errorf("list tmux panes for %s: %w", windowTarget, err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	panes := make([]tmuxPane, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.SplitN(strings.TrimSpace(line), "\t", 8)
		if len(fields) < 8 {
			return nil, fmt.Errorf("invalid tmux pane row %q", line)
		}
		windowIndex, err := strconv.Atoi(strings.TrimSpace(fields[1]))
		if err != nil {
			return nil, fmt.Errorf("parse tmux pane window index: %w", err)
		}
		paneIndex, err := strconv.Atoi(strings.TrimSpace(fields[2]))
		if err != nil {
			return nil, fmt.Errorf("parse tmux pane index: %w", err)
		}
		pid, err := strconv.Atoi(strings.TrimSpace(fields[6]))
		if err != nil {
			return nil, fmt.Errorf("parse tmux pane pid: %w", err)
		}
		panes = append(panes, tmuxPane{
			SessionName:    strings.TrimSpace(fields[0]),
			WindowIndex:    windowIndex,
			PaneIndex:      paneIndex,
			Active:         strings.TrimSpace(fields[3]) == "1",
			Title:          strings.TrimSpace(fields[4]),
			CurrentCommand: strings.TrimSpace(fields[5]),
			PID:            pid,
			TTY:            strings.TrimSpace(fields[7]),
		})
	}
	return panes, nil
}

func (a TmuxAdapter) currentPaneTarget() (string, error) {
	output, err := a.runner.Output("tmux", "display-message", "-p", "#{session_name}:#{window_index}.#{pane_index}")
	if err != nil {
		return "", fmt.Errorf("detect tmux pane target: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
