package runtime

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/core"
	"github.com/ham-agents/ham-agents/go/internal/inference"
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

	agents, err = r.refreshObservedAgents(ctx, agents)
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

func (r *Registry) FollowEvents(ctx context.Context, afterEventID string, limit int, wait time.Duration) ([]core.Event, error) {
	pollInterval := 200 * time.Millisecond
	deadline := r.clock().Add(wait)

	for {
		events, err := r.Events(ctx, 0)
		if err != nil {
			return nil, err
		}

		followed := eventsAfterID(events, afterEventID, limit)
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
	agents, err := r.store.LoadAgents(ctx)
	if err != nil {
		return err
	}

	now := r.clock().UTC()
	refreshed, changed := refreshAttachedAgents(agents, sessions, now)
	if !changed {
		return nil
	}

	return r.store.SaveAgents(ctx, refreshed)
}

func openTargetFromSessionRef(sessionRef string) (core.OpenTarget, bool) {
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

func refreshAttachedAgents(agents []core.Agent, sessions []core.AttachableSession, now time.Time) ([]core.Agent, bool) {
	if len(agents) == 0 {
		return agents, false
	}

	sessionsByRef := make(map[string]core.AttachableSession, len(sessions))
	for _, session := range sessions {
		sessionsByRef[strings.TrimSpace(session.SessionRef)] = session
	}

	refreshed := append([]core.Agent(nil), agents...)
	changed := false

	for index, agent := range refreshed {
		if agent.Mode != core.AgentModeAttached {
			continue
		}

		sessionRef := strings.TrimSpace(agent.SessionRef)
		if sessionRef == "" {
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
			if refreshed[index].SessionWorkingDirectory != session.WorkingDirectory {
				refreshed[index].SessionWorkingDirectory = session.WorkingDirectory
				changed = true
			}
			if refreshed[index].SessionActivity != session.Activity {
				refreshed[index].SessionActivity = session.Activity
				changed = true
			}
		}

		switch {
		case !attached && agent.Status != core.AgentStatusDisconnected:
			refreshed[index].Status = core.AgentStatusDisconnected
			refreshed[index].StatusConfidence = 0.75
			refreshed[index].SessionIsActive = false
			refreshed[index].LastEventAt = now
			refreshed[index].LastUserVisibleSummary = "Attached session disappeared from iTerm."
			changed = true
		case attached && agent.Status == core.AgentStatusDisconnected:
			refreshed[index].Status = core.AgentStatusIdle
			refreshed[index].StatusConfidence = 0.6
			refreshed[index].LastEventAt = now
			refreshed[index].LastUserVisibleSummary = "Attached session became reachable again."
			changed = true
		}
	}

	return refreshed, changed
}

func eventsAfterID(events []core.Event, afterEventID string, limit int) []core.Event {
	if afterEventID == "" {
		if limit <= 0 || len(events) <= limit {
			return events
		}
		return events[len(events)-limit:]
	}

	start := -1
	for index, event := range events {
		if event.ID == afterEventID {
			start = index + 1
			break
		}
	}
	if start == -1 {
		start = 0
	}
	if start >= len(events) {
		return []core.Event{}
	}

	followed := events[start:]
	if limit > 0 && len(followed) > limit {
		return followed[len(followed)-limit:]
	}
	return followed
}

func (r *Registry) RefreshObserved(ctx context.Context) error {
	agents, err := r.store.LoadAgents(ctx)
	if err != nil {
		return err
	}

	_, err = r.refreshObservedAgents(ctx, agents)
	return err
}

func (r *Registry) refreshObservedAgents(ctx context.Context, agents []core.Agent) ([]core.Agent, error) {
	if len(agents) == 0 {
		return agents, nil
	}

	now := r.clock().UTC()
	refreshed := append([]core.Agent(nil), agents...)
	changed := false

	for index, agent := range refreshed {
		if agent.Mode != core.AgentModeObserved {
			continue
		}

		updated := inference.RefreshObservedAgent(agent, now)
		if updated != agent {
			refreshed[index] = updated
			changed = true
		}
	}

	if changed {
		if err := r.store.SaveAgents(ctx, refreshed); err != nil {
			return nil, err
		}
	}

	return refreshed, nil
}
