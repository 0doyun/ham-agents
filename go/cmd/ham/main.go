package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/ham-agents/ham-agents/go/internal/adapters"
	"github.com/ham-agents/ham-agents/go/internal/core"
	"github.com/ham-agents/ham-agents/go/internal/ipc"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "ham: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	ctx := context.Background()
	socketPath, err := ipc.DefaultSocketPath()
	if err != nil {
		return err
	}

	if len(args) == 0 {
		printHelp(socketPath)
		return nil
	}

	if commandRequiresDaemon(args[0]) {
		if err := ensureDaemon(socketPath); err != nil {
			return err
		}
	}

	client := ipc.NewClient(socketPath)

	switch args[0] {
	case "help", "--help", "-h":
		printHelp(socketPath)
		return nil
	case "run":
		return runRegister(ctx, client, args[1:])
	case "attach":
		return runAttach(ctx, client, args[1:])
	case "observe":
		return runObserve(ctx, client, args[1:])
	case "open":
		return runOpen(ctx, client, args[1:])
	case "ask":
		return runAsk(ctx, client, args[1:])
	case "stop":
		return runStop(ctx, client, args[1:])
	case "detach":
		return runDetach(ctx, client, args[1:])
	case "logs":
		return runLogs(ctx, client, args[1:])
	case "doctor":
		return runDoctor(socketPath, args[1:])
	case "ui":
		return runUI(args[1:])
	case "team":
		return runTeam(ctx, client, args[1:])
	case "hook":
		return runHook(ctx, client, args[1:])
	case "setup":
		return runSetup(args[1:], os.Stdin, os.Stdout, os.Stderr)
	case "settings":
		return runSettings(ctx, client, args[1:])
	case "list":
		return runList(ctx, client, args[1:])
	case "status":
		return runStatus(ctx, client, args[1:])
	case "events":
		return runEvents(ctx, client, args[1:])
	case "rename":
		return runRename(ctx, client, args[1:])
	case "inbox":
		return runInbox(ctx, client, args[1:])
	case "down":
		return runDown(ctx, client, args[1:])
	case "uninstall":
		return runUninstall(ctx, client, args[1:], os.Stdin, os.Stdout, os.Stderr)
	default:
		return fmt.Errorf("unsupported command %q", args[0])
	}
}

func printHelp(socketPath string) {
	fmt.Printf(`ham-agents Go CLI bootstrap

Usage:
  ham help
  ham run <provider> [name] [--project path] [--role role]
  ham attach <session-ref> [name] [--project path] [--role role] [--provider provider]
  ham attach --pick-iterm-session [--json] [--project path] [--role role] [--provider provider]
  ham attach --pick-tmux-session [--json] [--project path] [--role role] [--provider provider]
  ham attach --list-iterm-sessions [--json]
  ham attach --list-tmux-sessions [--json]
  ham observe <source-ref> [name] [--project path] [--role role] [--provider provider]
  ham open <agent-id> [--json] [--print]
  ham ask <agent-or-team> <message>
  ham rename <agent-id> <new-name>
  ham stop <agent-id> [--json]
  ham detach <agent-id> [--json]
  ham logs <agent-id> [--json] [--limit N] [--export path]
  ham doctor [--json]
  ham ui [--json] [--print]
  ham team create <name> [--json]
  ham team add <team> <agent-id> [--json]
  ham team list [--json]
  ham team open <team>
  ham settings [--json]
  ham settings general [--launch-at-login=true|false] [--compact-mode=true|false] [--show-menu-bar-animation-always=true|false]
  ham settings notifications [--done=true|false] [--error=true|false] [--waiting-input=true|false] [--silence=true|false] [--quiet-hours=true|false] [--quiet-start-hour=0-23] [--quiet-end-hour=0-23] [--preview-text=true|false] [--heartbeat-minutes=0|10|30|60]
  ham settings appearance [--theme=auto|day|night] [--animation-speed=0.25-3] [--reduce-motion=true|false] [--hamster-skin=name] [--hat=name] [--desk-theme=name]
  ham settings integrations [--iterm-enabled=true|false] [--transcript-dirs=dir1,dir2] [--provider-adapter=name=true|false]
  ham settings privacy [--local-only-mode=true|false] [--event-history-retention-days=N] [--transcript-excerpt-storage=true|false]
  ham setup
  ham hook tool-start <tool-name>
  ham hook tool-done <tool-name>
  ham hook notification
  ham hook stop-failure
  ham hook session-start
  ham hook session-end
  ham hook subagent-start [--description ...]
  ham hook subagent-stop [--description ...]
  ham list [--json] [--team ref] [--workspace ref]
  ham status [--json] [--team ref] [--workspace ref]
  ham events [--json] [--limit N] [--follow] [--after-id ID] [--wait-ms N]
  ham uninstall [--purge]
  ham down

Daemon socket:
  %s
`, socketPath)
}

var openTarget = func(target core.OpenTarget) error {
	switch target.Kind {
	case core.OpenTargetKindExternalURL, core.OpenTargetKindItermSession, core.OpenTargetKindWorkspace:
		command := exec.Command("open", target.Value)
		return command.Run()
	case core.OpenTargetKindTmuxPane:
		ref, err := adapters.ParseTmuxSessionRef(target.Value)
		if err != nil {
			return err
		}
		if err := exec.Command("tmux", "select-window", "-t", ref.WindowTarget()).Run(); err != nil {
			return err
		}
		return exec.Command("tmux", "select-pane", "-t", ref.PaneTarget()).Run()
	default:
		return fmt.Errorf("unsupported open target kind %q", target.Kind)
	}
}
