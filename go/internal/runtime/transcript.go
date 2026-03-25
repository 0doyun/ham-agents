package runtime

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/ham-agents/ham-agents/go/internal/core"
)

func (r *Registry) EnsureObservedTranscripts(ctx context.Context, paths []string) error {
	agents, err := r.store.LoadAgents(ctx)
	if err != nil {
		return err
	}

	existing := make(map[string]struct{}, len(agents))
	for _, agent := range agents {
		if agent.Mode != core.AgentModeObserved {
			continue
		}
		existing[strings.TrimSpace(agent.SessionRef)] = struct{}{}
	}

	for _, path := range paths {
		trimmed := strings.TrimSpace(path)
		if trimmed == "" {
			continue
		}
		if _, ok := existing[trimmed]; ok {
			continue
		}
		if _, err := r.RegisterObserved(ctx, RegisterObservedInput{
			Provider:    "transcript",
			DisplayName: filepath.Base(trimmed),
			ProjectPath: filepath.Dir(trimmed),
			SessionRef:  trimmed,
		}); err != nil {
			return err
		}
		existing[trimmed] = struct{}{}
	}

	return r.RefreshObserved(ctx)
}
