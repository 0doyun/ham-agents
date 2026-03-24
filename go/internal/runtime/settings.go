package runtime

import (
	"context"

	"github.com/ham-agents/ham-agents/go/internal/core"
	"github.com/ham-agents/ham-agents/go/internal/store"
)

type SettingsService struct {
	store store.SettingsStore
}

func NewSettingsService(store store.SettingsStore) *SettingsService {
	return &SettingsService{store: store}
}

func (s *SettingsService) Get(ctx context.Context) (core.Settings, error) {
	settings, err := s.store.Load(ctx)
	if err != nil {
		return core.Settings{}, err
	}
	if err := settings.Validate(); err != nil {
		return core.Settings{}, err
	}
	return settings, nil
}

func (s *SettingsService) Update(ctx context.Context, settings core.Settings) (core.Settings, error) {
	if err := settings.Validate(); err != nil {
		return core.Settings{}, err
	}
	if err := s.store.Save(ctx, settings); err != nil {
		return core.Settings{}, err
	}
	return settings, nil
}
