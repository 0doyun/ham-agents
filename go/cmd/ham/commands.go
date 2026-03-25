package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/adapters"
	"github.com/ham-agents/ham-agents/go/internal/core"
	"github.com/ham-agents/ham-agents/go/internal/ipc"
	"github.com/ham-agents/ham-agents/go/internal/runtime"
)

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
	if len(args) > 0 {
		switch args[0] {
		case "--pick-iterm-session":
			return runAttachPicker(ctx, client, args[1:], os.Stdin, os.Stdout)
		case "--list-iterm-sessions":
			return runListItermSessions(ctx, client, args[1:])
		}
	}

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

func runListItermSessions(ctx context.Context, client *ipc.Client, args []string) error {
	asJSON := false
	for _, argument := range args {
		switch argument {
		case "--json":
			asJSON = true
		default:
			return fmt.Errorf("unsupported attach listing flag %q", argument)
		}
	}

	sessions, err := client.ListItermSessions(ctx)
	if err != nil {
		return err
	}

	if asJSON {
		return writeJSON(sessions)
	}
	if len(sessions) == 0 {
		fmt.Println("no attachable iTerm sessions")
		return nil
	}

	for index, session := range sessions {
		activeMarker := " "
		if session.IsActive {
			activeMarker = "*"
		}
		fmt.Printf("%s %d\t%s\t%s\n", activeMarker, index+1, session.Title, session.SessionRef)
	}
	return nil
}

func runAttachPicker(ctx context.Context, client *ipc.Client, args []string, in io.Reader, out io.Writer) error {
	options, err := parseAttachPickerOptions(args)
	if err != nil {
		return err
	}

	sessions, err := client.ListItermSessions(ctx)
	if err != nil {
		return err
	}
	if len(sessions) == 0 {
		return fmt.Errorf("no attachable iTerm sessions")
	}

	if options.asJSON {
		return writeJSON(sessions)
	}

	selected, err := chooseAttachableSession(in, out, sessions)
	if err != nil {
		return err
	}

	agent, err := client.AttachSession(ctx, runtime.RegisterAttachedInput{
		Provider:    options.provider,
		DisplayName: selected.Title,
		ProjectPath: options.projectPath,
		Role:        options.role,
		SessionRef:  selected.SessionRef,
	})
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

func runStop(ctx context.Context, client *ipc.Client, args []string) error {
	agentID, asJSON, err := parseStopInput(args)
	if err != nil {
		return err
	}

	if err := client.RemoveAgent(ctx, agentID); err != nil {
		return err
	}

	return renderStopResult(os.Stdout, agentID, asJSON)
}

func runLogs(ctx context.Context, client *ipc.Client, args []string) error {
	agentID, limit, asJSON, err := parseLogsInput(args)
	if err != nil {
		return err
	}

	events, err := client.Events(ctx, agentLogFetchLimit(limit))
	if err != nil {
		return err
	}

	return printEvents(os.Stdout, eventsForAgent(events, agentID, limit), asJSON)
}

func runDoctor(socketPath string, args []string) error {
	flags := flag.NewFlagSet("doctor", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	asJSON := flags.Bool("json", false, "emit JSON")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if len(flags.Args()) > 0 {
		return fmt.Errorf("unexpected argument %q", flags.Args()[0])
	}

	report, err := gatherDoctorReport(socketPath)
	if err != nil {
		return err
	}
	return renderDoctorReport(os.Stdout, report, *asJSON)
}

func runUI(args []string) error {
	target, printOnly, asJSON, err := resolveUICommand(args, os.Executable, os.LookupEnv, os.Getwd, exec.LookPath)
	if err != nil {
		return err
	}

	if asJSON {
		return writeJSONTo(os.Stdout, target)
	}
	if printOnly {
		_, err := fmt.Fprintln(os.Stdout, target.Executable)
		return err
	}

	cmd := exec.Command(target.Executable)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("launch ham ui: %w", err)
	}
	return cmd.Process.Release()
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
	case "appearance":
		return runSettingsAppearance(ctx, client, args[1:])
	case "integrations":
		return runSettingsIntegrations(ctx, client, args[1:])
	default:
		return fmt.Errorf("unsupported settings subcommand %q", args[0])
	}
}

func runSettingsNotifications(ctx context.Context, client *ipc.Client, args []string) error {
	settings, err := client.Settings(ctx)
	if err != nil {
		return err
	}

	if err := applyNotificationSettingsArgs(&settings.Notifications, args); err != nil {
		return err
	}

	updated, err := client.UpdateSettings(ctx, settings)
	if err != nil {
		return err
	}
	return writeJSON(updated)
}

func runSettingsAppearance(ctx context.Context, client *ipc.Client, args []string) error {
	settings, err := client.Settings(ctx)
	if err != nil {
		return err
	}

	for _, argument := range args {
		switch {
		case strings.HasPrefix(argument, "--theme="):
			settings.Appearance.Theme = strings.TrimSpace(strings.TrimPrefix(argument, "--theme="))
		default:
			return fmt.Errorf("unsupported appearance flag %q", argument)
		}
	}

	updated, err := client.UpdateSettings(ctx, settings)
	if err != nil {
		return err
	}
	return writeJSON(updated)
}

func runSettingsIntegrations(ctx context.Context, client *ipc.Client, args []string) error {
	settings, err := client.Settings(ctx)
	if err != nil {
		return err
	}

	for _, argument := range args {
		switch {
		case strings.HasPrefix(argument, "--iterm-enabled="):
			value, err := parseBoolFlag(argument, "--iterm-enabled=")
			if err != nil {
				return err
			}
			settings.Integrations.ItermEnabled = value
		default:
			return fmt.Errorf("unsupported integrations flag %q", argument)
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

	return renderAgents(os.Stdout, agents, *asJSON)
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

	return renderStatus(os.Stdout, snapshot, *asJSON)
}

func runEvents(ctx context.Context, client *ipc.Client, args []string) error {
	flags := flag.NewFlagSet("events", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	asJSON := flags.Bool("json", false, "emit JSON")
	limit := flags.Int("limit", 20, "maximum events to show")
	follow := flags.Bool("follow", false, "follow new events")
	afterID := flags.String("after-id", "", "only show events after this id")
	waitMillis := flags.Int("wait-ms", 15000, "long-poll wait in milliseconds when following")
	if err := flags.Parse(args); err != nil {
		return err
	}

	events, err := client.Events(ctx, *limit)
	if err != nil {
		return err
	}
	currentAfterID := *afterID
	if currentAfterID != "" {
		events = core.EventsAfterID(events, currentAfterID, *limit)
	}

	if err := printEvents(os.Stdout, events, *asJSON); err != nil {
		return err
	}
	if !*follow {
		return nil
	}

	if currentAfterID == "" && len(events) > 0 {
		currentAfterID = events[len(events)-1].ID
	}

	for {
		followed, err := client.FollowEvents(ctx, currentAfterID, *limit, time.Duration(*waitMillis)*time.Millisecond)
		if err != nil {
			return err
		}
		if len(followed) == 0 {
			continue
		}

		if err := printEvents(os.Stdout, followed, *asJSON); err != nil {
			return err
		}
		currentAfterID = followed[len(followed)-1].ID
	}
}

func chooseAttachableSession(in io.Reader, out io.Writer, sessions []core.AttachableSession) (core.AttachableSession, error) {
	if len(sessions) == 0 {
		return core.AttachableSession{}, fmt.Errorf("no attachable sessions")
	}
	if len(sessions) == 1 {
		return sessions[0], nil
	}

	for index, session := range sessions {
		activeMarker := " "
		if session.IsActive {
			activeMarker = "*"
		}
		if _, err := fmt.Fprintf(out, "%s %d) %s [%s]\n", activeMarker, index+1, session.Title, session.ID); err != nil {
			return core.AttachableSession{}, err
		}
	}
	if _, err := fmt.Fprint(out, "Select iTerm session: "); err != nil {
		return core.AttachableSession{}, err
	}

	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && err != io.EOF {
		return core.AttachableSession{}, err
	}

	selection, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil {
		return core.AttachableSession{}, fmt.Errorf("invalid session selection %q", strings.TrimSpace(line))
	}
	if selection < 1 || selection > len(sessions) {
		return core.AttachableSession{}, fmt.Errorf("session selection must be between 1 and %d", len(sessions))
	}

	return sessions[selection-1], nil
}
