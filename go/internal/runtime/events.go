package runtime

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/core"
)

func (r *Registry) Events(ctx context.Context, limit int) ([]core.Event, error) {
	if r.eventStore == nil {
		return []core.Event{}, nil
	}

	events, err := r.eventStore.Load(ctx)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || len(events) <= limit {
		return events, nil
	}
	return events[len(events)-limit:], nil
}

const maxFollowWait = 60 * time.Second

func (r *Registry) FollowEvents(ctx context.Context, afterEventID string, limit int, wait time.Duration) ([]core.Event, error) {
	if wait > maxFollowWait {
		wait = maxFollowWait
	}
	pollInterval := 200 * time.Millisecond
	deadline := r.clock().Add(wait)

	for {
		events, err := r.Events(ctx, 0)
		if err != nil {
			return nil, err
		}

		followed := core.EventsAfterID(events, afterEventID, limit)
		if len(followed) > 0 || wait <= 0 {
			return followed, nil
		}

		if !deadline.After(r.clock()) {
			return []core.Event{}, nil
		}

		sleepDuration := pollInterval
		if remaining := time.Until(deadline); remaining < sleepDuration {
			sleepDuration = remaining
		}

		timer := time.NewTimer(sleepDuration)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func statusTransitionSummary(agent core.Agent) string {
	if agent.Mode == core.AgentModeObserved {
		if summary := strings.TrimSpace(agent.LastUserVisibleSummary); summary != "" {
			return fmt.Sprintf("Status changed to %s. %s", agent.Status, summary)
		}
	}
	if strings.TrimSpace(agent.StatusReason) == "" {
		return fmt.Sprintf("Status changed to %s.", agent.Status)
	}
	return fmt.Sprintf("Status changed to %s. %s", agent.Status, strings.TrimSpace(agent.StatusReason))
}

func eventPresentationHint(event core.Event) (label string, emphasis string, presentationSummary string) {
	lowerSummary := strings.ToLower(event.Summary)

	switch event.Type {
	case core.EventTypeAgentRegistered:
		switch {
		case strings.Contains(lowerSummary, "attached session registered"):
			return "Attached", "info", event.Summary
		case strings.Contains(lowerSummary, "observed source registered"):
			return "Observed", "info", event.Summary
		default:
			return "Managed", "info", event.Summary
		}
	case core.EventTypeAgentRoleUpdated:
		return "Role", "info", event.Summary
	case core.EventTypeAgentNotificationPolicyUpdated:
		return "Notifications", "info", event.Summary
	case core.EventTypeAgentLayoutChanged:
		return "Layout", "info", event.Summary
	case core.EventTypeAgentDisconnected:
		return "Disconnected", "warning", trimLifecyclePresentationSummary(event.Summary)
	case core.EventTypeAgentReconnected:
		return "Reconnected", "positive", trimLifecyclePresentationSummary(event.Summary)
	case core.EventTypeAgentRemoved:
		return "Stopped", "neutral", removalPresentationSummary(event)
	case core.EventTypeAgentStatusUpdated:
		switch {
		case strings.Contains(lowerSummary, "status changed to idle") &&
			(strings.Contains(lowerSummary, "connection restored") || strings.Contains(lowerSummary, "back online") || strings.Contains(lowerSummary, "connected again") || strings.Contains(lowerSummary, "reconnected")):
			return "Reconnected", "positive", trimLifecyclePresentationSummary(event.Summary)
		case strings.Contains(lowerSummary, "status changed to error"):
			return "Error", "warning", trimLifecyclePresentationSummary(event.Summary)
		case strings.Contains(lowerSummary, "status changed to waiting_input"):
			return "Needs Input", "warning", trimLifecyclePresentationSummary(event.Summary)
		case strings.Contains(lowerSummary, "status changed to running_tool"):
			return "Running Tool", "info", trimLifecyclePresentationSummary(event.Summary)
		case strings.Contains(lowerSummary, "status changed to reading"):
			return "Reading", "info", trimLifecyclePresentationSummary(event.Summary)
		case strings.Contains(lowerSummary, "status changed to booting"):
			return "Booting", "info", trimLifecyclePresentationSummary(event.Summary)
		case strings.Contains(lowerSummary, "status changed to thinking"):
			return "Thinking", "info", trimLifecyclePresentationSummary(event.Summary)
		case strings.Contains(lowerSummary, "status changed to sleeping"):
			return "Sleeping", "neutral", trimLifecyclePresentationSummary(event.Summary)
		case strings.Contains(lowerSummary, "status changed to done"):
			return "Done", "positive", trimLifecyclePresentationSummary(event.Summary)
		case strings.Contains(lowerSummary, "status changed to disconnected"):
			return "Disconnected", "warning", trimLifecyclePresentationSummary(event.Summary)
		case strings.Contains(lowerSummary, "status changed to idle"):
			return "Idle", "info", trimLifecyclePresentationSummary(event.Summary)
		default:
			return "Status", "info", event.Summary
		}
	default:
		return "", "", ""
	}
}

func removalPresentationSummary(event core.Event) string {
	status := strings.TrimSpace(event.LifecycleStatus)
	reason := strings.TrimSpace(event.LifecycleReason)

	if status == "" && reason == "" {
		return event.Summary
	}

	base := "Stopped tracking."
	if status != "" {
		base = fmt.Sprintf("Stopped tracking while %s.", humanLifecycleStatusForPresentation(status))
	}
	if reason == "" {
		return base
	}
	return fmt.Sprintf("%s %s", base, reason)
}

func humanLifecycleStatusForPresentation(status string) string {
	switch strings.TrimSpace(status) {
	case "waiting_input":
		return "waiting for input"
	default:
		return strings.TrimSpace(status)
	}
}

func trimLifecyclePresentationSummary(summary string) string {
	parts := strings.SplitN(summary, ". ", 2)
	if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
		return strings.TrimSpace(parts[1])
	}
	return summary
}
