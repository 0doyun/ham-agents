package store_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/core"
	"github.com/ham-agents/ham-agents/go/internal/store"
)

func TestFileAgentStoreRoundTrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	statePath := filepath.Join(t.TempDir(), "managed-agents.json")
	agentStore := store.NewFileAgentStore(statePath)

	loaded, err := agentStore.LoadAgents(ctx)
	if err != nil {
		t.Fatalf("load missing file: %v", err)
	}
	if len(loaded) != 0 {
		t.Fatalf("expected empty registry, got %d agents", len(loaded))
	}

	agents := []core.Agent{{
		ID:                 "agent-1",
		DisplayName:        "builder",
		Provider:           "claude",
		Host:               "localhost",
		Mode:               core.AgentModeManaged,
		ProjectPath:        "/tmp/project",
		Status:             core.AgentStatusThinking,
		StatusConfidence:   0.9,
		LastEventAt:        time.Unix(1700000000, 0).UTC(),
		NotificationPolicy: core.NotificationPolicyDefault,
		AvatarVariant:      "default",
	}}

	if err := agentStore.SaveAgents(ctx, agents); err != nil {
		t.Fatalf("save agents: %v", err)
	}

	reloaded, err := agentStore.LoadAgents(ctx)
	if err != nil {
		t.Fatalf("reload agents: %v", err)
	}
	if len(reloaded) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(reloaded))
	}
	if reloaded[0].DisplayName != "builder" {
		t.Fatalf("unexpected display name %q", reloaded[0].DisplayName)
	}
}

func TestFileEventStoreRoundTrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eventPath := filepath.Join(t.TempDir(), "events.jsonl")
	eventStore := store.NewFileEventStore(eventPath)

	event := core.Event{
		ID:         "event-1",
		AgentID:    "agent-1",
		Type:       core.EventTypeAgentRegistered,
		Summary:    "Managed session registered.",
		OccurredAt: time.Unix(1700000100, 0).UTC(),
	}

	if err := eventStore.Append(ctx, event); err != nil {
		t.Fatalf("append event: %v", err)
	}

	events, err := eventStore.Load(ctx)
	if err != nil {
		t.Fatalf("load events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != core.EventTypeAgentRegistered {
		t.Fatalf("unexpected event type %q", events[0].Type)
	}
}
