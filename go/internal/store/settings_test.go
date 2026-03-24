package store_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ham-agents/ham-agents/go/internal/store"
)

func TestFileSettingsStoreRoundTrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "settings.json")
	settingsStore := store.NewFileSettingsStore(path)

	settings, err := settingsStore.Load(ctx)
	if err != nil {
		t.Fatalf("load default settings: %v", err)
	}
	settings.Notifications.PreviewText = true

	if err := settingsStore.Save(ctx, settings); err != nil {
		t.Fatalf("save settings: %v", err)
	}

	reloaded, err := settingsStore.Load(ctx)
	if err != nil {
		t.Fatalf("reload settings: %v", err)
	}
	if !reloaded.Notifications.PreviewText {
		t.Fatal("expected preview text to persist")
	}
}
