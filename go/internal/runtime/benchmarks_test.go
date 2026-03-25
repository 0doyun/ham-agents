package runtime

import (
	"fmt"
	"testing"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/core"
)

func makeBenchmarkAgents(n int) []core.Agent {
	agents := make([]core.Agent, 0, n)
	statuses := []core.AgentStatus{
		core.AgentStatusThinking,
		core.AgentStatusRunningTool,
		core.AgentStatusWaitingInput,
		core.AgentStatusError,
		core.AgentStatusDone,
		core.AgentStatusIdle,
	}
	for i := 0; i < n; i++ {
		agents = append(agents, core.Agent{
			ID:               fmt.Sprintf("agent-%d", i),
			DisplayName:      fmt.Sprintf("agent-%d", i),
			ProjectPath:      fmt.Sprintf("/tmp/project-%d", i%20),
			Status:           statuses[i%len(statuses)],
			StatusConfidence: 1,
			LastEventAt:      time.Unix(int64(i), 0).UTC(),
		})
	}
	return agents
}

func BenchmarkSnapshotAttentionOrder(b *testing.B) {
	agents := makeBenchmarkAgents(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = snapshotAttentionOrder(agents)
	}
}

func BenchmarkSnapshotAttentionSubtitles(b *testing.B) {
	agents := makeBenchmarkAgents(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = snapshotAttentionSubtitles(agents)
	}
}
