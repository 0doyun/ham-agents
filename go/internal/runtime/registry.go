package runtime

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/core"
	"github.com/ham-agents/ham-agents/go/internal/store"
)

type Registry struct {
	store      store.AgentStore
	eventStore store.EventStore
	clock      func() time.Time
	idProvider func(time.Time) string
	hostname   func() (string, error)
}

func NewRegistry(agentStore store.AgentStore, eventStore store.EventStore) *Registry {
	return &Registry{
		store:      agentStore,
		eventStore: eventStore,
		clock:      time.Now,
		idProvider: func(now time.Time) string {
			return fmt.Sprintf("managed-%d", now.UnixNano())
		},
		hostname: os.Hostname,
	}
}

func (r *Registry) List(ctx context.Context) ([]core.Agent, error) {
	agents, err := r.store.LoadAgents(ctx)
	if err != nil {
		return nil, err
	}

	agents, err = r.applyObservedRefresh(ctx, agents)
	if err != nil {
		return nil, err
	}

	sort.SliceStable(agents, func(i, j int) bool {
		if agents[i].DisplayName == agents[j].DisplayName {
			return agents[i].ID < agents[j].ID
		}
		return agents[i].DisplayName < agents[j].DisplayName
	})

	return agents, nil
}

func (r *Registry) Snapshot(ctx context.Context) (core.RuntimeSnapshot, error) {
	agents, err := r.List(ctx)
	if err != nil {
		return core.RuntimeSnapshot{}, err
	}

	attentionBreakdown := snapshotAttentionBreakdown(agents)

	return core.RuntimeSnapshot{
		Agents:             agents,
		GeneratedAt:        r.clock().UTC(),
		AttentionCount:     attentionBreakdown.Error + attentionBreakdown.WaitingInput + attentionBreakdown.Disconnected,
		AttentionBreakdown: attentionBreakdown,
		AttentionOrder:     snapshotAttentionOrder(agents),
		AttentionSubtitles: snapshotAttentionSubtitles(agents),
	}, nil
}

func (r *Registry) UpdateNotificationPolicy(ctx context.Context, agentID string, policy core.NotificationPolicy) (core.Agent, error) {
	return r.mutateAgent(ctx, agentID, func(agent *core.Agent, now time.Time) (*core.Event, error) {
		if agent.NotificationPolicy == policy {
			return nil, nil
		}
		agent.NotificationPolicy = policy
		agent.LastEventAt = now
		return &core.Event{
			AgentID:       agent.ID,
			Type:          core.EventTypeAgentNotificationPolicyUpdated,
			Summary:       fmt.Sprintf("Notification policy set to %s.", policy),
			LifecycleMode: string(agent.Mode),
		}, nil
	})
}

func (r *Registry) UpdateRole(ctx context.Context, agentID string, role string) (core.Agent, error) {
	trimmedRole := strings.TrimSpace(role)
	return r.mutateAgent(ctx, agentID, func(agent *core.Agent, now time.Time) (*core.Event, error) {
		if strings.TrimSpace(agent.Role) == trimmedRole {
			return nil, nil
		}
		agent.Role = trimmedRole
		agent.LastEventAt = now
		summary := "Role cleared."
		if trimmedRole != "" {
			summary = fmt.Sprintf("Role updated to %s.", trimmedRole)
		}
		return &core.Event{
			AgentID:       agent.ID,
			Type:          core.EventTypeAgentRoleUpdated,
			Summary:       summary,
			LifecycleMode: string(agent.Mode),
		}, nil
	})
}

func (r *Registry) Remove(ctx context.Context, agentID string) error {
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
			removedAgent = agent
			continue
		}
		filtered = append(filtered, agent)
	}

	if !removed {
		return fmt.Errorf("agent %q not found", agentID)
	}

	return r.saveAgentsAndEvents(ctx, filtered, []core.Event{{
		AgentID:             agentID,
		Type:                core.EventTypeAgentRemoved,
		Summary:             "Tracking stopped.",
		LifecycleStatus:     string(removedAgent.Status),
		LifecycleMode:       string(removedAgent.Mode),
		LifecycleReason:     removedAgent.StatusReason,
		LifecycleConfidence: removedAgent.StatusConfidence,
	}})
}

func (r *Registry) mutateAgent(
	ctx context.Context,
	agentID string,
	mutate func(agent *core.Agent, now time.Time) (*core.Event, error),
) (core.Agent, error) {
	agents, err := r.store.LoadAgents(ctx)
	if err != nil {
		return core.Agent{}, err
	}

	now := r.clock().UTC()
	for index := range agents {
		if agents[index].ID != agentID {
			continue
		}

		before := agents[index]
		event, err := mutate(&agents[index], now)
		if err != nil {
			return core.Agent{}, err
		}
		events := make([]core.Event, 0, 1)
		if event != nil {
			events = append(events, *event)
		}
		if event == nil && reflect.DeepEqual(agents[index], before) {
			return agents[index], nil
		}
		if err := r.saveAgentsAndEvents(ctx, agents, events); err != nil {
			return core.Agent{}, err
		}
		return agents[index], nil
	}

	return core.Agent{}, fmt.Errorf("agent %q not found", agentID)
}

func (r *Registry) registerAgent(ctx context.Context, agents []core.Agent, agent core.Agent) (core.Agent, error) {
	updatedAgents := append(append([]core.Agent(nil), agents...), agent)
	if err := r.saveAgentsAndEvents(ctx, updatedAgents, []core.Event{{
		AgentID:             agent.ID,
		Type:                core.EventTypeAgentRegistered,
		Summary:             agent.LastUserVisibleSummary,
		LifecycleStatus:     string(agent.Status),
		LifecycleMode:       string(agent.Mode),
		LifecycleReason:     agent.StatusReason,
		LifecycleConfidence: agent.StatusConfidence,
	}}); err != nil {
		return core.Agent{}, err
	}
	return agent, nil
}

func (r *Registry) applyRefreshedAgents(
	ctx context.Context,
	previous []core.Agent,
	refreshed []core.Agent,
	events []core.Event,
) ([]core.Agent, error) {
	if len(events) == 0 && agentsEqual(previous, refreshed) {
		return refreshed, nil
	}
	if err := r.saveAgentsAndEvents(ctx, refreshed, events); err != nil {
		return nil, err
	}
	return refreshed, nil
}

func (r *Registry) saveAgentsAndEvents(ctx context.Context, agents []core.Agent, events []core.Event) error {
	if err := r.store.SaveAgents(ctx, agents); err != nil {
		return err
	}
	for _, event := range events {
		r.appendEvent(ctx, event)
	}
	return nil
}

func (r *Registry) appendEvent(ctx context.Context, event core.Event) {
	if r.eventStore == nil {
		return
	}

	if event.ID == "" {
		event.ID = fmt.Sprintf("event-%d", r.clock().UTC().UnixNano())
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = r.clock().UTC()
	}
	if event.PresentationLabel == "" || event.PresentationEmphasis == "" || event.PresentationSummary == "" {
		label, emphasis, presentationSummary := eventPresentationHint(event)
		if event.PresentationLabel == "" {
			event.PresentationLabel = label
		}
		if event.PresentationEmphasis == "" {
			event.PresentationEmphasis = emphasis
		}
		if event.PresentationSummary == "" {
			event.PresentationSummary = presentationSummary
		}
	}
	_ = r.eventStore.Append(ctx, event)
}

func (r *Registry) RecordInformationalEvent(ctx context.Context, event core.Event) {
	r.appendEvent(ctx, event)
}

func agentsEqual(lhs []core.Agent, rhs []core.Agent) bool {
	if len(lhs) != len(rhs) {
		return false
	}
	for index := range lhs {
		if !reflect.DeepEqual(lhs[index], rhs[index]) {
			return false
		}
	}
	return true
}
