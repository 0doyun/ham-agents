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

// Hook-based state tracking: Claude Code emits PreToolUse, PostToolUse, and Stop hooks.
// There is no hook for waiting_input (prompt idle / end_turn). That state is inferred
// via the PTY fallback path (silence detection). This is a Claude Code API limitation,
// not an omission — the hook system does not expose assistant turn boundaries.

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

func (r *Registry) RecordHookSessionEnd(ctx context.Context, agentID string, omcMode string) error {
	_, err := r.mutateAgent(ctx, agentID, func(agent *core.Agent, now time.Time) (*core.Event, error) {
		applyOmcMode(agent, omcMode)
		agent.Status = core.AgentStatusDone
		agent.StatusConfidence = 1
		agent.StatusReason = "Hook: session ended."
		agent.LastEventAt = now
		return &core.Event{
			AgentID:             agent.ID,
			Type:                core.EventTypeAgentProcessExited,
			Summary:             "Session ended via hook.",
			LifecycleStatus:     string(agent.Status),
			LifecycleMode:       string(agent.Mode),
			LifecycleReason:     agent.StatusReason,
			LifecycleConfidence: agent.StatusConfidence,
		}, nil
	})
	return err
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

func (r *Registry) RecordHookAgentFinished(ctx context.Context, agentID string, omcMode string) error {
	_, err := r.mutateAgent(ctx, agentID, func(agent *core.Agent, now time.Time) (*core.Event, error) {
		applyOmcMode(agent, omcMode)
		if agent.SubAgentCount > 0 {
			agent.SubAgentCount--
		}
		agent.LastEventAt = now
		agent.LastUserVisibleSummary = "Agent finished"
		return &core.Event{
			AgentID:             agent.ID,
			Type:                core.EventTypeAgentProcessOutput,
			Summary:             "Agent finished",
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
