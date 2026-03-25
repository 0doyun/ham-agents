package core

import (
	"path/filepath"
	"sort"
	"strings"
)

type Workspace struct {
	ID            string   `json:"id"`
	DisplayName   string   `json:"display_name"`
	ProjectPath   string   `json:"project_path"`
	AgentIDs      []string `json:"agent_ids"`
	MemberTeamIDs []string `json:"member_team_ids,omitempty"`
}

func BuildWorkspaces(agents []Agent, teams []Team) []Workspace {
	agentToTeamIDs := make(map[string][]string)
	for _, team := range teams {
		for _, agentID := range team.MemberAgentIDs {
			agentToTeamIDs[agentID] = append(agentToTeamIDs[agentID], team.ID)
		}
	}

	byPath := make(map[string]*Workspace)
	for _, agent := range agents {
		projectPath := strings.TrimSpace(agent.ProjectPath)
		if projectPath == "" {
			continue
		}

		workspace, ok := byPath[projectPath]
		if !ok {
			displayName := filepath.Base(projectPath)
			if displayName == "." || displayName == string(filepath.Separator) || displayName == "" {
				displayName = projectPath
			}
			workspace = &Workspace{
				ID:          projectPath,
				DisplayName: displayName,
				ProjectPath: projectPath,
			}
			byPath[projectPath] = workspace
		}

		workspace.AgentIDs = append(workspace.AgentIDs, agent.ID)
		workspace.MemberTeamIDs = appendUniqueStrings(workspace.MemberTeamIDs, agentToTeamIDs[agent.ID]...)
	}

	workspaces := make([]Workspace, 0, len(byPath))
	for _, workspace := range byPath {
		sort.Strings(workspace.AgentIDs)
		sort.Strings(workspace.MemberTeamIDs)
		workspaces = append(workspaces, *workspace)
	}
	sort.SliceStable(workspaces, func(i, j int) bool {
		if workspaces[i].DisplayName == workspaces[j].DisplayName {
			return workspaces[i].ProjectPath < workspaces[j].ProjectPath
		}
		return workspaces[i].DisplayName < workspaces[j].DisplayName
	})
	return workspaces
}

func (w Workspace) Matches(ref string) bool {
	trimmed := strings.TrimSpace(ref)
	return w.ID == trimmed || w.ProjectPath == trimmed || w.DisplayName == trimmed
}

func appendUniqueStrings(current []string, values ...string) []string {
	if len(values) == 0 {
		return current
	}
	seen := make(map[string]struct{}, len(current))
	for _, value := range current {
		seen[value] = struct{}{}
	}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		current = append(current, trimmed)
	}
	return current
}
