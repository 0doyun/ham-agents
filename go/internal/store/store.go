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

type AgentStore interface {
	LoadAgents(ctx context.Context) ([]core.Agent, error)
	SaveAgents(ctx context.Context, agents []core.Agent) error
}

type FileAgentStore struct {
	path string
	mu   sync.Mutex
}

type persistedRegistry struct {
	Agents []core.Agent `json:"agents"`
}

func NewFileAgentStore(path string) *FileAgentStore {
	return &FileAgentStore{path: path}
}

func DefaultStatePath() (string, error) {
	if root := os.Getenv("HAM_AGENTS_HOME"); root != "" {
		return filepath.Join(root, "managed-agents.json"), nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}

	return filepath.Join(homeDir, "Library", "Application Support", "ham-agents", "managed-agents.json"), nil
}

func (s *FileAgentStore) LoadAgents(ctx context.Context) ([]core.Agent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.loadAgents(ctx)
}

func (s *FileAgentStore) SaveAgents(ctx context.Context, agents []core.Agent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}

	normalized := append([]core.Agent(nil), agents...)
	sort.SliceStable(normalized, func(i, j int) bool {
		if normalized[i].DisplayName == normalized[j].DisplayName {
			return normalized[i].ID < normalized[j].ID
		}
		return normalized[i].DisplayName < normalized[j].DisplayName
	})

	payload, err := json.MarshalIndent(persistedRegistry{Agents: normalized}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal agents: %w", err)
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, append(payload, '\n'), 0o644); err != nil {
		return fmt.Errorf("write temp registry: %w", err)
	}

	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("swap registry file: %w", err)
	}

	return nil
}

func (s *FileAgentStore) loadAgents(ctx context.Context) ([]core.Agent, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	payload, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []core.Agent{}, nil
		}
		return nil, fmt.Errorf("read registry file: %w", err)
	}

	if len(payload) == 0 {
		return []core.Agent{}, nil
	}

	var registry persistedRegistry
	if err := json.Unmarshal(payload, &registry); err != nil {
		return nil, fmt.Errorf("decode registry file: %w", err)
	}

	return registry.Agents, nil
}
