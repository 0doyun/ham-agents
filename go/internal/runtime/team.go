package runtime

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/core"
	"github.com/ham-agents/ham-agents/go/internal/store"
)

type TeamService struct {
	store      store.TeamStore
	clock      func() time.Time
	idProvider func(time.Time) string
}

func NewTeamService(teamStore store.TeamStore) *TeamService {
	return &TeamService{
		store: teamStore,
		clock: time.Now,
		idProvider: func(now time.Time) string {
			return fmt.Sprintf("team-%d", now.UnixNano())
		},
	}
}

func (s *TeamService) List(ctx context.Context) ([]core.Team, error) {
	return s.store.LoadTeams(ctx)
}

func (s *TeamService) Create(ctx context.Context, name string) (core.Team, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return core.Team{}, fmt.Errorf("team name is required")
	}
	teams, err := s.store.LoadTeams(ctx)
	if err != nil {
		return core.Team{}, err
	}
	for _, team := range teams {
		if team.DisplayName == trimmed {
			return core.Team{}, fmt.Errorf("team %q already exists", trimmed)
		}
	}
	now := s.clock().UTC()
	team := core.Team{ID: s.idProvider(now), DisplayName: trimmed, MemberAgentIDs: []string{}, CreatedAt: now, UpdatedAt: now}
	teams = append(teams, team)
	if err := s.store.SaveTeams(ctx, teams); err != nil {
		return core.Team{}, err
	}
	return team, nil
}

func (s *TeamService) AddMember(ctx context.Context, teamRef string, agentID string) (core.Team, error) {
	trimmedTeam := strings.TrimSpace(teamRef)
	trimmedAgent := strings.TrimSpace(agentID)
	if trimmedTeam == "" || trimmedAgent == "" {
		return core.Team{}, fmt.Errorf("team and agent are required")
	}
	teams, err := s.store.LoadTeams(ctx)
	if err != nil {
		return core.Team{}, err
	}
	for index := range teams {
		if !teams[index].Matches(trimmedTeam) {
			continue
		}
		for _, existing := range teams[index].MemberAgentIDs {
			if existing == trimmedAgent {
				teams[index].UpdatedAt = s.clock().UTC()
				if err := s.store.SaveTeams(ctx, teams); err != nil {
					return core.Team{}, err
				}
				return teams[index], nil
			}
		}
		teams[index].MemberAgentIDs = append(teams[index].MemberAgentIDs, trimmedAgent)
		teams[index].UpdatedAt = s.clock().UTC()
		if err := s.store.SaveTeams(ctx, teams); err != nil {
			return core.Team{}, err
		}
		return teams[index], nil
	}
	return core.Team{}, fmt.Errorf("team %q not found", trimmedTeam)
}

func (s *TeamService) Find(ctx context.Context, ref string) (core.Team, error) {
	teams, err := s.store.LoadTeams(ctx)
	if err != nil {
		return core.Team{}, err
	}
	trimmed := strings.TrimSpace(ref)
	for _, team := range teams {
		if team.Matches(trimmed) {
			return team, nil
		}
	}
	return core.Team{}, fmt.Errorf("team %q not found", trimmed)
}
