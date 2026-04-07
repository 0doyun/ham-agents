package core

import "time"

// SessionNode represents one agent in the session tree.
type SessionNode struct {
	Agent       Agent         `json:"agent"`
	Children    []SessionNode `json:"children,omitempty"`
	BlockReason string        `json:"block_reason,omitempty"` // "waiting_input" | "permission_request" | "done" | "" (not blocked)
	Depth       int           `json:"depth"`
}

// SessionGraph is the full session tree returned by BuildSessionGraph.
type SessionGraph struct {
	Roots        []SessionNode `json:"roots"`
	TotalCount   int           `json:"total_count"`
	BlockedCount int           `json:"blocked_count"`
	GeneratedAt  time.Time     `json:"generated_at"`
}

// blockReasonForStatus maps an AgentStatus to a BlockReason string.
// Running and idle statuses return "" (not blocked).
func blockReasonForStatus(status AgentStatus) string {
	switch status {
	case AgentStatusWaitingInput:
		return "waiting_input"
	case AgentStatusError, AgentStatusDisconnected:
		return "permission_request"
	case AgentStatusDone:
		return "done"
	default:
		return ""
	}
}

// BuildSessionGraph constructs a tree from a flat list of agents.
// Parent-child links come from each Agent's SubAgents field. Cycles are
// defended against by tracking visited IDs during DFS. Orphan references
// (child IDs not present in the input slice) are silently skipped.
func BuildSessionGraph(agents []Agent) SessionGraph {
	byID := make(map[string]*Agent, len(agents))
	for i := range agents {
		byID[agents[i].ID] = &agents[i]
	}

	childIDs := make(map[string]bool)
	for _, a := range agents {
		for _, sub := range a.SubAgents {
			childIDs[sub.AgentID] = true
		}
	}

	var roots []SessionNode
	totalCount := 0
	blockedCount := 0

	var buildNode func(agentID string, depth int, visited map[string]bool) (SessionNode, bool)
	buildNode = func(agentID string, depth int, visited map[string]bool) (SessionNode, bool) {
		a, ok := byID[agentID]
		if !ok {
			// Orphan reference — agent not in input slice, skip.
			return SessionNode{}, false
		}
		if visited[agentID] {
			// Cycle detected — skip to avoid infinite recursion.
			return SessionNode{}, false
		}
		visited[agentID] = true
		defer func() { delete(visited, agentID) }()

		blockReason := blockReasonForStatus(a.Status)
		node := SessionNode{
			Agent:       *a,
			Depth:       depth,
			BlockReason: blockReason,
		}

		totalCount++
		if blockReason != "" && blockReason != "done" {
			blockedCount++
		}

		for _, sub := range a.SubAgents {
			child, ok := buildNode(sub.AgentID, depth+1, visited)
			if ok {
				node.Children = append(node.Children, child)
			}
		}

		return node, true
	}

	for _, a := range agents {
		if childIDs[a.ID] {
			continue
		}
		visited := make(map[string]bool)
		node, ok := buildNode(a.ID, 0, visited)
		if ok {
			roots = append(roots, node)
		}
	}

	return SessionGraph{
		Roots:        roots,
		TotalCount:   totalCount,
		BlockedCount: blockedCount,
		GeneratedAt:  time.Now().UTC(),
	}
}
