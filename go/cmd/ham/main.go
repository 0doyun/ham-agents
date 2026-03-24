package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/adapters"
	"github.com/ham-agents/ham-agents/go/internal/core"
	"github.com/ham-agents/ham-agents/go/internal/ipc"
	"github.com/ham-agents/ham-agents/go/internal/runtime"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "ham: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	ctx := context.Background()
	client, socketPath, err := newClient()
	if err != nil {
		return err
	}

	if len(args) == 0 {
		printHelp(socketPath)
		return nil
	}

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
	case "settings":
		return runSettings(ctx, client, args[1:])
	case "list":
		return runList(ctx, client, args[1:])
	case "status":
		return runStatus(ctx, client, args[1:])
	case "events":
		return runEvents(ctx, client, args[1:])
	default:
		return fmt.Errorf("unsupported command %q", args[0])
	}
}

func newClient() (*ipc.Client, string, error) {
	socketPath, err := ipc.DefaultSocketPath()
	if err != nil {
		return nil, "", err
	}

	return ipc.NewClient(socketPath), socketPath, nil
}

func printHelp(socketPath string) {
	fmt.Printf(`ham-agents Go CLI bootstrap

Usage:
  ham help
  ham run <provider> [name] [--project path] [--role role]
  ham attach <session-ref> [name] [--project path] [--role role] [--provider provider]
  ham observe <source-ref> [name] [--project path] [--role role] [--provider provider]
  ham open <agent-id> [--json] [--print]
  ham ask <agent-id> <message>
  ham settings [--json]
  ham settings notifications [--done=true|false] [--error=true|false] [--waiting-input=true|false] [--preview-text=true|false]
  ham list [--json]
  ham status [--json]
  ham events [--json] [--limit N]

Daemon socket:
  %s
`, socketPath)
}

func runRegister(ctx context.Context, client *ipc.Client, args []string) error {
	input, err := parseRunInput(args)
	if err != nil {
		return err
	}

	agent, err := client.RunManaged(ctx, input)
	if err != nil {
		return err
	}

	fmt.Printf("registered %s [%s] via %s\n", agent.DisplayName, agent.ID, agent.Provider)
	return nil
}

func runAttach(ctx context.Context, client *ipc.Client, args []string) error {
	input, err := parseAttachInput(args)
	if err != nil {
		return err
	}

	agent, err := client.AttachSession(ctx, input)
	if err != nil {
		return err
	}

	fmt.Printf("attached %s [%s] via %s\n", agent.DisplayName, agent.ID, agent.Provider)
	return nil
}

func runObserve(ctx context.Context, client *ipc.Client, args []string) error {
	input, err := parseObserveInput(args)
	if err != nil {
		return err
	}

	agent, err := client.ObserveSource(ctx, input)
	if err != nil {
		return err
	}

	fmt.Printf("observing %s [%s] via %s\n", agent.DisplayName, agent.ID, agent.Provider)
	return nil
}

func runOpen(ctx context.Context, client *ipc.Client, args []string) error {
	agentID, asJSON, printOnly, err := parseOpenInput(args)
	if err != nil {
		return err
	}

	target, err := client.OpenTarget(ctx, agentID)
	if err != nil {
		return err
	}

	if asJSON {
		return writeJSON(target)
	}
	if printOnly {
		fmt.Printf("%s\t%s\n", target.Kind, target.Value)
		return nil
	}

	return openTarget(target)
}

func runAsk(ctx context.Context, client *ipc.Client, args []string) error {
	agentID, message, err := parseAskInput(args)
	if err != nil {
		return err
	}

	target, err := client.OpenTarget(ctx, agentID)
	if err != nil {
		return err
	}

	result, err := adapters.NewQuickMessageSender(nil).Send(target, message)
	if err != nil {
		return err
	}

	fmt.Println(result)
	return nil
}

func runSettings(ctx context.Context, client *ipc.Client, args []string) error {
	if len(args) == 0 {
		settings, err := client.Settings(ctx)
		if err != nil {
			return err
		}
		return writeJSON(settings)
	}

	switch args[0] {
	case "--json":
		settings, err := client.Settings(ctx)
		if err != nil {
			return err
		}
		return writeJSON(settings)
	case "notifications":
		return runSettingsNotifications(ctx, client, args[1:])
	default:
		return fmt.Errorf("unsupported settings subcommand %q", args[0])
	}
}

func runSettingsNotifications(ctx context.Context, client *ipc.Client, args []string) error {
	settings, err := client.Settings(ctx)
	if err != nil {
		return err
	}

	for _, argument := range args {
		switch {
		case strings.HasPrefix(argument, "--done="):
			value, err := parseBoolFlag(argument, "--done=")
			if err != nil {
				return err
			}
			settings.Notifications.Done = value
		case strings.HasPrefix(argument, "--error="):
			value, err := parseBoolFlag(argument, "--error=")
			if err != nil {
				return err
			}
			settings.Notifications.Error = value
		case strings.HasPrefix(argument, "--waiting-input="):
			value, err := parseBoolFlag(argument, "--waiting-input=")
			if err != nil {
				return err
			}
			settings.Notifications.WaitingInput = value
		case strings.HasPrefix(argument, "--preview-text="):
			value, err := parseBoolFlag(argument, "--preview-text=")
			if err != nil {
				return err
			}
			settings.Notifications.PreviewText = value
		default:
			return fmt.Errorf("unsupported notifications flag %q", argument)
		}
	}

	updated, err := client.UpdateSettings(ctx, settings)
	if err != nil {
		return err
	}
	return writeJSON(updated)
}

func runList(ctx context.Context, client *ipc.Client, args []string) error {
	flags := flag.NewFlagSet("list", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	asJSON := flags.Bool("json", false, "emit JSON")
	if err := flags.Parse(args); err != nil {
		return err
	}

	agents, err := client.ListAgents(ctx)
	if err != nil {
		return err
	}

	if *asJSON {
		return writeJSON(agents)
	}

	if len(agents) == 0 {
		fmt.Println("no tracked agents")
		return nil
	}

	for _, agent := range agents {
		fmt.Printf("%s\t%s\t%s\t%s\t%s\n", agent.ID, agent.DisplayName, agent.Provider, agent.Status, agent.Mode)
	}
	return nil
}

func runStatus(ctx context.Context, client *ipc.Client, args []string) error {
	flags := flag.NewFlagSet("status", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	asJSON := flags.Bool("json", false, "emit JSON")
	if err := flags.Parse(args); err != nil {
		return err
	}

	snapshot, err := client.Status(ctx)
	if err != nil {
		return err
	}

	if *asJSON {
		return writeJSON(map[string]any{
			"total":       snapshot.TotalCount(),
			"running":     snapshot.RunningCount(),
			"waiting":     snapshot.WaitingCount(),
			"done":        snapshot.DoneCount(),
			"generatedAt": snapshot.GeneratedAt,
		})
	}

	fmt.Printf("total=%d running=%d waiting=%d done=%d\n", snapshot.TotalCount(), snapshot.RunningCount(), snapshot.WaitingCount(), snapshot.DoneCount())
	return nil
}

func runEvents(ctx context.Context, client *ipc.Client, args []string) error {
	flags := flag.NewFlagSet("events", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	asJSON := flags.Bool("json", false, "emit JSON")
	limit := flags.Int("limit", 20, "maximum events to show")
	if err := flags.Parse(args); err != nil {
		return err
	}

	events, err := client.Events(ctx, *limit)
	if err != nil {
		return err
	}
	if *asJSON {
		return writeJSON(events)
	}
	if len(events) == 0 {
		fmt.Println("no events")
		return nil
	}
	for _, event := range events {
		fmt.Printf("%s\t%s\t%s\t%s\n", event.OccurredAt.Format(time.RFC3339), event.Type, event.AgentID, event.Summary)
	}
	return nil
}

func parseRunInput(args []string) (runtime.RegisterManagedInput, error) {
	provider, remainder := splitProvider(args)
	input := runtime.RegisterManagedInput{Provider: provider}

	for index := 0; index < len(remainder); index++ {
		argument := remainder[index]

		switch {
		case argument == "--project":
			index++
			if index >= len(remainder) {
				return runtime.RegisterManagedInput{}, fmt.Errorf("missing value for --project")
			}
			input.ProjectPath = remainder[index]
		case strings.HasPrefix(argument, "--project="):
			input.ProjectPath = strings.TrimPrefix(argument, "--project=")
		case argument == "--role":
			index++
			if index >= len(remainder) {
				return runtime.RegisterManagedInput{}, fmt.Errorf("missing value for --role")
			}
			input.Role = remainder[index]
		case strings.HasPrefix(argument, "--role="):
			input.Role = strings.TrimPrefix(argument, "--role=")
		case strings.HasPrefix(argument, "-"):
			return runtime.RegisterManagedInput{}, fmt.Errorf("unsupported flag %q", argument)
		case input.DisplayName == "":
			input.DisplayName = argument
		default:
			return runtime.RegisterManagedInput{}, fmt.Errorf("unexpected argument %q", argument)
		}
	}

	return input, nil
}

func parseAttachInput(args []string) (runtime.RegisterAttachedInput, error) {
	if len(args) == 0 {
		return runtime.RegisterAttachedInput{}, fmt.Errorf("session ref is required")
	}

	input := runtime.RegisterAttachedInput{
		Provider:   "iterm2",
		SessionRef: args[0],
	}
	remainder := args[1:]

	for index := 0; index < len(remainder); index++ {
		argument := remainder[index]

		switch {
		case argument == "--project":
			index++
			if index >= len(remainder) {
				return runtime.RegisterAttachedInput{}, fmt.Errorf("missing value for --project")
			}
			input.ProjectPath = remainder[index]
		case strings.HasPrefix(argument, "--project="):
			input.ProjectPath = strings.TrimPrefix(argument, "--project=")
		case argument == "--role":
			index++
			if index >= len(remainder) {
				return runtime.RegisterAttachedInput{}, fmt.Errorf("missing value for --role")
			}
			input.Role = remainder[index]
		case strings.HasPrefix(argument, "--role="):
			input.Role = strings.TrimPrefix(argument, "--role=")
		case argument == "--provider":
			index++
			if index >= len(remainder) {
				return runtime.RegisterAttachedInput{}, fmt.Errorf("missing value for --provider")
			}
			input.Provider = remainder[index]
		case strings.HasPrefix(argument, "--provider="):
			input.Provider = strings.TrimPrefix(argument, "--provider=")
		case strings.HasPrefix(argument, "-"):
			return runtime.RegisterAttachedInput{}, fmt.Errorf("unsupported flag %q", argument)
		case input.DisplayName == "":
			input.DisplayName = argument
		default:
			return runtime.RegisterAttachedInput{}, fmt.Errorf("unexpected argument %q", argument)
		}
	}

	return input, nil
}

func parseObserveInput(args []string) (runtime.RegisterObservedInput, error) {
	if len(args) == 0 {
		return runtime.RegisterObservedInput{}, fmt.Errorf("source ref is required")
	}

	input := runtime.RegisterObservedInput{
		Provider:   "log",
		SessionRef: args[0],
	}
	remainder := args[1:]

	for index := 0; index < len(remainder); index++ {
		argument := remainder[index]

		switch {
		case argument == "--project":
			index++
			if index >= len(remainder) {
				return runtime.RegisterObservedInput{}, fmt.Errorf("missing value for --project")
			}
			input.ProjectPath = remainder[index]
		case strings.HasPrefix(argument, "--project="):
			input.ProjectPath = strings.TrimPrefix(argument, "--project=")
		case argument == "--role":
			index++
			if index >= len(remainder) {
				return runtime.RegisterObservedInput{}, fmt.Errorf("missing value for --role")
			}
			input.Role = remainder[index]
		case strings.HasPrefix(argument, "--role="):
			input.Role = strings.TrimPrefix(argument, "--role=")
		case argument == "--provider":
			index++
			if index >= len(remainder) {
				return runtime.RegisterObservedInput{}, fmt.Errorf("missing value for --provider")
			}
			input.Provider = remainder[index]
		case strings.HasPrefix(argument, "--provider="):
			input.Provider = strings.TrimPrefix(argument, "--provider=")
		case strings.HasPrefix(argument, "-"):
			return runtime.RegisterObservedInput{}, fmt.Errorf("unsupported flag %q", argument)
		case input.DisplayName == "":
			input.DisplayName = argument
		default:
			return runtime.RegisterObservedInput{}, fmt.Errorf("unexpected argument %q", argument)
		}
	}

	return input, nil
}

func parseOpenInput(args []string) (agentID string, asJSON bool, printOnly bool, err error) {
	for _, argument := range args {
		switch argument {
		case "--json":
			asJSON = true
		case "--print":
			printOnly = true
		default:
			if strings.HasPrefix(argument, "-") {
				err = fmt.Errorf("unsupported flag %q", argument)
				return
			}
			if agentID == "" {
				agentID = argument
				continue
			}
			err = fmt.Errorf("unexpected argument %q", argument)
			return
		}
	}

	if agentID == "" {
		err = fmt.Errorf("agent id is required")
	}
	return
}

func parseAskInput(args []string) (agentID string, message string, err error) {
	if len(args) < 2 {
		return "", "", fmt.Errorf("agent id and message are required")
	}
	agentID = args[0]
	message = strings.Join(args[1:], " ")
	if strings.TrimSpace(message) == "" {
		err = fmt.Errorf("message is required")
	}
	return
}

func parseBoolFlag(argument string, prefix string) (bool, error) {
	value := strings.TrimPrefix(argument, prefix)
	switch value {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value %q", value)
	}
}

func splitProvider(args []string) (string, []string) {
	if len(args) == 0 {
		return "unknown", args
	}
	if strings.HasPrefix(args[0], "-") {
		return "unknown", args
	}
	return args[0], args[1:]
}

func writeJSON(value any) error {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(os.Stdout, "%s\n", payload)
	return err
}

var openTarget = func(target core.OpenTarget) error {
	switch target.Kind {
	case core.OpenTargetKindExternalURL, core.OpenTargetKindWorkspace:
		command := exec.Command("open", target.Value)
		return command.Run()
	default:
		return fmt.Errorf("unsupported open target kind %q", target.Kind)
	}
}
