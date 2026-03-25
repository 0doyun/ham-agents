package store_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/core"
	"github.com/ham-agents/ham-agents/go/internal/store"
)

func TestFileTeamStoreRoundTrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	teamStore := store.NewFileTeamStore(filepath.Join(t.TempDir(), "teams.json"))

	loaded, err := teamStore.LoadTeams(ctx)
	if err != nil {
		t.Fatalf("load missing team file: %v", err)
	}
	if len(loaded) != 0 {
		t.Fatalf("expected empty team list, got %#v", loaded)
	}

	teams := []core.Team{{
		ID:             "team-1",
		DisplayName:    "frontend",
		MemberAgentIDs: []string{"agent-1", "agent-2"},
		CreatedAt:      time.Unix(1, 0).UTC(),
		UpdatedAt:      time.Unix(2, 0).UTC(),
	}}

	if err := teamStore.SaveTeams(ctx, teams); err != nil {
		t.Fatalf("save teams: %v", err)
	}

	reloaded, err := teamStore.LoadTeams(ctx)
	if err != nil {
		t.Fatalf("reload teams: %v", err)
	}
	if len(reloaded) != 1 || reloaded[0].DisplayName != "frontend" {
		t.Fatalf("unexpected reloaded teams %#v", reloaded)
	}
}
