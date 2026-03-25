package runtime

import (
	"context"

	"github.com/ham-agents/ham-agents/go/internal/core"
	"github.com/ham-agents/ham-agents/go/internal/inference"
)

func (r *Registry) RefreshObserved(ctx context.Context) error {
	agents, err := r.store.LoadAgents(ctx)
	if err != nil {
		return err
	}

	_, err = r.applyObservedRefresh(ctx, agents)
	return err
}

func (r *Registry) refreshObservedAgents(ctx context.Context, agents []core.Agent) ([]core.Agent, []core.Event, error) {
	if len(agents) == 0 {
		return agents, nil, nil
	}

	now := r.clock().UTC()
	refreshed := append([]core.Agent(nil), agents...)
	changed := false
	events := make([]core.Event, 0)

	for index, agent := range refreshed {
		if agent.Mode != core.AgentModeObserved {
			continue
		}

		updated := inference.RefreshObservedAgent(agent, now)
		if updated != agent {
			if updated.Status != agent.Status {
				events = append(events, core.Event{
					AgentID:             agent.ID,
					Type:                core.EventTypeAgentStatusUpdated,
					Summary:             statusTransitionSummary(updated),
					LifecycleStatus:     string(updated.Status),
					LifecycleMode:       string(updated.Mode),
					LifecycleReason:     updated.StatusReason,
					LifecycleConfidence: updated.StatusConfidence,
				})
			}
			refreshed[index] = updated
			changed = true
		}
	}

	if changed {
		// Persistence is handled by applyObservedRefresh so read-triggered and explicit
		// observed refreshes follow the same save-and-append boundary.
	}

	return refreshed, events, nil
}

func (r *Registry) applyObservedRefresh(ctx context.Context, agents []core.Agent) ([]core.Agent, error) {
	refreshed, events, err := r.refreshObservedAgents(ctx, agents)
	if err != nil {
		return nil, err
	}
	return r.applyRefreshedAgents(ctx, agents, refreshed, events)
}
