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
