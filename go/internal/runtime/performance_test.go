package runtime_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/core"
	"github.com/ham-agents/ham-agents/go/internal/runtime"
	"github.com/ham-agents/ham-agents/go/internal/store"
)

func BenchmarkRegistrySnapshot100Agents(b *testing.B) {
	ctx := context.Background()
	root := b.TempDir()
	agentStore := store.NewFileAgentStore(filepath.Join(root, "managed-agents.json"))
	registry := runtime.NewRegistry(agentStore, store.NewFileEventStore(filepath.Join(root, "events.jsonl")))

	agents := make([]core.Agent, 0, 100)
	for index := 0; index < 100; index++ {
		agents = append(agents, core.Agent{
			ID:                 fmt.Sprintf("agent-%d", index),
			DisplayName:        fmt.Sprintf("agent-%d", index),
			Provider:           "claude",
			Host:               "localhost",
			Mode:               core.AgentModeManaged,
			ProjectPath:        filepath.Join(root, fmt.Sprintf("project-%d", index%5)),
			Status:             core.AgentStatusThinking,
			StatusConfidence:   1,
			LastEventAt:        time.Unix(int64(index), 0).UTC(),
			NotificationPolicy: core.NotificationPolicyDefault,
			AvatarVariant:      "default",
		})
	}
	if err := agentStore.SaveAgents(ctx, agents); err != nil {
		b.Fatalf("save agents: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := registry.Snapshot(ctx); err != nil {
			b.Fatalf("snapshot: %v", err)
		}
	}
}
