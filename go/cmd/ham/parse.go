package main

import (
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/ham-agents/ham-agents/go/internal/core"
	"github.com/ham-agents/ham-agents/go/internal/runtime"
)

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
		Provider:   providerForSessionRef(args[0]),
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

type attachPickerOptions struct {
	projectPath string
	role        string
	provider    string
	asJSON      bool
}

func parseAttachPickerOptions(args []string) (attachPickerOptions, error) {
	options := attachPickerOptions{}

	for index := 0; index < len(args); index++ {
		argument := args[index]

		switch {
		case argument == "--json":
			options.asJSON = true
		case argument == "--project":
			index++
			if index >= len(args) {
				return attachPickerOptions{}, fmt.Errorf("missing value for --project")
			}
			options.projectPath = args[index]
		case strings.HasPrefix(argument, "--project="):
			options.projectPath = strings.TrimPrefix(argument, "--project=")
		case argument == "--role":
			index++
			if index >= len(args) {
				return attachPickerOptions{}, fmt.Errorf("missing value for --role")
			}
			options.role = args[index]
		case strings.HasPrefix(argument, "--role="):
			options.role = strings.TrimPrefix(argument, "--role=")
		case argument == "--provider":
			index++
			if index >= len(args) {
				return attachPickerOptions{}, fmt.Errorf("missing value for --provider")
			}
			options.provider = args[index]
		case strings.HasPrefix(argument, "--provider="):
			options.provider = strings.TrimPrefix(argument, "--provider=")
		default:
			return attachPickerOptions{}, fmt.Errorf("unsupported attach picker flag %q", argument)
		}
	}

	return options, nil
}

func providerForSessionRef(sessionRef string) string {
	trimmed := strings.TrimSpace(sessionRef)
	switch {
	case strings.HasPrefix(trimmed, "tmux://"):
		return "tmux"
	case strings.HasPrefix(trimmed, "iterm2://"):
		return "iterm2"
	default:
		return "iterm2"
	}
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

func parseStopInput(args []string) (agentID string, asJSON bool, err error) {
	for _, argument := range args {
		switch argument {
		case "--json":
			asJSON = true
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

func parseLogsInput(args []string) (agentID string, limit int, asJSON bool, exportPath string, err error) {
	flags := flag.NewFlagSet("logs", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	asJSONFlag := flags.Bool("json", false, "emit JSON")
	limitFlag := flags.Int("limit", 20, "maximum events to show")
	exportFlag := flags.String("export", "", "write output to this path")
	if err = flags.Parse(args); err != nil {
		return
	}

	remaining := flags.Args()
	if len(remaining) != 1 {
		err = fmt.Errorf("agent id is required")
		return
	}
	if *limitFlag < 1 {
		err = fmt.Errorf("limit must be at least 1")
		return
	}

	agentID = remaining[0]
	limit = *limitFlag
	asJSON = *asJSONFlag
	exportPath = strings.TrimSpace(*exportFlag)
	return
}

func applyNotificationSettingsArgs(settings *core.NotificationSettings, args []string) error {
	for _, argument := range args {
		switch {
		case strings.HasPrefix(argument, "--done="):
			value, err := parseBoolFlag(argument, "--done=")
			if err != nil {
				return err
			}
			settings.Done = value
		case strings.HasPrefix(argument, "--error="):
			value, err := parseBoolFlag(argument, "--error=")
			if err != nil {
				return err
			}
			settings.Error = value
		case strings.HasPrefix(argument, "--waiting-input="):
			value, err := parseBoolFlag(argument, "--waiting-input=")
			if err != nil {
				return err
			}
			settings.WaitingInput = value
		case strings.HasPrefix(argument, "--silence="):
			value, err := parseBoolFlag(argument, "--silence=")
			if err != nil {
				return err
			}
			settings.Silence = value
		case strings.HasPrefix(argument, "--quiet-hours="):
			value, err := parseBoolFlag(argument, "--quiet-hours=")
			if err != nil {
				return err
			}
			settings.QuietHoursEnabled = value
		case strings.HasPrefix(argument, "--quiet-start-hour="):
			value, err := parseHourFlag(argument, "--quiet-start-hour=")
			if err != nil {
				return err
			}
			settings.QuietHoursStartHour = value
		case strings.HasPrefix(argument, "--quiet-end-hour="):
			value, err := parseHourFlag(argument, "--quiet-end-hour=")
			if err != nil {
				return err
			}
			settings.QuietHoursEndHour = value
		case strings.HasPrefix(argument, "--preview-text="):
			value, err := parseBoolFlag(argument, "--preview-text=")
			if err != nil {
				return err
			}
			settings.PreviewText = value
		case strings.HasPrefix(argument, "--heartbeat-minutes="):
			value, err := strconv.Atoi(strings.TrimPrefix(argument, "--heartbeat-minutes="))
			if err != nil {
				return fmt.Errorf("invalid heartbeat minutes %q", strings.TrimPrefix(argument, "--heartbeat-minutes="))
			}
			settings.HeartbeatMinutes = value
		default:
			return fmt.Errorf("unsupported notifications flag %q", argument)
		}
	}
	return nil
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

func parseHourFlag(argument string, prefix string) (int, error) {
	value := strings.TrimPrefix(argument, prefix)
	hour, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid hour value %q", value)
	}
	if hour < core.MinQuietHour || hour > core.MaxQuietHour {
		return 0, fmt.Errorf("hour value %d must be between %d and %d", hour, core.MinQuietHour, core.MaxQuietHour)
	}
	return hour, nil
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

type agentQueryOptions struct {
	asJSON       bool
	teamRef      string
	workspaceRef string
	graph        bool
}

func parseAgentQueryOptions(command string, args []string) (agentQueryOptions, error) {
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	asJSON := flags.Bool("json", false, "emit JSON")
	teamRef := flags.String("team", "", "filter by team id or name")
	workspaceRef := flags.String("workspace", "", "filter by workspace path or name")
	graph := flags.Bool("graph", false, "render session tree")
	if err := flags.Parse(args); err != nil {
		return agentQueryOptions{}, err
	}
	if len(flags.Args()) > 0 {
		return agentQueryOptions{}, fmt.Errorf("unexpected argument %q", flags.Args()[0])
	}
	return agentQueryOptions{
		asJSON:       *asJSON,
		teamRef:      strings.TrimSpace(*teamRef),
		workspaceRef: strings.TrimSpace(*workspaceRef),
		graph:        *graph,
	}, nil
}
