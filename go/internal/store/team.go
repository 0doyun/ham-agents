package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/ham-agents/ham-agents/go/internal/core"
)

type TeamStore interface {
	LoadTeams(ctx context.Context) ([]core.Team, error)
	SaveTeams(ctx context.Context, teams []core.Team) error
}

type FileTeamStore struct {
	path string
	mu   sync.Mutex
}

type persistedTeams struct {
	Teams []core.Team `json:"teams"`
}

func NewFileTeamStore(path string) *FileTeamStore { return &FileTeamStore{path: path} }

func DefaultTeamPath() (string, error) {
	if root := os.Getenv("HAM_AGENTS_HOME"); root != "" {
		return filepath.Join(root, "teams.json"), nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}
	return filepath.Join(homeDir, "Library", "Application Support", "ham-agents", "teams.json"), nil
}

func (s *FileTeamStore) LoadTeams(ctx context.Context) ([]core.Team, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	payload, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []core.Team{}, nil
		}
		return nil, fmt.Errorf("read teams file: %w", err)
	}
	if len(payload) == 0 {
		return []core.Team{}, nil
	}
	var teams persistedTeams
	if err := json.Unmarshal(payload, &teams); err != nil {
		return nil, fmt.Errorf("decode teams file: %w", err)
	}
	return teams.Teams, nil
}

func (s *FileTeamStore) SaveTeams(ctx context.Context, teams []core.Team) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create team directory: %w", err)
	}
	normalized := append([]core.Team(nil), teams...)
	sort.SliceStable(normalized, func(i, j int) bool {
		if normalized[i].DisplayName == normalized[j].DisplayName {
			return normalized[i].ID < normalized[j].ID
		}
		return normalized[i].DisplayName < normalized[j].DisplayName
	})
	payload, err := json.MarshalIndent(persistedTeams{Teams: normalized}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal teams: %w", err)
	}
	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, append(payload, '\n'), 0o644); err != nil {
		return fmt.Errorf("write temp teams: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("swap teams file: %w", err)
	}
	return nil
}
