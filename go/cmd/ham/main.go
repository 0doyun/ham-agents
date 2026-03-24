package main

import (
	"bufio"
	"context"
	"encoding/json"
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
  ham attach --pick-iterm-session [--json] [--project path] [--role role] [--provider provider]
  ham attach --list-iterm-sessions [--json]
  ham observe <source-ref> [name] [--project path] [--role role] [--provider provider]
  ham open <agent-id> [--json] [--print]
  ham ask <agent-id> <message>
  ham settings [--json]
  ham settings notifications [--done=true|false] [--error=true|false] [--waiting-input=true|false] [--quiet-hours=true|false] [--quiet-start-hour=0-23] [--quiet-end-hour=0-23] [--preview-text=true|false]
  ham settings appearance [--theme=auto|day|night]
  ham settings integrations [--iterm-enabled=true|false]
  ham list [--json]
  ham status [--json]
  ham events [--json] [--limit N] [--follow] [--after-id ID] [--wait-ms N]

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
		case strings.HasPrefix(argument, "--quiet-hours="):
			value, err := parseBoolFlag(argument, "--quiet-hours=")
			if err != nil {
				return err
			}
			settings.Notifications.QuietHoursEnabled = value
		case strings.HasPrefix(argument, "--quiet-start-hour="):
			value, err := parseHourFlag(argument, "--quiet-start-hour=")
			if err != nil {
				return err
			}
			settings.Notifications.QuietHoursStartHour = value
		case strings.HasPrefix(argument, "--quiet-end-hour="):
			value, err := parseHourFlag(argument, "--quiet-end-hour=")
			if err != nil {
				return err
			}
			settings.Notifications.QuietHoursEndHour = value
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
		events = eventsAfterIDForDisplay(events, currentAfterID, *limit)
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

type attachPickerOptions struct {
	projectPath string
	role        string
	provider    string
	asJSON      bool
}

func parseAttachPickerOptions(args []string) (attachPickerOptions, error) {
	options := attachPickerOptions{provider: "iterm2"}

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

func writeJSON(value any) error {
	return writeJSONTo(os.Stdout, value)
}

func writeJSONTo(out io.Writer, value any) error {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "%s\n", payload)
	return err
}

func renderAgents(out io.Writer, agents []core.Agent, asJSON bool) error {
	if asJSON {
		return writeJSONTo(out, agents)
	}

	if len(agents) == 0 {
		_, err := fmt.Fprintln(out, "no tracked agents")
		return err
	}

	for _, agent := range agents {
		if _, err := fmt.Fprintln(out, formatAgentListLine(agent)); err != nil {
			return err
		}
	}
	return nil
}

func formatAgentListLine(agent core.Agent) string {
	parts := []string{
		agent.ID,
		agent.DisplayName,
		agent.Provider,
		string(agent.Mode),
		formatAgentStatusLabel(agent),
		formatConfidenceLabel(agent.StatusConfidence),
	}
	if reason := strings.TrimSpace(agent.StatusReason); reason != "" {
		parts = append(parts, reason)
	}
	return strings.Join(parts, "\t")
}

func formatAgentStatusLabel(agent core.Agent) string {
	if agent.StatusConfidence < 0.5 {
		return "likely " + string(agent.Status)
	}
	return string(agent.Status)
}

func formatConfidenceLabel(confidence float64) string {
	percent := int((confidence * 100) + 0.5)
	level := "low"
	switch {
	case confidence >= 0.8:
		level = "high"
	case confidence >= 0.5:
		level = "medium"
	}
	return fmt.Sprintf("%s %d%%", level, percent)
}

func countAttentionAgents(agents []core.Agent) int {
	count := 0
	for _, agent := range agents {
		switch agent.Status {
		case core.AgentStatusError, core.AgentStatusWaitingInput, core.AgentStatusDisconnected:
			count++
		}
	}
	return count
}

func renderStatus(out io.Writer, snapshot core.RuntimeSnapshot, asJSON bool) error {
	if asJSON {
		return writeJSONTo(out, map[string]any{
			"total":       snapshot.TotalCount(),
			"running":     snapshot.RunningCount(),
			"waiting":     snapshot.WaitingCount(),
			"done":        snapshot.DoneCount(),
			"generatedAt": snapshot.GeneratedAt,
		})
	}

	attentionCount := countAttentionAgents(snapshot.Agents)
	_, err := fmt.Fprintf(
		out,
		"total=%d running=%d waiting=%d done=%d attention=%d\n",
		snapshot.TotalCount(),
		snapshot.RunningCount(),
		snapshot.WaitingCount(),
		snapshot.DoneCount(),
		attentionCount,
	)
	return err
}

func printEvents(out io.Writer, events []core.Event, asJSON bool) error {
	if asJSON {
		if len(events) == 0 {
			return writeJSON([]core.Event{})
		}
		for _, event := range events {
			payload, err := json.Marshal(event)
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintf(out, "%s\n", payload); err != nil {
				return err
			}
		}
		return nil
	}

	if len(events) == 0 {
		_, err := fmt.Fprintln(out, "no events")
		return err
	}
	for _, event := range events {
		if _, err := fmt.Fprintf(out, "%s\t%s\t%s\t%s\n", event.OccurredAt.Format(time.RFC3339), event.Type, event.AgentID, event.Summary); err != nil {
			return err
		}
	}
	return nil
}

func eventsAfterIDForDisplay(events []core.Event, afterEventID string, limit int) []core.Event {
	if afterEventID == "" {
		return events
	}

	start := -1
	for index, event := range events {
		if event.ID == afterEventID {
			start = index + 1
			break
		}
	}
	if start == -1 {
		start = 0
	}
	if start >= len(events) {
		return []core.Event{}
	}
	filtered := events[start:]
	if limit > 0 && len(filtered) > limit {
		return filtered[len(filtered)-limit:]
	}
	return filtered
}

var openTarget = func(target core.OpenTarget) error {
	switch target.Kind {
	case core.OpenTargetKindExternalURL, core.OpenTargetKindItermSession, core.OpenTargetKindWorkspace:
		command := exec.Command("open", target.Value)
		return command.Run()
	default:
		return fmt.Errorf("unsupported open target kind %q", target.Kind)
	}
}
