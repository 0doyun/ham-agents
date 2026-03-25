package core

import "testing"

func TestBuildWorkspacesGroupsAgentsByProjectPathAndCollectsTeamMembership(t *testing.T) {
	t.Parallel()

	workspaces := BuildWorkspaces(
		[]Agent{
			{ID: "agent-1", ProjectPath: "/tmp/app"},
			{ID: "agent-2", ProjectPath: "/tmp/app"},
			{ID: "agent-3", ProjectPath: "/tmp/infra"},
		},
		[]Team{
			{ID: "team-1", DisplayName: "frontend", MemberAgentIDs: []string{"agent-1", "agent-2"}},
			{ID: "team-2", DisplayName: "ops", MemberAgentIDs: []string{"agent-3"}},
		},
	)

	if len(workspaces) != 2 {
		t.Fatalf("expected 2 workspaces, got %d", len(workspaces))
	}
	if workspaces[0].ProjectPath != "/tmp/app" {
		t.Fatalf("unexpected first workspace %#v", workspaces[0])
	}
	if got := workspaces[0].MemberTeamIDs; len(got) != 1 || got[0] != "team-1" {
		t.Fatalf("unexpected team ids %#v", got)
	}
	if got := workspaces[0].AgentIDs; len(got) != 2 || got[0] != "agent-1" || got[1] != "agent-2" {
		t.Fatalf("unexpected agent ids %#v", got)
	}
}

func TestWorkspaceMatchesByPathOrDisplayName(t *testing.T) {
	t.Parallel()

	workspace := Workspace{
		ID:          "/tmp/app",
		DisplayName: "app",
		ProjectPath: "/tmp/app",
	}

	if !workspace.Matches("/tmp/app") || !workspace.Matches("app") {
		t.Fatalf("expected workspace %#v to match path and display name", workspace)
	}
	if workspace.Matches("/tmp/other") {
		t.Fatalf("expected workspace %#v not to match unrelated path", workspace)
	}
}
