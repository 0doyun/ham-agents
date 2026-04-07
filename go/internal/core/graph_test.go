package core

import (
	"testing"
	"time"
)

func makeAgent(id string, status AgentStatus, subAgentIDs ...string) Agent {
	var subs []SubAgentInfo
	for _, childID := range subAgentIDs {
		subs = append(subs, SubAgentInfo{
			AgentID:   childID,
			AgentType: "managed",
			Status:    AgentStatusIdle,
			StartTime: time.Now(),
		})
	}
	return Agent{
		ID:          id,
		DisplayName: id,
		Status:      status,
		SubAgents:   subs,
	}
}

// TestBuildSessionGraph_SingleRoot: 1 agent, no children.
func TestBuildSessionGraph_SingleRoot(t *testing.T) {
	agents := []Agent{
		makeAgent("a1", AgentStatusIdle),
	}
	g := BuildSessionGraph(agents)

	if len(g.Roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(g.Roots))
	}
	if g.Roots[0].Depth != 0 {
		t.Errorf("expected depth 0, got %d", g.Roots[0].Depth)
	}
	if g.TotalCount != 1 {
		t.Errorf("expected TotalCount=1, got %d", g.TotalCount)
	}
	if g.BlockedCount != 0 {
		t.Errorf("expected BlockedCount=0, got %d", g.BlockedCount)
	}
	if g.GeneratedAt.IsZero() {
		t.Error("GeneratedAt should be set")
	}
}

// TestBuildSessionGraph_RootWithTwoChildren: parent with 2 children in the input.
func TestBuildSessionGraph_RootWithTwoChildren(t *testing.T) {
	agents := []Agent{
		makeAgent("parent", AgentStatusIdle, "child1", "child2"),
		makeAgent("child1", AgentStatusIdle),
		makeAgent("child2", AgentStatusIdle),
	}
	g := BuildSessionGraph(agents)

	if len(g.Roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(g.Roots))
	}
	root := g.Roots[0]
	if root.Agent.ID != "parent" {
		t.Errorf("expected root ID=parent, got %s", root.Agent.ID)
	}
	if len(root.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(root.Children))
	}
	for _, child := range root.Children {
		if child.Depth != 1 {
			t.Errorf("expected child depth 1, got %d", child.Depth)
		}
	}
	if g.TotalCount != 3 {
		t.Errorf("expected TotalCount=3, got %d", g.TotalCount)
	}
}

// TestBuildSessionGraph_ThreeLevelNested: root → mid → leaf.
func TestBuildSessionGraph_ThreeLevelNested(t *testing.T) {
	agents := []Agent{
		makeAgent("root", AgentStatusIdle, "mid"),
		makeAgent("mid", AgentStatusIdle, "leaf"),
		makeAgent("leaf", AgentStatusIdle),
	}
	g := BuildSessionGraph(agents)

	if len(g.Roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(g.Roots))
	}
	root := g.Roots[0]
	if root.Depth != 0 {
		t.Errorf("root depth want 0, got %d", root.Depth)
	}
	if len(root.Children) != 1 {
		t.Fatalf("expected 1 mid child, got %d", len(root.Children))
	}
	mid := root.Children[0]
	if mid.Depth != 1 {
		t.Errorf("mid depth want 1, got %d", mid.Depth)
	}
	if len(mid.Children) != 1 {
		t.Fatalf("expected 1 leaf child, got %d", len(mid.Children))
	}
	leaf := mid.Children[0]
	if leaf.Depth != 2 {
		t.Errorf("leaf depth want 2, got %d", leaf.Depth)
	}
	if g.TotalCount != 3 {
		t.Errorf("expected TotalCount=3, got %d", g.TotalCount)
	}
}

// TestBuildSessionGraph_OrphanAgent: child referenced but not present in input → silently skipped.
func TestBuildSessionGraph_OrphanAgent(t *testing.T) {
	agents := []Agent{
		makeAgent("root", AgentStatusIdle, "missing-child"),
		makeAgent("other", AgentStatusIdle),
	}
	// Should not crash. "missing-child" is not in the slice.
	g := BuildSessionGraph(agents)

	// "other" is not a child of anyone and "root" is not a child → both are roots.
	if len(g.Roots) != 2 {
		t.Fatalf("expected 2 roots (root + other), got %d", len(g.Roots))
	}
	// root has no valid children (orphan skipped).
	var rootNode SessionNode
	for _, r := range g.Roots {
		if r.Agent.ID == "root" {
			rootNode = r
		}
	}
	if len(rootNode.Children) != 0 {
		t.Errorf("expected orphan child to be skipped, got %d children", len(rootNode.Children))
	}
	// TotalCount counts only agents that exist in byID.
	if g.TotalCount != 2 {
		t.Errorf("expected TotalCount=2 (only real agents), got %d", g.TotalCount)
	}
}

// TestBuildSessionGraph_BlockReasonMapping: verify each status maps to the correct BlockReason.
func TestBuildSessionGraph_BlockReasonMapping(t *testing.T) {
	agents := []Agent{
		makeAgent("idle-agent", AgentStatusIdle),
		makeAgent("waiting-agent", AgentStatusWaitingInput),
		makeAgent("error-agent", AgentStatusError),
		makeAgent("disconnected-agent", AgentStatusDisconnected),
		makeAgent("done-agent", AgentStatusDone),
		makeAgent("thinking-agent", AgentStatusThinking),
	}
	g := BuildSessionGraph(agents)

	byID := make(map[string]SessionNode)
	var collect func(nodes []SessionNode)
	collect = func(nodes []SessionNode) {
		for _, n := range nodes {
			byID[n.Agent.ID] = n
			collect(n.Children)
		}
	}
	collect(g.Roots)

	cases := []struct {
		id     string
		reason string
	}{
		{"idle-agent", ""},
		{"waiting-agent", "waiting_input"},
		{"error-agent", "permission_request"},
		{"disconnected-agent", "permission_request"},
		{"done-agent", "done"},
		{"thinking-agent", ""},
	}
	for _, tc := range cases {
		n, ok := byID[tc.id]
		if !ok {
			t.Errorf("agent %s not found in graph", tc.id)
			continue
		}
		if n.BlockReason != tc.reason {
			t.Errorf("agent %s: BlockReason want %q, got %q", tc.id, tc.reason, n.BlockReason)
		}
	}

	// BlockedCount: waiting + error + disconnected = 3 (done doesn't count, idle/thinking don't count)
	if g.BlockedCount != 3 {
		t.Errorf("expected BlockedCount=3, got %d", g.BlockedCount)
	}
}

// TestBuildSessionGraph_EmptySnapshot: empty input → empty graph.
func TestBuildSessionGraph_EmptySnapshot(t *testing.T) {
	g := BuildSessionGraph(nil)

	if len(g.Roots) != 0 {
		t.Errorf("expected no roots, got %d", len(g.Roots))
	}
	if g.TotalCount != 0 {
		t.Errorf("expected TotalCount=0, got %d", g.TotalCount)
	}
	if g.BlockedCount != 0 {
		t.Errorf("expected BlockedCount=0, got %d", g.BlockedCount)
	}
	if g.GeneratedAt.IsZero() {
		t.Error("GeneratedAt should be set even for empty input")
	}
}

// TestBuildSessionGraph_CycleDefense: a → b → a cycle must terminate and return sane output.
func TestBuildSessionGraph_CycleDefense(t *testing.T) {
	// a references b, b references a — cycle.
	agents := []Agent{
		makeAgent("a", AgentStatusIdle, "b"),
		makeAgent("b", AgentStatusIdle, "a"),
	}
	// Must not hang or panic.
	done := make(chan SessionGraph, 1)
	go func() {
		done <- BuildSessionGraph(agents)
	}()

	select {
	case g := <-done:
		// Either a or b will be a root (the one not listed as child of the other first in iteration).
		// Both can't be roots simultaneously if each references the other — but since both are in
		// childIDs (a is child of b, b is child of a), neither is a root. So Roots is empty.
		// Either way, the function must terminate and TotalCount must be sane (0 or small).
		if g.TotalCount > 2 {
			t.Errorf("cycle defense: TotalCount should be <=2, got %d", g.TotalCount)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("BuildSessionGraph did not terminate within 3s (possible infinite loop on cycle)")
	}
}
