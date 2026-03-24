package runtime

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/core"
	"github.com/ham-agents/ham-agents/go/internal/store"
)

type RegisterManagedInput struct {
	Provider    string
	DisplayName string
	ProjectPath string
	Role        string
	SessionRef  string
}

type RegisterAttachedInput struct {
	Provider    string
	DisplayName string
	ProjectPath string
	Role        string
	SessionRef  string
}

type RegisterObservedInput struct {
	Provider    string
	DisplayName string
	ProjectPath string
	Role        string
	SessionRef  string
}

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

func (r *Registry) RegisterManaged(ctx context.Context, input RegisterManagedInput) (core.Agent, error) {
	agents, err := r.store.LoadAgents(ctx)
	if err != nil {
		return core.Agent{}, err
	}

	now := r.clock().UTC()
	hostname, err := r.hostname()
	if err != nil {
		hostname = "localhost"
	}

	provider := strings.TrimSpace(input.Provider)
	if provider == "" {
		provider = "unknown"
	}

	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		displayName = provider + "-agent"
	}

	projectPath := strings.TrimSpace(input.ProjectPath)
	if projectPath == "" {
		projectPath, err = os.Getwd()
		if err != nil {
			return core.Agent{}, fmt.Errorf("resolve working directory: %w", err)
		}
	}

	agent := core.Agent{
		ID:                     r.idProvider(now),
		DisplayName:            displayName,
		Provider:               provider,
		Host:                   hostname,
		Mode:                   core.AgentModeManaged,
		ProjectPath:            projectPath,
		Role:                   strings.TrimSpace(input.Role),
		Status:                 core.AgentStatusBooting,
		StatusConfidence:       1,
		LastEventAt:            now,
		LastUserVisibleSummary: "Managed session registered.",
		NotificationPolicy:     core.NotificationPolicyDefault,
		SessionRef:             strings.TrimSpace(input.SessionRef),
		AvatarVariant:          "default",
	}

	agents = append(agents, agent)
	if err := r.store.SaveAgents(ctx, agents); err != nil {
		return core.Agent{}, err
	}

	if r.eventStore != nil {
		event := core.Event{
			ID:         fmt.Sprintf("event-%d", now.UnixNano()),
			AgentID:    agent.ID,
			Type:       core.EventTypeAgentRegistered,
			Summary:    agent.LastUserVisibleSummary,
			OccurredAt: now,
		}
		_ = r.eventStore.Append(ctx, event)
	}

	return agent, nil
}

func (r *Registry) RegisterAttached(ctx context.Context, input RegisterAttachedInput) (core.Agent, error) {
	agents, err := r.store.LoadAgents(ctx)
	if err != nil {
		return core.Agent{}, err
	}

	now := r.clock().UTC()
	hostname, err := r.hostname()
	if err != nil {
		hostname = "localhost"
	}

	sessionRef := strings.TrimSpace(input.SessionRef)
	if sessionRef == "" {
		return core.Agent{}, fmt.Errorf("session ref is required for attach")
	}

	provider := strings.TrimSpace(input.Provider)
	if provider == "" {
		provider = "iterm2"
	}

	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		displayName = "attached-agent"
	}

	projectPath := strings.TrimSpace(input.ProjectPath)
	if projectPath == "" {
		projectPath, err = os.Getwd()
		if err != nil {
			return core.Agent{}, fmt.Errorf("resolve working directory: %w", err)
		}
	}

	agent := core.Agent{
		ID:                     r.idProvider(now),
		DisplayName:            displayName,
		Provider:               provider,
		Host:                   hostname,
		Mode:                   core.AgentModeAttached,
		ProjectPath:            projectPath,
		Role:                   strings.TrimSpace(input.Role),
		Status:                 core.AgentStatusIdle,
		StatusConfidence:       0.6,
		LastEventAt:            now,
		LastUserVisibleSummary: "Attached session registered.",
		NotificationPolicy:     core.NotificationPolicyDefault,
		SessionRef:             sessionRef,
		AvatarVariant:          "default",
	}

	agents = append(agents, agent)
	if err := r.store.SaveAgents(ctx, agents); err != nil {
		return core.Agent{}, err
	}

	if r.eventStore != nil {
		event := core.Event{
			ID:         fmt.Sprintf("event-%d", now.UnixNano()),
			AgentID:    agent.ID,
			Type:       core.EventTypeAgentRegistered,
			Summary:    agent.LastUserVisibleSummary,
			OccurredAt: now,
		}
		_ = r.eventStore.Append(ctx, event)
	}

	return agent, nil
}

func (r *Registry) RegisterObserved(ctx context.Context, input RegisterObservedInput) (core.Agent, error) {
	agents, err := r.store.LoadAgents(ctx)
	if err != nil {
		return core.Agent{}, err
	}

	now := r.clock().UTC()
	hostname, err := r.hostname()
	if err != nil {
		hostname = "localhost"
	}

	sessionRef := strings.TrimSpace(input.SessionRef)
	if sessionRef == "" {
		return core.Agent{}, fmt.Errorf("observed source is required")
	}

	provider := strings.TrimSpace(input.Provider)
	if provider == "" {
		provider = "log"
	}

	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		displayName = "observed-agent"
	}

	projectPath := strings.TrimSpace(input.ProjectPath)
	if projectPath == "" {
		projectPath, err = os.Getwd()
		if err != nil {
			return core.Agent{}, fmt.Errorf("resolve working directory: %w", err)
		}
	}

	agent := core.Agent{
		ID:                     r.idProvider(now),
		DisplayName:            displayName,
		Provider:               provider,
		Host:                   hostname,
		Mode:                   core.AgentModeObserved,
		ProjectPath:            projectPath,
		Role:                   strings.TrimSpace(input.Role),
		Status:                 core.AgentStatusIdle,
		StatusConfidence:       0.35,
		LastEventAt:            now,
		LastUserVisibleSummary: "Observed source registered.",
		NotificationPolicy:     core.NotificationPolicyDefault,
		SessionRef:             sessionRef,
		AvatarVariant:          "default",
	}

	agents = append(agents, agent)
	if err := r.store.SaveAgents(ctx, agents); err != nil {
		return core.Agent{}, err
	}

	if r.eventStore != nil {
		event := core.Event{
			ID:         fmt.Sprintf("event-%d", now.UnixNano()),
			AgentID:    agent.ID,
			Type:       core.EventTypeAgentRegistered,
			Summary:    agent.LastUserVisibleSummary,
			OccurredAt: now,
		}
		_ = r.eventStore.Append(ctx, event)
	}

	return agent, nil
}

func (r *Registry) List(ctx context.Context) ([]core.Agent, error) {
	agents, err := r.store.LoadAgents(ctx)
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

	return core.RuntimeSnapshot{Agents: agents, GeneratedAt: r.clock().UTC()}, nil
}

func (r *Registry) UpdateNotificationPolicy(ctx context.Context, agentID string, policy core.NotificationPolicy) (core.Agent, error) {
	agents, err := r.store.LoadAgents(ctx)
	if err != nil {
		return core.Agent{}, err
	}

	for index, agent := range agents {
		if agent.ID != agentID {
			continue
		}

		agents[index].NotificationPolicy = policy
		agents[index].LastEventAt = r.clock().UTC()
		if err := r.store.SaveAgents(ctx, agents); err != nil {
			return core.Agent{}, err
		}
		return agents[index], nil
	}

	return core.Agent{}, fmt.Errorf("agent %q not found", agentID)
}

func (r *Registry) UpdateRole(ctx context.Context, agentID string, role string) (core.Agent, error) {
	agents, err := r.store.LoadAgents(ctx)
	if err != nil {
		return core.Agent{}, err
	}

	trimmedRole := strings.TrimSpace(role)

	for index, agent := range agents {
		if agent.ID != agentID {
			continue
		}

		agents[index].Role = trimmedRole
		agents[index].LastEventAt = r.clock().UTC()
		if err := r.store.SaveAgents(ctx, agents); err != nil {
			return core.Agent{}, err
		}
		return agents[index], nil
	}

	return core.Agent{}, fmt.Errorf("agent %q not found", agentID)
}

func (r *Registry) Remove(ctx context.Context, agentID string) error {
	agents, err := r.store.LoadAgents(ctx)
	if err != nil {
		return err
	}

	filtered := make([]core.Agent, 0, len(agents))
	removed := false
	for _, agent := range agents {
		if agent.ID == agentID {
			removed = true
			continue
		}
		filtered = append(filtered, agent)
	}

	if !removed {
		return fmt.Errorf("agent %q not found", agentID)
	}

	return r.store.SaveAgents(ctx, filtered)
}

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
