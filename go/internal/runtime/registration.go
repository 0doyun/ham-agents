package runtime

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/core"
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

type registrationContext struct {
	now         time.Time
	host        string
	provider    string
	displayName string
	projectPath string
	role        string
	sessionRef  string
}

func (r *Registry) RegisterManaged(ctx context.Context, input RegisterManagedInput) (core.Agent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	agents, err := r.store.LoadAgents(ctx)
	if err != nil {
		return core.Agent{}, err
	}

	registration, err := r.resolveRegistrationContext(
		input.Provider,
		"unknown",
		input.DisplayName,
		func(provider string) string { return provider + "-agent" },
		input.ProjectPath,
		input.Role,
		input.SessionRef,
	)
	if err != nil {
		return core.Agent{}, err
	}

	return r.registerAgent(ctx, agents, core.Agent{
		ID:                     r.idProvider(registration.now),
		DisplayName:            registration.displayName,
		Provider:               registration.provider,
		Host:                   registration.host,
		Mode:                   core.AgentModeManaged,
		ProjectPath:            registration.projectPath,
		Role:                   registration.role,
		Status:                 core.AgentStatusBooting,
		StatusConfidence:       1,
		StatusReason:           "Managed launch requested.",
		RegisteredAt:           registration.now,
		LastEventAt:            registration.now,
		LastUserVisibleSummary: "Managed session registered.",
		NotificationPolicy:     core.NotificationPolicyDefault,
		SessionRef:             registration.sessionRef,
		AvatarVariant:          "default",
	})
}

func (r *Registry) RegisterAttached(ctx context.Context, input RegisterAttachedInput) (core.Agent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	agents, err := r.store.LoadAgents(ctx)
	if err != nil {
		return core.Agent{}, err
	}

	registration, err := r.resolveRequiredRegistrationContext(
		input.Provider,
		"iterm2",
		input.DisplayName,
		func(string) string { return "attached-agent" },
		input.ProjectPath,
		input.Role,
		input.SessionRef,
		"session ref is required for attach",
	)
	if err != nil {
		return core.Agent{}, err
	}

	if existing, ok := existingObservedOrAttachedAgent(agents, core.AgentModeAttached, registration.sessionRef); ok {
		return r.mutateAgentLocked(ctx, existing.ID, func(agent *core.Agent, now time.Time) (*core.Event, error) {
			agent.DisplayName = registration.displayName
			agent.Provider = registration.provider
			agent.ProjectPath = registration.projectPath
			agent.Role = registration.role
			agent.LastEventAt = now
			agent.LastUserVisibleSummary = "Attached session re-associated."
			return &core.Event{
				AgentID:       agent.ID,
				Type:          core.EventTypeAgentReconnected,
				Summary:       "Attached session re-associated.",
				LifecycleMode: string(agent.Mode),
			}, nil
		})
	}

	return r.registerAgent(ctx, agents, core.Agent{
		ID:                     r.idProvider(registration.now),
		DisplayName:            registration.displayName,
		Provider:               registration.provider,
		Host:                   registration.host,
		Mode:                   core.AgentModeAttached,
		ProjectPath:            registration.projectPath,
		Role:                   registration.role,
		Status:                 core.AgentStatusIdle,
		StatusConfidence:       0.6,
		StatusReason:           "Attached to an existing iTerm session.",
		RegisteredAt:           registration.now,
		LastEventAt:            registration.now,
		LastUserVisibleSummary: "Attached session registered.",
		NotificationPolicy:     core.NotificationPolicyDefault,
		SessionRef:             registration.sessionRef,
		AvatarVariant:          "default",
	})
}

func (r *Registry) RegisterObserved(ctx context.Context, input RegisterObservedInput) (core.Agent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	agents, err := r.store.LoadAgents(ctx)
	if err != nil {
		return core.Agent{}, err
	}

	registration, err := r.resolveRequiredRegistrationContext(
		input.Provider,
		"log",
		input.DisplayName,
		func(string) string { return "observed-agent" },
		input.ProjectPath,
		input.Role,
		input.SessionRef,
		"observed source is required",
	)
	if err != nil {
		return core.Agent{}, err
	}

	if existing, ok := existingObservedOrAttachedAgent(agents, core.AgentModeObserved, registration.sessionRef); ok {
		return r.mutateAgentLocked(ctx, existing.ID, func(agent *core.Agent, now time.Time) (*core.Event, error) {
			agent.DisplayName = registration.displayName
			agent.Provider = registration.provider
			agent.ProjectPath = registration.projectPath
			agent.Role = registration.role
			agent.LastEventAt = now
			agent.LastUserVisibleSummary = "Observed source re-associated."
			return &core.Event{
				AgentID:       agent.ID,
				Type:          core.EventTypeAgentReconnected,
				Summary:       "Observed source re-associated.",
				LifecycleMode: string(agent.Mode),
			}, nil
		})
	}

	return r.registerAgent(ctx, agents, core.Agent{
		ID:                     r.idProvider(registration.now),
		DisplayName:            registration.displayName,
		Provider:               registration.provider,
		Host:                   registration.host,
		Mode:                   core.AgentModeObserved,
		ProjectPath:            registration.projectPath,
		Role:                   registration.role,
		Status:                 core.AgentStatusIdle,
		StatusConfidence:       0.35,
		StatusReason:           "Waiting for observed signals.",
		RegisteredAt:           registration.now,
		LastEventAt:            registration.now,
		LastUserVisibleSummary: "Observed source registered.",
		NotificationPolicy:     core.NotificationPolicyDefault,
		SessionRef:             registration.sessionRef,
		AvatarVariant:          "default",
	})
}

func existingObservedOrAttachedAgent(agents []core.Agent, mode core.AgentMode, sessionRef string) (core.Agent, bool) {
	for _, agent := range agents {
		if agent.Mode == mode && strings.TrimSpace(agent.SessionRef) == sessionRef {
			return agent, true
		}
	}
	return core.Agent{}, false
}

func (r *Registry) resolveRegistrationContext(
	provider string,
	defaultProvider string,
	displayName string,
	defaultDisplayName func(string) string,
	projectPath string,
	role string,
	sessionRef string,
) (registrationContext, error) {
	now := r.clock().UTC()
	host, err := r.hostname()
	if err != nil {
		host = "localhost"
	}

	resolvedProvider := strings.TrimSpace(provider)
	if resolvedProvider == "" {
		resolvedProvider = defaultProvider
	}

	resolvedDisplayName := strings.TrimSpace(displayName)
	if resolvedDisplayName == "" {
		resolvedDisplayName = defaultDisplayName(resolvedProvider)
	}

	resolvedProjectPath := strings.TrimSpace(projectPath)
	if resolvedProjectPath == "" {
		resolvedProjectPath, err = os.Getwd()
		if err != nil {
			return registrationContext{}, fmt.Errorf("resolve working directory: %w", err)
		}
	}

	return registrationContext{
		now:         now,
		host:        host,
		provider:    resolvedProvider,
		displayName: resolvedDisplayName,
		projectPath: resolvedProjectPath,
		role:        strings.TrimSpace(role),
		sessionRef:  strings.TrimSpace(sessionRef),
	}, nil
}

func (r *Registry) resolveRequiredRegistrationContext(
	provider string,
	defaultProvider string,
	displayName string,
	defaultDisplayName func(string) string,
	projectPath string,
	role string,
	sessionRef string,
	missingSessionRefMessage string,
) (registrationContext, error) {
	registration, err := r.resolveRegistrationContext(
		provider,
		defaultProvider,
		displayName,
		defaultDisplayName,
		projectPath,
		role,
		sessionRef,
	)
	if err != nil {
		return registrationContext{}, err
	}
	if registration.sessionRef == "" {
		return registrationContext{}, fmt.Errorf("%s", missingSessionRefMessage)
	}
	return registration, nil
}
