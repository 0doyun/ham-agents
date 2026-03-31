package runtime

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/adapters"
	"github.com/ham-agents/ham-agents/go/internal/core"
)

func (r *Registry) RecordManagedStarted(ctx context.Context, agentID string, pid int, command string) (core.Agent, error) {
	return r.mutateAgent(ctx, agentID, func(agent *core.Agent, now time.Time) (*core.Event, error) {
		agent.SessionProcessID = pid
		agent.SessionCommand = strings.TrimSpace(command)
		agent.Status = core.AgentStatusThinking
		agent.StatusConfidence = 1
		agent.StatusReason = "Managed process started."
		agent.LastUserVisibleSummary = "Managed process is running."
		agent.LastEventAt = now
		return &core.Event{AgentID: agent.ID, Type: core.EventTypeAgentProcessStarted, Summary: fmt.Sprintf("Managed process started: %s", strings.TrimSpace(command)), LifecycleStatus: string(agent.Status), LifecycleMode: string(agent.Mode), LifecycleReason: agent.StatusReason, LifecycleConfidence: agent.StatusConfidence}, nil
	})
}

func (r *Registry) RecordManagedStartFailure(ctx context.Context, agentID string, message string) error {
	_, err := r.mutateAgent(ctx, agentID, func(agent *core.Agent, now time.Time) (*core.Event, error) {
		agent.Status = core.AgentStatusError
		agent.StatusConfidence = 1
		agent.StatusReason = "Managed process failed to start."
		agent.LastUserVisibleSummary = strings.TrimSpace(message)
		agent.LastEventAt = now
		return &core.Event{AgentID: agent.ID, Type: core.EventTypeAgentProcessExited, Summary: strings.TrimSpace(message), LifecycleStatus: string(agent.Status), LifecycleMode: string(agent.Mode), LifecycleReason: agent.StatusReason, LifecycleConfidence: agent.StatusConfidence}, nil
	})
	return err
}

func (r *Registry) RecordManagedOutput(ctx context.Context, agentID string, line string, isStderr bool, providerHintsEnabled bool) error {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return nil
	}
	_, err := r.mutateAgent(ctx, agentID, func(agent *core.Agent, now time.Time) (*core.Event, error) {
		if (agent.Status == core.AgentStatusDone || agent.Status == core.AgentStatusError) &&
			strings.HasPrefix(agent.StatusReason, "Managed process exited") {
			return nil, nil
		}
		status := core.AgentStatusThinking
		reason := "Managed process emitted output."
		summary := trimmed
		if providerHintsEnabled {
			if hintedStatus, hintedReason, hintedSummary, ok := adapters.ManagedProviderHint(agent.Provider, trimmed, isStderr); ok {
				status = hintedStatus
				reason = hintedReason
				if hintedSummary != "" {
					summary = hintedSummary
				}
			}
		}
		lower := strings.ToLower(trimmed)
		if strings.Contains(lower, "need input") || strings.Contains(lower, "needs input") || strings.Contains(lower, "approval") || strings.Contains(lower, "?") {
			status = core.AgentStatusWaitingInput
			reason = "Managed process is waiting for input."
		} else if strings.Contains(lower, "done") || strings.Contains(lower, "complete") || strings.Contains(lower, "finished successfully") {
			status = core.AgentStatusDone
			reason = "Managed process reported completion."
		} else if isStderr && (strings.Contains(lower, "error") || strings.Contains(lower, "failed")) {
			status = core.AgentStatusError
			reason = "Managed process emitted error output."
		}
		agent.Status = status
		agent.StatusConfidence = 1
		agent.StatusReason = reason
		agent.LastUserVisibleSummary = summary
		agent.LastEventAt = now
		return &core.Event{AgentID: agent.ID, Type: core.EventTypeAgentProcessOutput, Summary: summary, LifecycleStatus: string(agent.Status), LifecycleMode: string(agent.Mode), LifecycleReason: agent.StatusReason, LifecycleConfidence: agent.StatusConfidence}, nil
	})
	return err
}

func (r *Registry) RecordManagedStopped(ctx context.Context, agentID string) error {
	_, err := r.mutateAgent(ctx, agentID, func(agent *core.Agent, now time.Time) (*core.Event, error) {
		agent.Status = core.AgentStatusDone
		agent.StatusConfidence = 1
		agent.StatusReason = "Managed process stopped."
		if strings.TrimSpace(agent.LastUserVisibleSummary) == "" {
			agent.LastUserVisibleSummary = "Managed process stopped."
		}
		agent.LastEventAt = now
		return &core.Event{
			AgentID:             agent.ID,
			Type:                core.EventTypeAgentProcessExited,
			Summary:             "Managed process stopped.",
			LifecycleStatus:     string(agent.Status),
			LifecycleMode:       string(agent.Mode),
			LifecycleReason:     agent.StatusReason,
			LifecycleConfidence: agent.StatusConfidence,
		}, nil
	})
	return err
}

func (r *Registry) RecordManagedExit(ctx context.Context, agentID string, exitErr error) error {
	_, err := r.mutateAgent(ctx, agentID, func(agent *core.Agent, now time.Time) (*core.Event, error) {
		exitSummary := adapters.ClassifyProcessExit(exitErr)
		summary := agent.LastUserVisibleSummary
		if summary == "" || exitErr != nil {
			summary = exitSummary.Summary
		}
		agent.Status = exitSummary.Status
		agent.StatusConfidence = 1
		agent.StatusReason = exitSummary.Reason
		agent.LastUserVisibleSummary = summary
		agent.LastEventAt = now
		return &core.Event{AgentID: agent.ID, Type: core.EventTypeAgentProcessExited, Summary: summary, LifecycleStatus: string(agent.Status), LifecycleMode: string(agent.Mode), LifecycleReason: agent.StatusReason, LifecycleConfidence: agent.StatusConfidence}, nil
	})
	return err
}

func (r *Registry) RecordHookToolStart(ctx context.Context, agentID string, toolName string, toolInputPreview string, omcMode string) error {
	_, err := r.mutateAgent(ctx, agentID, func(agent *core.Agent, now time.Time) (*core.Event, error) {
		status := core.AgentStatusRunningTool
		lower := strings.ToLower(strings.TrimSpace(toolName))
		if lower == "read" || lower == "grep" || lower == "glob" {
			status = core.AgentStatusReading
		}
		summary := structuredToolSummary(toolName, toolInputPreview)
		applyOmcMode(agent, omcMode)
		agent.Status = status
		agent.StatusConfidence = 1
		agent.StatusReason = fmt.Sprintf("Hook: tool started: %s", toolName)
		agent.ErrorType = ""
		agent.LastUserVisibleSummary = summary
		pushRecentTool(agent, summary)
		agent.LastEventAt = now
		return &core.Event{
			AgentID:             agent.ID,
			Type:                core.EventTypeAgentProcessOutput,
			Summary:             summary,
			LifecycleStatus:     string(agent.Status),
			LifecycleMode:       string(agent.Mode),
			LifecycleReason:     agent.StatusReason,
			LifecycleConfidence: agent.StatusConfidence,
		}, nil
	})
	return err
}

func (r *Registry) RecordHookToolDone(ctx context.Context, agentID string, toolName string, toolInputPreview string, omcMode string) error {
	_, err := r.mutateAgent(ctx, agentID, func(agent *core.Agent, now time.Time) (*core.Event, error) {
		summary := structuredToolSummary(toolName, toolInputPreview)
		applyOmcMode(agent, omcMode)
		agent.Status = core.AgentStatusThinking
		agent.StatusConfidence = 1
		agent.StatusReason = fmt.Sprintf("Hook: tool completed: %s", toolName)
		agent.ErrorType = ""
		if summary != "" {
			agent.LastUserVisibleSummary = summary
		}
		agent.LastEventAt = now
		return &core.Event{
			AgentID:             agent.ID,
			Type:                core.EventTypeAgentProcessOutput,
			Summary:             "Completed " + summary,
			LifecycleStatus:     string(agent.Status),
			LifecycleMode:       string(agent.Mode),
			LifecycleReason:     agent.StatusReason,
			LifecycleConfidence: agent.StatusConfidence,
		}, nil
	})
	return err
}

func (r *Registry) RecordHookNotification(ctx context.Context, agentID string, notificationType string, omcMode string) error {
	trimmedType := strings.TrimSpace(notificationType)
	_, err := r.mutateAgent(ctx, agentID, func(agent *core.Agent, now time.Time) (*core.Event, error) {
		applyOmcMode(agent, omcMode)
		agent.LastEventAt = now
		summary := "Notification received."
		reason := "Hook: notification received."
		if trimmedType != "" {
			summary = fmt.Sprintf("Notification: %s", trimmedType)
			reason = fmt.Sprintf("Hook: notification received: %s", trimmedType)
		}
		agent.LastUserVisibleSummary = summary
		agent.StatusConfidence = 1
		agent.ErrorType = ""
		if trimmedType == "idle_prompt" || trimmedType == "permission_prompt" {
			agent.Status = core.AgentStatusWaitingInput
			agent.StatusReason = reason
		} else {
			agent.StatusReason = reason
		}
		return &core.Event{
			AgentID:             agent.ID,
			Type:                core.EventTypeAgentProcessOutput,
			Summary:             summary,
			LifecycleStatus:     string(agent.Status),
			LifecycleMode:       string(agent.Mode),
			LifecycleReason:     agent.StatusReason,
			LifecycleConfidence: agent.StatusConfidence,
		}, nil
	})
	return err
}

func (r *Registry) RecordHookStopFailure(ctx context.Context, agentID string, errorType string, omcMode string) error {
	trimmedType := strings.TrimSpace(errorType)
	_, err := r.mutateAgent(ctx, agentID, func(agent *core.Agent, now time.Time) (*core.Event, error) {
		applyOmcMode(agent, omcMode)
		agent.Status = core.AgentStatusError
		agent.StatusConfidence = 1
		agent.ErrorType = trimmedType
		reason := "Hook: stop failure."
		summary := "Session stop failed."
		if trimmedType != "" {
			reason = fmt.Sprintf("Hook: stop failure: %s", trimmedType)
			summary = fmt.Sprintf("Stop failure: %s", trimmedType)
		}
		agent.StatusReason = reason
		agent.LastUserVisibleSummary = summary
		agent.LastEventAt = now
		return &core.Event{
			AgentID:             agent.ID,
			Type:                core.EventTypeAgentProcessExited,
			Summary:             summary,
			LifecycleStatus:     string(agent.Status),
			LifecycleMode:       string(agent.Mode),
			LifecycleReason:     agent.StatusReason,
			LifecycleConfidence: agent.StatusConfidence,
		}, nil
	})
	return err
}

func (r *Registry) RecordHookSessionStart(ctx context.Context, agentID string, sessionID string, omcMode string) error {
	trimmedSessionID := strings.TrimSpace(sessionID)
	_, err := r.mutateAgent(ctx, agentID, func(agent *core.Agent, now time.Time) (*core.Event, error) {
		applyOmcMode(agent, omcMode)
		if trimmedSessionID != "" {
			agent.SessionID = trimmedSessionID
		}
		agent.Status = core.AgentStatusBooting
		agent.StatusConfidence = 1
		agent.StatusReason = "Hook: session started."
		agent.ErrorType = ""
		agent.LastUserVisibleSummary = "Session started via hook."
		agent.LastEventAt = now
		summary := agent.LastUserVisibleSummary
		if trimmedSessionID != "" {
			summary = fmt.Sprintf("Session started: %s", trimmedSessionID)
		}
		return &core.Event{
			AgentID:             agent.ID,
			Type:                core.EventTypeAgentProcessStarted,
			Summary:             summary,
			LifecycleStatus:     string(agent.Status),
			LifecycleMode:       string(agent.Mode),
			LifecycleReason:     agent.StatusReason,
			LifecycleConfidence: agent.StatusConfidence,
		}, nil
	})
	return err
}

func (r *Registry) RecordHookStop(ctx context.Context, agentID string, omcMode string) error {
	_, err := r.mutateAgent(ctx, agentID, func(agent *core.Agent, now time.Time) (*core.Event, error) {
		applyOmcMode(agent, omcMode)
		agent.Status = core.AgentStatusIdle
		agent.StatusConfidence = 1
		agent.StatusReason = "Hook: response completed."
		agent.LastUserVisibleSummary = "Waiting for next prompt."
		agent.LastEventAt = now
		return &core.Event{
			AgentID:             agent.ID,
			Type:                core.EventTypeAgentStatusUpdated,
			Summary:             "Response completed, waiting for input.",
			LifecycleStatus:     string(agent.Status),
			LifecycleMode:       string(agent.Mode),
			LifecycleReason:     agent.StatusReason,
			LifecycleConfidence: agent.StatusConfidence,
		}, nil
	})
	return err
}

func (r *Registry) RecordHookSessionEnd(ctx context.Context, agentID string, omcMode string) error {
	agents, err := r.store.LoadAgents(ctx)
	if err != nil {
		return err
	}
	filtered := make([]core.Agent, 0, len(agents))
	removed := false
	var removedAgent core.Agent
	for _, agent := range agents {
		if agent.ID == agentID {
			removed = true
			applyOmcMode(&agent, omcMode)
			agent.Status = core.AgentStatusDone
			agent.StatusConfidence = 1
			agent.StatusReason = "Hook: session ended."
			agent.LastUserVisibleSummary = "Session ended via hook."
			agent.LastEventAt = r.clock().UTC()
			removedAgent = agent
			continue
		}
		filtered = append(filtered, agent)
	}
	if !removed {
		return fmt.Errorf("agent %q not found", agentID)
	}
	return r.saveAgentsAndEvents(ctx, filtered, []core.Event{{
		AgentID:             removedAgent.ID,
		Type:                core.EventTypeAgentRemoved,
		Summary:             "Session ended via hook.",
		LifecycleStatus:     string(removedAgent.Status),
		LifecycleMode:       string(removedAgent.Mode),
		LifecycleReason:     removedAgent.StatusReason,
		LifecycleConfidence: removedAgent.StatusConfidence,
	}})
}

func (r *Registry) RecordHookAgentSpawned(ctx context.Context, agentID string, description string, omcMode string) error {
	_, err := r.mutateAgent(ctx, agentID, func(agent *core.Agent, now time.Time) (*core.Event, error) {
		applyOmcMode(agent, omcMode)
		agent.SubAgentCount++
		agent.LastEventAt = now
		summary := "Agent spawned"
		if description != "" {
			summary = fmt.Sprintf("Agent spawned: %s", description)
		}
		agent.LastUserVisibleSummary = summary
		pushRecentTool(agent, summary)
		return &core.Event{
			AgentID:             agent.ID,
			Type:                core.EventTypeAgentProcessOutput,
			Summary:             summary,
			LifecycleStatus:     string(agent.Status),
			LifecycleMode:       string(agent.Mode),
			LifecycleReason:     agent.StatusReason,
			LifecycleConfidence: agent.StatusConfidence,
		}, nil
	})
	return err
}

func (r *Registry) RecordHookAgentFinished(ctx context.Context, agentID string, description string, omcMode string) error {
	_, err := r.mutateAgent(ctx, agentID, func(agent *core.Agent, now time.Time) (*core.Event, error) {
		applyOmcMode(agent, omcMode)
		if agent.SubAgentCount > 0 {
			agent.SubAgentCount--
		}
		agent.LastEventAt = now
		summary := "Agent finished"
		if description != "" {
			summary = fmt.Sprintf("Agent finished: %s", description)
		}
		agent.LastUserVisibleSummary = summary
		return &core.Event{
			AgentID:             agent.ID,
			Type:                core.EventTypeAgentProcessOutput,
			Summary:             summary,
			LifecycleStatus:     string(agent.Status),
			LifecycleMode:       string(agent.Mode),
			LifecycleReason:     agent.StatusReason,
			LifecycleConfidence: agent.StatusConfidence,
		}, nil
	})
	return err
}

func (r *Registry) RecordHookTeammateIdle(ctx context.Context, agentID string, teammateName string, teamRole string, omcMode string) error {
	_, err := r.mutateAgent(ctx, agentID, func(agent *core.Agent, now time.Time) (*core.Event, error) {
		applyOmcMode(agent, omcMode)
		if teamRole != "" {
			agent.TeamRole = teamRole
		}
		agent.LastEventAt = now
		summary := "Teammate idle"
		if teammateName != "" {
			summary = fmt.Sprintf("Teammate idle: %s", teammateName)
		}
		agent.LastUserVisibleSummary = summary
		return &core.Event{
			AgentID:             agent.ID,
			Type:                core.EventTypeTeammateIdle,
			Summary:             summary,
			LifecycleStatus:     string(agent.Status),
			LifecycleMode:       string(agent.Mode),
			LifecycleReason:     agent.StatusReason,
			LifecycleConfidence: agent.StatusConfidence,
		}, nil
	})
	return err
}

func (r *Registry) RecordHookTaskCreated(ctx context.Context, agentID string, taskName string, taskDescription string, omcMode string) error {
	_, err := r.mutateAgent(ctx, agentID, func(agent *core.Agent, now time.Time) (*core.Event, error) {
		applyOmcMode(agent, omcMode)
		if agent.TeamRole == "" {
			agent.TeamRole = "lead"
		}
		agent.TeamTaskTotal++
		agent.LastEventAt = now
		summary := "Team task created"
		if taskName != "" {
			summary = fmt.Sprintf("Task created: %s", taskName)
		}
		agent.LastUserVisibleSummary = summary
		pushRecentTool(agent, summary)
		return &core.Event{
			AgentID:             agent.ID,
			Type:                core.EventTypeTeamTaskCreated,
			Summary:             summary,
			LifecycleStatus:     string(agent.Status),
			LifecycleMode:       string(agent.Mode),
			LifecycleReason:     agent.StatusReason,
			LifecycleConfidence: agent.StatusConfidence,
		}, nil
	})
	return err
}

func (r *Registry) RecordHookTaskCompleted(ctx context.Context, agentID string, taskName string, omcMode string) error {
	// If this agent has no tasks, find the team lead (agent with tasks) in the same project.
	targetID := agentID
	agents, _ := r.store.LoadAgents(ctx)
	var thisAgent *core.Agent
	for i := range agents {
		if agents[i].ID == agentID {
			thisAgent = &agents[i]
			break
		}
	}
	if thisAgent != nil && thisAgent.TeamTaskTotal == 0 {
		for _, a := range agents {
			if a.ID != agentID && a.TeamTaskTotal > 0 && a.ProjectPath == thisAgent.ProjectPath {
				targetID = a.ID
				break
			}
		}
	}

	_, err := r.mutateAgent(ctx, targetID, func(agent *core.Agent, now time.Time) (*core.Event, error) {
		applyOmcMode(agent, omcMode)
		if agent.TeamTaskCompleted < agent.TeamTaskTotal {
			agent.TeamTaskCompleted++
		}
		agent.LastEventAt = now
		summary := "Team task completed"
		if taskName != "" {
			summary = fmt.Sprintf("Task completed: %s", taskName)
		}
		agent.LastUserVisibleSummary = summary
		pushRecentTool(agent, summary)
		// Reset team data when all tasks are done.
		if agent.TeamTaskCompleted >= agent.TeamTaskTotal && agent.TeamTaskTotal > 0 {
			agent.TeamRole = ""
			agent.TeamTaskTotal = 0
			agent.TeamTaskCompleted = 0
			summary = fmt.Sprintf("All tasks completed (%s)", summary)
		}
		return &core.Event{
			AgentID:             agent.ID,
			Type:                core.EventTypeTeamTaskCompleted,
			Summary:             summary,
			LifecycleStatus:     string(agent.Status),
			LifecycleMode:       string(agent.Mode),
			LifecycleReason:     agent.StatusReason,
			LifecycleConfidence: agent.StatusConfidence,
		}, nil
	})
	return err
}

func structuredToolSummary(toolName string, toolInputPreview string) string {
	label := strings.TrimSpace(toolName)
	if label == "" {
		label = "Tool"
	}
	if preview := strings.TrimSpace(toolInputPreview); preview != "" {
		return fmt.Sprintf("%s: %s", label, preview)
	}
	return label
}

func pushRecentTool(agent *core.Agent, summary string) {
	trimmed := strings.TrimSpace(summary)
	if trimmed == "" {
		return
	}

	recent := []string{trimmed}
	for _, existing := range agent.RecentTools {
		if strings.TrimSpace(existing) == "" || existing == trimmed {
			continue
		}
		recent = append(recent, existing)
		if len(recent) >= 5 {
			break
		}
	}
	agent.RecentTools = recent
}

func applyOmcMode(agent *core.Agent, omcMode string) {
	if normalized := strings.TrimSpace(omcMode); normalized != "" {
		agent.OmcMode = normalized
	}
}
