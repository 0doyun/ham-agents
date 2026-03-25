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

	if team, teamErr := resolveTeam(ctx, client, agentID); teamErr == nil {
		for _, memberAgentID := range team.MemberAgentIDs {
			target, err := client.OpenTarget(ctx, memberAgentID)
			if err != nil {
				return err
			}

			if _, err := adapters.NewQuickMessageSender(nil).Send(target, message); err != nil {
				return err
			}
		}
		fmt.Printf("sent message to team %s (%d agents)\n", team.DisplayName, len(team.MemberAgentIDs))
		return nil
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

func runTeam(ctx context.Context, client *ipc.Client, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("team subcommand is required")
	}

	switch args[0] {
	case "create":
		if len(args) < 2 {
			return fmt.Errorf("team name is required")
		}
		asJSON := len(args) > 2 && args[2] == "--json"
		team, err := client.CreateTeam(ctx, args[1])
		if err != nil {
			return err
		}
		if asJSON {
			return writeJSON(team)
		}
		fmt.Printf("created team %s [%s]\n", team.DisplayName, team.ID)
		return nil
	case "add":
		if len(args) < 3 {
			return fmt.Errorf("team and agent id are required")
		}
		asJSON := len(args) > 3 && args[3] == "--json"
		team, err := client.AddTeamMember(ctx, args[1], args[2])
		if err != nil {
			return err
		}
		if asJSON {
			return writeJSON(team)
		}
		fmt.Printf("added %s to team %s\n", args[2], team.DisplayName)
		return nil
	case "list":
		teams, err := client.ListTeams(ctx)
		if err != nil {
			return err
		}
		if len(args) > 1 && args[1] == "--json" {
			return writeJSON(teams)
		}
		if len(teams) == 0 {
			fmt.Println("no teams")
			return nil
		}
		for _, team := range teams {
			fmt.Printf("%s\t%s\t%d members\n", team.ID, team.DisplayName, len(team.MemberAgentIDs))
		}
		return nil
	case "open":
		if len(args) < 2 {
			return fmt.Errorf("team is required")
		}
		team, err := resolveTeam(ctx, client, args[1])
		if err != nil {
			return err
		}
		for _, memberAgentID := range team.MemberAgentIDs {
			target, err := client.OpenTarget(ctx, memberAgentID)
			if err != nil {
				return err
			}
			if err := openTarget(target); err != nil {
				return err
			}
		}
		fmt.Printf("opened %d agents for team %s\n", len(team.MemberAgentIDs), team.DisplayName)
		return nil
	default:
		return fmt.Errorf("unsupported team subcommand %q", args[0])
	}
}

func runStop(ctx context.Context, client *ipc.Client, args []string) error {
	agentID, asJSON, err := parseStopInput(args)
	if err != nil {
		return err
	}

	if err := client.StopManaged(ctx, agentID); err != nil {
		if removeErr := client.RemoveAgent(ctx, agentID); removeErr != nil {
			return err
		}
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
	case "general":
		return runSettingsGeneral(ctx, client, args[1:])
	case "notifications":
		return runSettingsNotifications(ctx, client, args[1:])
	case "appearance":
		return runSettingsAppearance(ctx, client, args[1:])
	case "integrations":
		return runSettingsIntegrations(ctx, client, args[1:])
	case "privacy":
		return runSettingsPrivacy(ctx, client, args[1:])
	default:
		return fmt.Errorf("unsupported settings subcommand %q", args[0])
	}
}

func runSettingsGeneral(ctx context.Context, client *ipc.Client, args []string) error {
	settings, err := client.Settings(ctx)
	if err != nil {
		return err
	}

	for _, argument := range args {
		switch {
		case strings.HasPrefix(argument, "--launch-at-login="):
			value, err := parseBoolFlag(argument, "--launch-at-login=")
			if err != nil {
				return err
			}
			settings.General.LaunchAtLogin = value
		case strings.HasPrefix(argument, "--compact-mode="):
			value, err := parseBoolFlag(argument, "--compact-mode=")
			if err != nil {
				return err
			}
			settings.General.CompactMode = value
		case strings.HasPrefix(argument, "--show-menu-bar-animation-always="):
			value, err := parseBoolFlag(argument, "--show-menu-bar-animation-always=")
			if err != nil {
				return err
			}
			settings.General.ShowMenuBarAnimationAlways = value
		default:
			return fmt.Errorf("unsupported general flag %q", argument)
		}
	}

	updated, err := client.UpdateSettings(ctx, settings)
	if err != nil {
		return err
	}
	return writeJSON(updated)
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
		case strings.HasPrefix(argument, "--animation-speed="):
			value, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimPrefix(argument, "--animation-speed=")), 64)
			if err != nil {
				return fmt.Errorf("invalid animation speed %q", strings.TrimPrefix(argument, "--animation-speed="))
			}
			settings.Appearance.AnimationSpeedMultiplier = value
		case strings.HasPrefix(argument, "--reduce-motion="):
			value, err := parseBoolFlag(argument, "--reduce-motion=")
			if err != nil {
				return err
			}
			settings.Appearance.ReduceMotion = value
		case strings.HasPrefix(argument, "--hamster-skin="):
			settings.Appearance.HamsterSkin = strings.TrimSpace(strings.TrimPrefix(argument, "--hamster-skin="))
		case strings.HasPrefix(argument, "--hat="):
			settings.Appearance.Hat = strings.TrimSpace(strings.TrimPrefix(argument, "--hat="))
		case strings.HasPrefix(argument, "--desk-theme="):
			settings.Appearance.DeskTheme = strings.TrimSpace(strings.TrimPrefix(argument, "--desk-theme="))
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
		case strings.HasPrefix(argument, "--transcript-dirs="):
			value := strings.TrimSpace(strings.TrimPrefix(argument, "--transcript-dirs="))
			if value == "" {
				settings.Integrations.TranscriptDirs = []string{}
				continue
			}
			parts := strings.Split(value, ",")
			dirs := make([]string, 0, len(parts))
			for _, part := range parts {
				trimmed := strings.TrimSpace(part)
				if trimmed != "" {
					dirs = append(dirs, trimmed)
				}
			}
			settings.Integrations.TranscriptDirs = dirs
		case strings.HasPrefix(argument, "--provider-adapter="):
			value := strings.TrimSpace(strings.TrimPrefix(argument, "--provider-adapter="))
			segments := strings.SplitN(value, "=", 2)
			if len(segments) != 2 {
				return fmt.Errorf("provider adapter flag must be name=true|false")
			}
			enabled, err := strconv.ParseBool(strings.TrimSpace(segments[1]))
			if err != nil {
				return fmt.Errorf("invalid provider adapter value %q", segments[1])
			}
			if settings.Integrations.ProviderAdapters == nil {
				settings.Integrations.ProviderAdapters = map[string]bool{}
			}
			settings.Integrations.ProviderAdapters[strings.TrimSpace(segments[0])] = enabled
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

func runSettingsPrivacy(ctx context.Context, client *ipc.Client, args []string) error {
	settings, err := client.Settings(ctx)
	if err != nil {
		return err
	}

	for _, argument := range args {
		switch {
		case strings.HasPrefix(argument, "--local-only-mode="):
			value, err := parseBoolFlag(argument, "--local-only-mode=")
			if err != nil {
				return err
			}
			settings.Privacy.LocalOnlyMode = value
		case strings.HasPrefix(argument, "--event-history-retention-days="):
			value := strings.TrimSpace(strings.TrimPrefix(argument, "--event-history-retention-days="))
			days, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("invalid retention days %q", value)
			}
			settings.Privacy.EventHistoryRetentionDays = days
		case strings.HasPrefix(argument, "--transcript-excerpt-storage="):
			value, err := parseBoolFlag(argument, "--transcript-excerpt-storage=")
			if err != nil {
				return err
			}
			settings.Privacy.TranscriptExcerptStorage = value
		default:
			return fmt.Errorf("unsupported privacy flag %q", argument)
		}
	}

	updated, err := client.UpdateSettings(ctx, settings)
	if err != nil {
		return err
	}
	return writeJSON(updated)
}

func runList(ctx context.Context, client *ipc.Client, args []string) error {
	options, err := parseAgentQueryOptions("list", args)
	if err != nil {
		return err
	}

	agents, err := client.ListAgents(ctx)
	if err != nil {
		return err
	}

	filtered, err := filterAgentsForQuery(ctx, client, agents, options.teamRef, options.workspaceRef)
	if err != nil {
		return err
	}

	return renderAgents(os.Stdout, filtered, options.asJSON)
}

func runStatus(ctx context.Context, client *ipc.Client, args []string) error {
	options, err := parseAgentQueryOptions("status", args)
	if err != nil {
		return err
	}

	snapshot, err := client.Status(ctx)
	if err != nil {
		return err
	}

	filteredAgents, err := filterAgentsForQuery(ctx, client, snapshot.Agents, options.teamRef, options.workspaceRef)
	if err != nil {
		return err
	}

	if options.teamRef != "" || options.workspaceRef != "" {
		snapshot = buildFilteredSnapshot(filteredAgents, snapshot.GeneratedAt)
	}

	return renderStatus(os.Stdout, snapshot, options.asJSON)
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

func resolveTeam(ctx context.Context, client *ipc.Client, ref string) (core.Team, error) {
	teams, err := client.ListTeams(ctx)
	if err != nil {
		return core.Team{}, err
	}
	for _, team := range teams {
		if team.Matches(ref) {
			return team, nil
		}
	}
	return core.Team{}, fmt.Errorf("team %q not found", ref)
}

func filterAgentsForQuery(ctx context.Context, client *ipc.Client, agents []core.Agent, teamRef string, workspaceRef string) ([]core.Agent, error) {
	filtered := append([]core.Agent(nil), agents...)

	if teamRef != "" {
		team, err := resolveTeam(ctx, client, teamRef)
		if err != nil {
			return nil, err
		}
		filtered = filterAgentsForTeam(filtered, team)
	}

	if workspaceRef != "" {
		teams, err := client.ListTeams(ctx)
		if err != nil {
			return nil, err
		}
		workspace, ok := resolveWorkspace(filtered, teams, workspaceRef)
		if !ok {
			return nil, fmt.Errorf("workspace %q not found", workspaceRef)
		}
		filtered = filterAgentsForWorkspace(filtered, workspace)
	}

	return filtered, nil
}

func filterAgents(agents []core.Agent, keep func(core.Agent) bool) []core.Agent {
	filtered := make([]core.Agent, 0, len(agents))
	for _, agent := range agents {
		if keep(agent) {
			filtered = append(filtered, agent)
		}
	}
	return filtered
}

func filterAgentsForTeam(agents []core.Agent, team core.Team) []core.Agent {
	memberSet := make(map[string]struct{}, len(team.MemberAgentIDs))
	for _, agentID := range team.MemberAgentIDs {
		memberSet[agentID] = struct{}{}
	}
	return filterAgents(agents, func(agent core.Agent) bool {
		_, ok := memberSet[agent.ID]
		return ok
	})
}

func resolveWorkspace(agents []core.Agent, teams []core.Team, ref string) (core.Workspace, bool) {
	for _, workspace := range core.BuildWorkspaces(agents, teams) {
		if workspace.Matches(ref) {
			return workspace, true
		}
	}
	return core.Workspace{}, false
}

func filterAgentsForWorkspace(agents []core.Agent, workspace core.Workspace) []core.Agent {
	agentSet := make(map[string]struct{}, len(workspace.AgentIDs))
	for _, agentID := range workspace.AgentIDs {
		agentSet[agentID] = struct{}{}
	}
	return filterAgents(agents, func(agent core.Agent) bool {
		_, ok := agentSet[agent.ID]
		return ok
	})
}
