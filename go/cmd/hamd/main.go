package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

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
	registry := runtime.NewRegistry(store.NewFileAgentStore(statePath))

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
			fmt.Printf("hamd bootstrap ready: tracked=%d socket=%s state=%s\n", snapshot.TotalCount(), ipcConfig.SocketPath, statePath)
			return nil
		}
		return fmt.Errorf("long-running daemon mode is not implemented yet; use --once")
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
