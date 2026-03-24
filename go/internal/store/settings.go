package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ham-agents/ham-agents/go/internal/core"
)

type SettingsStore interface {
	Load(ctx context.Context) (core.Settings, error)
	Save(ctx context.Context, settings core.Settings) error
}

type FileSettingsStore struct {
	path string
	mu   sync.Mutex
}

func NewFileSettingsStore(path string) *FileSettingsStore {
	return &FileSettingsStore{path: path}
}

func DefaultSettingsPath() (string, error) {
	if root := os.Getenv("HAM_AGENTS_HOME"); root != "" {
		return filepath.Join(root, "settings.json"), nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}

	return filepath.Join(homeDir, "Library", "Application Support", "ham-agents", "settings.json"), nil
}

func (s *FileSettingsStore) Load(ctx context.Context) (core.Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return core.Settings{}, ctx.Err()
	default:
	}

	payload, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return core.DefaultSettings(), nil
		}
		return core.Settings{}, fmt.Errorf("read settings file: %w", err)
	}

	var settings core.Settings
	if err := json.Unmarshal(payload, &settings); err != nil {
		return core.Settings{}, fmt.Errorf("decode settings file: %w", err)
	}

	return settings, nil
}

func (s *FileSettingsStore) Save(ctx context.Context, settings core.Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create settings directory: %w", err)
	}

	payload, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, append(payload, '\n'), 0o644); err != nil {
		return fmt.Errorf("write temp settings: %w", err)
	}

	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("swap settings file: %w", err)
	}

	return nil
}
