package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/adapters"
	"github.com/ham-agents/ham-agents/go/internal/core"
	"github.com/ham-agents/ham-agents/go/internal/ipc"
	"github.com/ham-agents/ham-agents/go/internal/runtime"
	"github.com/ham-agents/ham-agents/go/internal/store"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "hamd: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	ctx := context.Background()
	statePath, err := store.DefaultStatePath()
	if err != nil {
		return err
	}
	ipcConfig, err := ipc.DefaultConfig()
	if err != nil {
		return err
	}
	eventPath, err := store.DefaultEventLogPath()
	if err != nil {
		return err
	}
	settingsPath, err := store.DefaultSettingsPath()
	if err != nil {
		return err
	}
	teamPath, err := store.DefaultTeamPath()
	if err != nil {
		return err
	}
	registry := runtime.NewRegistry(
		store.NewFileAgentStore(statePath),
		store.NewFileEventStore(eventPath),
	)
	managedService := runtime.NewManagedService(registry)
	settingsService := runtime.NewSettingsService(store.NewFileSettingsStore(settingsPath))
	teamService := runtime.NewTeamService(store.NewFileTeamStore(teamPath))
	itermAdapter := adapters.NewIterm2Adapter(nil)
	tmuxAdapter := adapters.NewTmuxAdapter(nil)
	transcriptAdapter := adapters.NewTranscriptAdapter()

	command := "serve"
	if len(args) > 0 {
		command = args[0]
		args = args[1:]
	}

	switch command {
	case "serve":
		flags := flag.NewFlagSet("serve", flag.ContinueOnError)
		flags.SetOutput(os.Stderr)
		once := flags.Bool("once", false, "emit bootstrap status and exit")
		if err := flags.Parse(args); err != nil {
			return err
		}
		snapshot, err := registry.Snapshot(ctx)
		if err != nil {
			return err
		}
		if *once {
			fmt.Printf("hamd bootstrap ready: tracked=%d socket=%s state=%s events=%s\n", snapshot.TotalCount(), ipcConfig.SocketPath, statePath, eventPath)
			return nil
		}

		// Acquire exclusive PID file lock to prevent duplicate daemons.
		cleanStalePIDFile(ipcConfig.SocketPath)
		pidFile, err := acquirePIDFileLock(ipcConfig.SocketPath)
		if err != nil {
			return err
		}
		defer func() {
			pidFile.Close()
			removePIDFile(ipcConfig.SocketPath)
		}()

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
		go func() {
			<-sigCh
			fmt.Fprintf(os.Stderr, "hamd: shutting down\n")
			cancel()
		}()

		server := ipc.NewServer(ipcConfig.SocketPath, registry, managedService, settingsService, teamService, itermAdapter, tmuxAdapter)
		go pollRuntimeState(ctx, registry, settingsService, itermAdapter, tmuxAdapter, transcriptAdapter, 2*time.Second)
		fmt.Printf("hamd serving on %s\n", ipcConfig.SocketPath)
		return server.Serve(ctx)
	case "snapshot":
		snapshot, err := registry.Snapshot(ctx)
		if err != nil {
			return err
		}
		payload, err := json.MarshalIndent(snapshot, "", "  ")
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", payload)
		return nil
	default:
		return fmt.Errorf("unsupported command %q", command)
	}
}

func pollRuntimeState(ctx context.Context, registry *runtime.Registry, settings *runtime.SettingsService, itermAdapter adapters.Iterm2Adapter, tmuxAdapter adapters.TmuxAdapter, transcriptAdapter adapters.TranscriptAdapter, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	heartbeatSentAt := map[string]time.Time{}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = registry.RefreshObserved(ctx)
			settingsSnapshot, err := settings.Get(ctx)
			if err == nil && settingsSnapshot.Integrations.ProviderAdapters["transcript"] {
				_ = ensureObservedTranscripts(ctx, registry, transcriptAdapter, settingsSnapshot.Integrations.TranscriptDirs)
			}
			if sessions, err := itermAdapter.ListSessions(); err == nil {
				_ = registry.RefreshAttachedByScheme(ctx, "iterm2", sessions)
			}
			if sessions, err := tmuxAdapter.ListSessions(); err == nil {
				_ = registry.RefreshAttachedByScheme(ctx, "tmux", sessions)
			}
			if err == nil {
				emitHeartbeatEvents(ctx, registry, settingsSnapshot, heartbeatSentAt)
			}
		}
	}
}

func emitHeartbeatEvents(ctx context.Context, registry *runtime.Registry, settings core.Settings, heartbeatSentAt map[string]time.Time) {
	if settings.Notifications.HeartbeatMinutes <= 0 {
		return
	}

	agents, err := registry.List(ctx)
	if err != nil {
		return
	}

	now := time.Now().UTC()
	interval := time.Duration(settings.Notifications.HeartbeatMinutes) * time.Minute
	for _, agent := range agents {
		if !heartbeatEligible(agent) || agent.RegisteredAt.IsZero() {
			continue
		}
		if now.Sub(agent.RegisteredAt) < interval {
			continue
		}
		if lastSent, ok := heartbeatSentAt[agent.ID]; ok && now.Sub(lastSent) < interval {
			continue
		}

		summary := fmt.Sprintf("Heartbeat: %dm in %s.", int(now.Sub(agent.RegisteredAt)/time.Minute), core.HumanAgentStatusLabel(agent.Status))
		if strings.TrimSpace(agent.LastUserVisibleSummary) != "" {
			summary += " Last: " + strings.TrimSpace(agent.LastUserVisibleSummary)
		}
		registry.RecordInformationalEvent(ctx, core.Event{
			AgentID:             agent.ID,
			Type:                core.EventTypeAgentProcessOutput,
			Summary:             summary,
			LifecycleStatus:     string(agent.Status),
			LifecycleMode:       string(agent.Mode),
			LifecycleReason:     "Heartbeat update emitted.",
			LifecycleConfidence: agent.StatusConfidence,
		})
		heartbeatSentAt[agent.ID] = now
	}
}

func heartbeatEligible(agent core.Agent) bool {
	switch agent.OmcMode {
	case "autopilot", "ralph", "team":
		return core.IsRunningStatus(agent.Status)
	default:
		return false
	}
}

func ensureObservedTranscripts(ctx context.Context, registry *runtime.Registry, adapter adapters.TranscriptAdapter, dirs []string) error {
	sources, err := adapter.Discover(dirs)
	if err != nil {
		return err
	}
	agents, err := registry.List(ctx)
	if err != nil {
		return err
	}
	existing := map[string]struct{}{}
	for _, agent := range agents {
		if agent.Mode == "observed" {
			existing[agent.SessionRef] = struct{}{}
		}
	}
	for _, source := range sources {
		if _, ok := existing[source.Path]; ok {
			continue
		}
		if _, err := registry.RegisterObserved(ctx, runtime.RegisterObservedInput{
			Provider:    "transcript",
			DisplayName: source.DisplayName,
			SessionRef:  source.Path,
		}); err != nil {
			return err
		}
	}
	return nil
}
