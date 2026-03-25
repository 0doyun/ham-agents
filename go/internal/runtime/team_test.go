package runtime_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ham-agents/ham-agents/go/internal/runtime"
	"github.com/ham-agents/ham-agents/go/internal/store"
)

func TestTeamServiceCreateAndAddMember(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service := runtime.NewTeamService(store.NewFileTeamStore(filepath.Join(t.TempDir(), "teams.json")))

	created, err := service.Create(ctx, "frontend")
	if err != nil {
		t.Fatalf("create team: %v", err)
	}
	if created.DisplayName != "frontend" || len(created.MemberAgentIDs) != 0 {
		t.Fatalf("unexpected created team %#v", created)
	}

	updated, err := service.AddMember(ctx, created.ID, "agent-1")
	if err != nil {
		t.Fatalf("add member: %v", err)
	}
	if len(updated.MemberAgentIDs) != 1 || updated.MemberAgentIDs[0] != "agent-1" {
		t.Fatalf("unexpected updated team %#v", updated)
	}
	if !updated.UpdatedAt.After(created.CreatedAt) && !updated.UpdatedAt.Equal(created.CreatedAt) {
		t.Fatalf("expected updated_at to move forward, created=%s updated=%s", created.CreatedAt, updated.UpdatedAt)
	}
}

func TestTeamServiceFindMatchesDisplayName(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service := runtime.NewTeamService(store.NewFileTeamStore(filepath.Join(t.TempDir(), "teams.json")))

	created, err := service.Create(ctx, "backend")
	if err != nil {
		t.Fatalf("create team: %v", err)
	}

	found, err := service.Find(ctx, "backend")
	if err != nil {
		t.Fatalf("find team: %v", err)
	}
	if found.ID != created.ID {
		t.Fatalf("expected team %q, got %q", created.ID, found.ID)
	}
}
