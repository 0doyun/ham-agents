package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/adapters"
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
	settingsService := runtime.NewSettingsService(store.NewFileSettingsStore(settingsPath))
	teamService := runtime.NewTeamService(store.NewFileTeamStore(teamPath))
	itermAdapter := adapters.NewIterm2Adapter(nil)

	command := "serve"
	if len(args) > 0 {
		command = args[0]
		args = args[1:]
	}

	switch command {
	case "serve":
		flags := flag.NewFlagSet("serve", flag.ContinueOnError)
		flags.SetOutput(os.Stderr)
		once := flags.Bool("once", true, "emit bootstrap status and exit")
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

		server := ipc.NewServer(ipcConfig.SocketPath, registry, settingsService, teamService, itermAdapter)
		go pollRuntimeState(ctx, registry, itermAdapter, 2*time.Second)
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

func pollRuntimeState(ctx context.Context, registry *runtime.Registry, itermAdapter adapters.Iterm2Adapter, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = registry.RefreshObserved(ctx)
			sessions, err := itermAdapter.ListSessions()
			if err == nil {
				_ = registry.RefreshAttached(ctx, sessions)
			}
		}
	}
}
