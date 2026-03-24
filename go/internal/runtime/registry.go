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

type Registry struct {
	store      store.AgentStore
	clock      func() time.Time
	idProvider func(time.Time) string
	hostname   func() (string, error)
}

func NewRegistry(agentStore store.AgentStore) *Registry {
	return &Registry{
		store: agentStore,
		clock: time.Now,
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
