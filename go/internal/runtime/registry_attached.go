package runtime

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/adapters"
	"github.com/ham-agents/ham-agents/go/internal/core"
)

func (r *Registry) OpenTarget(ctx context.Context, agentID string) (core.OpenTarget, error) {
	agents, err := r.store.LoadAgents(ctx)
	if err != nil {
		return core.OpenTarget{}, err
	}

	for _, agent := range agents {
		if agent.ID != agentID {
			continue
		}

		sessionRef := strings.TrimSpace(agent.SessionRef)
		if sessionRef != "" {
			if target, ok := openTargetFromSessionRef(sessionRef); ok {
				return target, nil
			}
		}

		return core.OpenTarget{Kind: core.OpenTargetKindWorkspace, Value: agent.ProjectPath}, nil
	}

	return core.OpenTarget{}, fmt.Errorf("agent %q not found", agentID)
}

func (r *Registry) RefreshAttached(ctx context.Context, sessions []core.AttachableSession) error {
	return r.refreshAttachedWithScheme(ctx, "", sessions)
}

func (r *Registry) RefreshAttachedByScheme(ctx context.Context, scheme string, sessions []core.AttachableSession) error {
	return r.refreshAttachedWithScheme(ctx, scheme, sessions)
}

func (r *Registry) refreshAttachedWithScheme(ctx context.Context, scheme string, sessions []core.AttachableSession) error {
	agents, err := r.store.LoadAgents(ctx)
	if err != nil {
		return err
	}

	now := r.clock().UTC()
	refreshed, changed, changedEvents := refreshAttachedAgents(agents, sessions, now, scheme)
	if !changed {
		return nil
	}

	_, err = r.applyRefreshedAgents(ctx, agents, refreshed, changedEvents)
	return err
}

func openTargetFromSessionRef(sessionRef string) (core.OpenTarget, bool) {
	if _, err := adapters.ParseTmuxSessionRef(sessionRef); err == nil {
		return core.OpenTarget{
			Kind:  core.OpenTargetKindTmuxPane,
			Value: sessionRef,
		}, true
	}

	parsed, err := url.Parse(sessionRef)
	if err != nil || parsed.Scheme == "" {
		return core.OpenTarget{}, false
	}

	if parsed.Scheme == "iterm2" && parsed.Host == "session" {
		sessionID := strings.Trim(strings.TrimSpace(parsed.Path), "/")
		if sessionID != "" {
			return core.OpenTarget{
				Kind:        core.OpenTargetKindItermSession,
				Value:       sessionRef,
				Application: "iTerm",
				SessionID:   sessionID,
			}, true
		}
	}

	return core.OpenTarget{
		Kind:  core.OpenTargetKindExternalURL,
		Value: sessionRef,
	}, true
}

func refreshAttachedAgents(agents []core.Agent, sessions []core.AttachableSession, now time.Time, schemeFilter string) ([]core.Agent, bool, []core.Event) {
	if len(agents) == 0 {
		return agents, false, nil
	}

	sessionsByRef := make(map[string]core.AttachableSession, len(sessions))
	for _, session := range sessions {
		sessionsByRef[strings.TrimSpace(session.SessionRef)] = session
	}

	refreshed := append([]core.Agent(nil), agents...)
	changed := false
	events := make([]core.Event, 0)

	for index, agent := range refreshed {
		if agent.Mode != core.AgentModeAttached {
			continue
		}

		sessionRef := strings.TrimSpace(agent.SessionRef)
		if sessionRef == "" {
			continue
		}
		if schemeFilter != "" && sessionRefScheme(sessionRef) != schemeFilter {
			continue
		}

		session, attached := sessionsByRef[sessionRef]
		if attached {
			if refreshed[index].SessionTitle != session.Title {
				refreshed[index].SessionTitle = session.Title
				changed = true
			}
			if refreshed[index].SessionIsActive != session.IsActive {
				refreshed[index].SessionIsActive = session.IsActive
				changed = true
			}
			if refreshed[index].SessionTTY != session.TTY {
				refreshed[index].SessionTTY = session.TTY
				changed = true
			}
			layoutChanged := false
			if refreshed[index].SessionWindowIndex != session.WindowIndex {
				refreshed[index].SessionWindowIndex = session.WindowIndex
				layoutChanged = true
				changed = true
			}
			if refreshed[index].SessionTabIndex != session.TabIndex {
				refreshed[index].SessionTabIndex = session.TabIndex
				layoutChanged = true
				changed = true
			}
			if refreshed[index].SessionWorkingDirectory != session.WorkingDirectory {
				refreshed[index].SessionWorkingDirectory = session.WorkingDirectory
				changed = true
			}
			if refreshed[index].SessionActivity != session.Activity {
				refreshed[index].SessionActivity = session.Activity
				changed = true
			}
			if refreshed[index].SessionProcessID != session.ProcessID {
				refreshed[index].SessionProcessID = session.ProcessID
				changed = true
			}
			if refreshed[index].SessionCommand != session.Command {
				refreshed[index].SessionCommand = session.Command
				changed = true
			}
			if layoutChanged {
				refreshed[index].LastEventAt = now
				refreshed[index].LastUserVisibleSummary = "Attached session layout changed."
				events = append(events, core.Event{
					AgentID:             agent.ID,
					Type:                core.EventTypeAgentLayoutChanged,
					Summary:             "Attached session layout changed.",
					LifecycleStatus:     string(refreshed[index].Status),
					LifecycleMode:       string(refreshed[index].Mode),
					LifecycleReason:     refreshed[index].StatusReason,
					LifecycleConfidence: refreshed[index].StatusConfidence,
				})
			}
		}

		switch {
		case !attached && agent.Status != core.AgentStatusDisconnected:
			refreshed[index].Status = core.AgentStatusDisconnected
			refreshed[index].StatusConfidence = 0.75
			refreshed[index].StatusReason = "Session missing from iTerm session list."
			refreshed[index].SessionIsActive = false
			clearAttachedShellState(&refreshed[index])
			refreshed[index].LastEventAt = now
			refreshed[index].LastUserVisibleSummary = "Attached session disappeared from iTerm."
			events = append(events, core.Event{
				AgentID:             agent.ID,
				Type:                core.EventTypeAgentDisconnected,
				Summary:             statusTransitionSummary(refreshed[index]),
				LifecycleStatus:     string(refreshed[index].Status),
				LifecycleMode:       string(refreshed[index].Mode),
				LifecycleReason:     refreshed[index].StatusReason,
				LifecycleConfidence: refreshed[index].StatusConfidence,
			})
			changed = true
		case attached && agent.Status == core.AgentStatusDisconnected:
			refreshed[index].Status = core.AgentStatusIdle
			refreshed[index].StatusConfidence = 0.6
			refreshed[index].StatusReason = "Session reachable in iTerm again."
			refreshed[index].LastEventAt = now
			refreshed[index].LastUserVisibleSummary = "Attached session became reachable again."
			events = append(events, core.Event{
				AgentID:             agent.ID,
				Type:                core.EventTypeAgentReconnected,
				Summary:             statusTransitionSummary(refreshed[index]),
				LifecycleStatus:     string(refreshed[index].Status),
				LifecycleMode:       string(refreshed[index].Mode),
				LifecycleReason:     refreshed[index].StatusReason,
				LifecycleConfidence: refreshed[index].StatusConfidence,
			})
			changed = true
		}

		if attached && refreshed[index].Status != core.AgentStatusDisconnected {
			status, confidence, reason, summary := inferAttachedReachableStatus(refreshed[index])
			if refreshed[index].Status != status ||
				refreshed[index].StatusConfidence != confidence ||
				refreshed[index].StatusReason != reason ||
				refreshed[index].LastUserVisibleSummary != summary {
				statusChanged := refreshed[index].Status != status
				refreshed[index].Status = status
				refreshed[index].StatusConfidence = confidence
				refreshed[index].StatusReason = reason
				refreshed[index].LastUserVisibleSummary = summary
				refreshed[index].LastEventAt = now
				if statusChanged {
					events = append(events, core.Event{
						AgentID:             agent.ID,
						Type:                core.EventTypeAgentStatusUpdated,
						Summary:             statusTransitionSummary(refreshed[index]),
						LifecycleStatus:     string(refreshed[index].Status),
						LifecycleMode:       string(refreshed[index].Mode),
						LifecycleReason:     refreshed[index].StatusReason,
						LifecycleConfidence: refreshed[index].StatusConfidence,
					})
				}
				changed = true
			}
		}
	}

	return refreshed, changed, events
}

func sessionRefScheme(sessionRef string) string {
	if strings.HasPrefix(strings.TrimSpace(sessionRef), "tmux://") {
		return "tmux"
	}
	parsed, err := url.Parse(sessionRef)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(parsed.Scheme)
}

func clearAttachedShellState(agent *core.Agent) {
	agent.SessionTTY = ""
	agent.SessionWindowIndex = 0
	agent.SessionTabIndex = 0
	agent.SessionWorkingDirectory = ""
	agent.SessionActivity = ""
	agent.SessionProcessID = 0
	agent.SessionCommand = ""
}

func inferAttachedReachableStatus(agent core.Agent) (core.AgentStatus, float64, string, string) {
	signal := strings.ToLower(strings.TrimSpace(agent.SessionCommand + " " + agent.SessionActivity))

	switch {
	case strings.Contains(signal, "apply_patch"),
		strings.Contains(signal, "git "),
		strings.Contains(signal, "go test"),
		strings.Contains(signal, "swift test"),
		strings.Contains(signal, "npm "),
		strings.Contains(signal, "pnpm "),
		strings.Contains(signal, "cargo "),
		strings.Contains(signal, "pytest"):
		return core.AgentStatusRunningTool, 0.68, "Tool-like attached session activity detected.", "Attached session is running tool-like work."
	case strings.Contains(signal, "reading"),
		strings.Contains(signal, "reviewing"),
		strings.Contains(signal, "inspecting"),
		strings.Contains(signal, "less "),
		strings.Contains(signal, "cat "),
		strings.Contains(signal, "rg "),
		strings.Contains(signal, "grep "):
		return core.AgentStatusReading, 0.62, "Reading-like attached session activity detected.", "Attached session is reading output."
	default:
		return core.AgentStatusIdle, 0.6, "Attached to an existing iTerm session.", "Attached session is reachable."
	}
}
