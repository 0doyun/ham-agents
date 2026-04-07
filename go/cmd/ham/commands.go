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
	"path/filepath"
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

	// Capture the current terminal session so Open/Message can target it.
	if sessionRef := detectSessionRef(); sessionRef != "" {
		input.SessionRef = sessionRef
	}

	agent, err := client.RegisterManaged(ctx, input)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "ham: %s registered [%s]\n", agent.DisplayName, agent.ID)

	if err := ensureUIRunning(); err != nil {
		fmt.Fprintf(os.Stderr, "ham: warning: unable to auto-launch ham ui: %v\n", err)
	}

	// Run the provider inside a PTY so the user gets a full interactive session
	// while ham tees output to the daemon for state inference.
	providerBin, lookErr := exec.LookPath(agent.Provider)
	if lookErr != nil {
		_ = client.RemoveAgent(ctx, agent.ID)
		return fmt.Errorf("provider %q not found in PATH: %w", agent.Provider, lookErr)
	}

	runErr := runWithPTY(ctx, client, agent.ID, providerBin, agent.Provider, input.ProjectPath)

	// Notify daemon that the session ended so the hamster updates.
	if runErr != nil {
		_ = client.NotifyManagedExited(context.Background(), agent.ID, runErr)
		return fmt.Errorf("%s exited: %w", agent.Provider, runErr)
	}
	// Clean exit — remove the agent so the hamster disappears.
	_ = client.RemoveAgent(context.Background(), agent.ID)
	return nil
}

func runAttach(ctx context.Context, client *ipc.Client, args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "--pick-iterm-session":
			return runAttachPicker(ctx, client, args[1:], os.Stdin, os.Stdout, "iTerm session", "iterm2")
		case "--pick-tmux-session":
			return runAttachPicker(ctx, client, args[1:], os.Stdin, os.Stdout, "tmux pane", "tmux")
		case "--list-iterm-sessions":
			return runListAttachableSessions(ctx, args[1:], "iTerm sessions", client.ListItermSessions)
		case "--list-tmux-sessions":
			return runListAttachableSessions(ctx, args[1:], "tmux panes", client.ListTmuxSessions)
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

func runListAttachableSessions(ctx context.Context, args []string, emptyLabel string, list func(context.Context) ([]core.AttachableSession, error)) error {
	asJSON := false
	for _, argument := range args {
		switch argument {
		case "--json":
			asJSON = true
		default:
			return fmt.Errorf("unsupported attach listing flag %q", argument)
		}
	}

	sessions, err := list(ctx)
	if err != nil {
		return err
	}

	if asJSON {
		return writeJSON(sessions)
	}
	if len(sessions) == 0 {
		fmt.Printf("no attachable %s\n", emptyLabel)
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

func runAttachPicker(ctx context.Context, client *ipc.Client, args []string, in io.Reader, out io.Writer, promptLabel string, defaultProvider string) error {
	options, err := parseAttachPickerOptions(args)
	if err != nil {
		return err
	}

	list := client.ListItermSessions
	if defaultProvider == "tmux" {
		list = client.ListTmuxSessions
	}
	sessions, err := list(ctx)
	if err != nil {
		return err
	}
	if len(sessions) == 0 {
		return fmt.Errorf("no attachable %s", promptLabel)
	}

	if options.asJSON {
		return writeJSON(sessions)
	}

	selected, err := chooseAttachableSessionWithPrompt(in, out, sessions, promptLabel)
	if err != nil {
		return err
	}

	provider := options.provider
	if provider == "" {
		provider = defaultProvider
	}

	agent, err := client.AttachSession(ctx, runtime.RegisterAttachedInput{
		Provider:    provider,
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

func runHook(ctx context.Context, client *ipc.Client, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("hook subcommand required: tool-start, tool-done, notification, stop-failure, session-start, session-end, subagent-start, subagent-stop, teammate-idle, task-created, task-completed")
	}

	// Launch menubar early for session-start so it runs even without an agent ID.
	if args[0] == "session-start" {
		if err := ensureUIRunning(); err != nil {
			fmt.Fprintf(os.Stderr, "ham: warning: unable to auto-launch ham ui: %v\n", err)
		}
	}

	payload := readHookPayload(os.Stdin)
	agentID := os.Getenv("HAM_AGENT_ID")
	sessionRef := detectSessionRef()
	if agentID == "" && payload.SessionID == "" && args[0] != "session-start" {
		// Exit gracefully — hooks must not fail Claude Code sessions.
		fmt.Fprintf(os.Stderr, "ham: hook %s: no agent ID or session ID, skipping\n", args[0])
		return nil
	}

	var hookErr error
	switch args[0] {
	case "tool-start":
		toolName := firstNonEmpty(payload.ToolName, argAt(args, 1))
		hookErr = client.HookToolStart(ctx, agentID, payload.SessionID, sessionRef, toolName, hookToolInputPreview(toolName, payload.ToolInput), detectOmcMode())
	case "tool-done":
		toolName := firstNonEmpty(payload.ToolName, argAt(args, 1))
		hookErr = client.HookToolDone(ctx, agentID, payload.SessionID, sessionRef, toolName, hookToolInputPreview(toolName, payload.ToolInput), detectOmcMode())
	case "notification":
		hookErr = client.HookNotification(ctx, agentID, payload.SessionID, sessionRef, payload.NotificationType, detectOmcMode())
	case "stop-failure":
		hookErr = client.HookStopFailure(ctx, agentID, payload.SessionID, sessionRef, payload.ErrorType, detectOmcMode())
	case "session-start":
		hookErr = client.HookSessionStart(ctx, agentID, payload.SessionID, sessionRef, payload.Cwd, detectOmcMode())
		if hookErr != nil && agentID == "" {
			// No agent registered yet — menubar is already launched, silently succeed.
			hookErr = nil
		}
	case "stop":
		hookErr = client.HookStop(ctx, agentID, payload.SessionID, sessionRef, payload.LastMessage, detectOmcMode())
	case "session-end":
		hookErr = client.HookSessionEnd(ctx, agentID, payload.SessionID, sessionRef, detectOmcMode())
	case "subagent-start", "agent-spawned":
		description := parseHookDescription(args[1:])
		if description == "" {
			description = payload.subagentDescription()
		}
		hookErr = client.HookAgentSpawned(ctx, agentID, payload.SessionID, sessionRef, description, detectOmcMode())
	case "subagent-stop", "agent-finished":
		description := parseHookDescription(args[1:])
		if description == "" {
			description = payload.subagentCompletionDescription()
		}
		hookErr = client.HookAgentFinished(ctx, agentID, payload.SessionID, sessionRef, description, payload.LastMessage, detectOmcMode())
	case "teammate-idle":
		hookErr = client.HookTeammateIdle(ctx, agentID, payload.SessionID, sessionRef, payload.TeammateName, payload.TeamRole, detectOmcMode())
	case "task-created":
		hookErr = client.HookTaskCreated(ctx, agentID, payload.SessionID, sessionRef, payload.TaskName, payload.TaskDescription, detectOmcMode())
	case "task-completed":
		hookErr = client.HookTaskCompleted(ctx, agentID, payload.SessionID, sessionRef, payload.TaskName, detectOmcMode())
	case "tool-failed":
		toolName := firstNonEmpty(payload.ToolName, argAt(args, 1))
		hookErr = client.HookToolFailed(ctx, agentID, payload.SessionID, sessionRef, toolName, payload.Error, payload.IsInterrupt, detectOmcMode())
	case "user-prompt":
		hookErr = client.HookUserPrompt(ctx, agentID, payload.SessionID, sessionRef, payload.Prompt, detectOmcMode())
	case "permission-request":
		toolName := firstNonEmpty(payload.ToolName, argAt(args, 1))
		hookErr = client.HookPermissionRequest(ctx, agentID, payload.SessionID, sessionRef, toolName, detectOmcMode())
	case "permission-denied":
		toolName := firstNonEmpty(payload.ToolName, argAt(args, 1))
		hookErr = client.HookPermissionDenied(ctx, agentID, payload.SessionID, sessionRef, toolName, payload.Error, detectOmcMode())
	case "pre-compact":
		hookErr = client.HookPreCompact(ctx, agentID, payload.SessionID, sessionRef, payload.Trigger, detectOmcMode())
	case "post-compact":
		hookErr = client.HookPostCompact(ctx, agentID, payload.SessionID, sessionRef, payload.Trigger, payload.CompactSummary, detectOmcMode())
	case "setup":
		hookErr = client.HookSetup(ctx, agentID, payload.SessionID, sessionRef, detectOmcMode())
	case "elicitation":
		hookErr = client.HookElicitation(ctx, agentID, payload.SessionID, sessionRef, detectOmcMode())
	case "elicitation-result":
		hookErr = client.HookElicitationResult(ctx, agentID, payload.SessionID, sessionRef, detectOmcMode())
	case "config-change":
		hookErr = client.HookConfigChange(ctx, agentID, payload.SessionID, sessionRef, payload.Source, detectOmcMode())
	case "worktree-create":
		hookErr = client.HookWorktreeCreate(ctx, agentID, payload.SessionID, sessionRef, payload.WorktreeName, detectOmcMode())
	case "worktree-remove":
		hookErr = client.HookWorktreeRemove(ctx, agentID, payload.SessionID, sessionRef, payload.WorktreePath, detectOmcMode())
	case "instructions-loaded":
		hookErr = client.HookInstructionsLoaded(ctx, agentID, payload.SessionID, sessionRef, payload.FilePath, detectOmcMode())
	case "cwd-changed":
		hookErr = client.HookCwdChanged(ctx, agentID, payload.SessionID, sessionRef, payload.OldCwd, payload.NewCwd, detectOmcMode())
	case "file-changed":
		hookErr = client.HookFileChanged(ctx, agentID, payload.SessionID, sessionRef, payload.FilePath, payload.FileEvent, detectOmcMode())
	default:
		fmt.Fprintf(os.Stderr, "ham: unsupported hook subcommand %q, skipping\n", args[0])
		return nil
	}
	if hookErr != nil {
		fmt.Fprintf(os.Stderr, "ham: hook %s: %v\n", args[0], hookErr)
	}
	return nil
}

func parseHookDescription(args []string) string {
	for i, arg := range args {
		if arg == "--description" && i+1 < len(args) {
			return strings.Join(args[i+1:], " ")
		}
	}
	return ""
}

type hookPayload struct {
	SessionID           string         `json:"session_id"`
	Cwd                 string         `json:"cwd"`
	ToolName            string         `json:"tool_name"`
	ToolInput           map[string]any `json:"tool_input"`
	NotificationType    string         `json:"notification_type"`
	ErrorType           string         `json:"error_type"`
	AgentID             string         `json:"agent_id"`
	AgentType           string         `json:"agent_type"`
	AgentTranscriptPath string         `json:"agent_transcript_path"`
	TeammateName        string         `json:"teammate_name"`
	TeamRole            string         `json:"team_role"`
	TaskName            string         `json:"task_name"`
	TaskDescription     string         `json:"task_description"`
	Error               string         `json:"error"`
	IsInterrupt         bool           `json:"is_interrupt"`
	Prompt              string         `json:"prompt"`
	Trigger             string         `json:"trigger"`
	CompactSummary      string         `json:"compact_summary"`
	WorktreeName        string         `json:"name"`
	WorktreePath        string         `json:"worktree_path"`
	OldCwd              string         `json:"old_cwd"`
	NewCwd              string         `json:"new_cwd"`
	FilePath            string         `json:"file_path"`
	FileEvent           string         `json:"event"`
	Source              string         `json:"source"`
	LastMessage         string         `json:"last_assistant_message"`
}

func readHookPayload(in io.Reader) hookPayload {
	if file, ok := in.(*os.File); ok {
		if info, err := file.Stat(); err == nil && info.Mode()&os.ModeCharDevice != 0 {
			return hookPayload{}
		}
	}

	data, err := io.ReadAll(in)
	if err != nil || len(data) == 0 {
		return hookPayload{}
	}

	var payload hookPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return hookPayload{}
	}
	return payload
}

func hookToolInputPreview(toolName string, toolInput map[string]any) string {
	if len(toolInput) == 0 {
		return ""
	}
	return summarizeToolInput(toolName, toolInput)
}

func argAt(args []string, index int) string {
	if index >= 0 && index < len(args) {
		return args[index]
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func (p hookPayload) subagentDescription() string {
	if value := firstNonEmpty(p.AgentType, p.AgentID); value != "" {
		return value
	}
	return ""
}

func (p hookPayload) subagentCompletionDescription() string {
	if value := firstNonEmpty(p.AgentTranscriptPath, p.AgentType, p.AgentID); value != "" {
		return value
	}
	return ""
}

func summarizeToolInput(toolName string, input map[string]any) string {
	tool := strings.ToLower(strings.TrimSpace(toolName))
	switch tool {
	case "read", "edit", "write", "multiedit":
		return firstToolInputString(input, "file_path")
	case "bash":
		return compactToolPreview(firstToolInputString(input, "command", "cmd"))
	case "grep":
		if preview := firstToolInputString(input, "pattern"); preview != "" {
			return compactToolPreview(preview)
		}
		return firstToolInputString(input, "path", "file_path")
	case "glob":
		return compactToolPreview(firstToolInputString(input, "pattern"))
	case "webfetch":
		return firstToolInputString(input, "url")
	case "websearch":
		return compactToolPreview(firstToolInputString(input, "query"))
	case "agent", "task":
		return compactToolPreview(firstToolInputString(input, "description", "prompt"))
	default:
		return compactToolPreview(firstToolInputString(input, "file_path", "path", "command", "pattern", "url", "description", "prompt"))
	}
}

func firstToolInputString(input map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := input[key]
		if !ok {
			continue
		}
		if text, ok := value.(string); ok {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func compactToolPreview(value string) string {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.ReplaceAll(trimmed, "\n", " ")
	trimmed = strings.Join(strings.Fields(trimmed), " ")
	if len(trimmed) > 80 {
		return trimmed[:77] + "..."
	}
	return trimmed
}

func runInbox(ctx context.Context, client *ipc.Client, args []string) error {
	opts, err := parseInboxOptions(args)
	if err != nil {
		return err
	}

	if opts.markRead {
		n, err := client.InboxMarkRead(ctx, opts.markReadID)
		if err != nil {
			return err
		}
		if opts.markReadID != "" {
			fmt.Printf("Marked item %s as read (%d unread remaining)\n", opts.markReadID, n)
		} else {
			fmt.Printf("Marked all items as read (%d unread remaining)\n", n)
		}
		return nil
	}

	unreadOnly := !opts.all
	items, _, err := client.InboxList(ctx, opts.typeFilter, unreadOnly)
	if err != nil {
		return err
	}
	return renderInboxItems(os.Stdout, items)
}

func runStop(ctx context.Context, client *ipc.Client, args []string) error {
	agentID, asJSON, err := parseStopInput(args)
	if err != nil {
		return err
	}

	if err := client.StopManaged(ctx, agentID); err != nil {
		return err
	}

	return renderStopResult(os.Stdout, agentID, asJSON)
}

func runRename(ctx context.Context, client *ipc.Client, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: ham rename <agent-id> <new-name>")
	}
	agent, err := client.RenameAgent(ctx, args[0], strings.Join(args[1:], " "))
	if err != nil {
		return err
	}
	fmt.Printf("renamed to %s\n", agent.DisplayName)
	return nil
}

func runDown(_ context.Context, client *ipc.Client, _ []string) error {
	ctx := context.Background()

	// Kill the menu bar app.
	if pkillPath, err := exec.LookPath("pkill"); err == nil {
		_ = exec.Command(pkillPath, "-x", "ham-menubar").Run()
	}

	// Tell the daemon to stop all agents and shut down.
	if err := client.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "ham: daemon shutdown: %v\n", err)
	}

	// If managed via launchd, unload so it doesn't auto-restart.
	_ = uninstallDaemonFromLaunchd()

	fmt.Fprintln(os.Stderr, "ham: everything stopped")
	return nil
}

func runUninstall(_ context.Context, client *ipc.Client, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	purge := false
	for _, arg := range args {
		if arg == "--purge" {
			purge = true
		}
	}

	fmt.Fprintln(stdout, "Uninstalling ham-agents...")

	// Step 1: Kill menu bar app.
	if pkillPath, err := exec.LookPath("pkill"); err == nil {
		_ = exec.Command(pkillPath, "-x", "ham-menubar").Run()
		fmt.Fprintln(stdout, "  Menu bar app stopped.")
	}

	// Step 2: Shutdown daemon.
	if err := client.Shutdown(context.Background()); err != nil {
		fmt.Fprintf(stderr, "  Daemon shutdown: %v (may already be stopped)\n", err)
	} else {
		fmt.Fprintln(stdout, "  Daemon stopped.")
	}

	// Step 3: Remove launchd plist.
	_ = uninstallDaemonFromLaunchd()
	fmt.Fprintln(stdout, "  Launchd agent removed.")

	// Step 4: Remove ham hooks from Claude Code settings.json.
	home, err := os.UserHomeDir()
	if err == nil {
		settingsPath := filepath.Join(home, ".claude", "settings.json")
		if data, readErr := os.ReadFile(settingsPath); readErr == nil {
			var settings map[string]interface{}
			if json.Unmarshal(data, &settings) == nil {
				removed := removeHamHooks(settings)
				if removed > 0 {
					if out, marshalErr := json.MarshalIndent(settings, "", "  "); marshalErr == nil {
						_ = os.WriteFile(settingsPath, append(out, '\n'), 0o644)
					}
					fmt.Fprintf(stdout, "  Removed ham hooks from %d categories in settings.json.\n", removed)
				} else {
					fmt.Fprintln(stdout, "  No ham hooks found in settings.json.")
				}
			}
		} else {
			fmt.Fprintln(stdout, "  No Claude Code settings.json found (skipped).")
		}
	}

	// Step 5: Optionally remove ~/.ham-agents/ data directory.
	if home != "" && !purge {
		dataDir := filepath.Join(home, ".ham-agents")
		if _, statErr := os.Stat(dataDir); statErr == nil {
			fmt.Fprintf(stdout, "\n  Data directory exists: %s\n", dataDir)
			fmt.Fprint(stdout, "  Remove it? This deletes all agent history and settings. [y/N] ")
			scanner := bufio.NewScanner(stdin)
			if scanner.Scan() {
				answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
				if answer == "y" || answer == "yes" {
					if err := os.RemoveAll(dataDir); err != nil {
						fmt.Fprintf(stderr, "  Failed to remove data directory: %v\n", err)
					} else {
						fmt.Fprintln(stdout, "  Data directory removed.")
					}
				} else {
					fmt.Fprintln(stdout, "  Data directory kept.")
				}
			}
		}
	} else if purge && home != "" {
		dataDir := filepath.Join(home, ".ham-agents")
		if err := os.RemoveAll(dataDir); err != nil {
			fmt.Fprintf(stderr, "  Failed to remove data directory: %v\n", err)
		} else {
			fmt.Fprintln(stdout, "  Data directory removed.")
		}
	}

	fmt.Fprintln(stdout, "\nham-agents uninstalled (hooks, daemon, launchd removed).")
	fmt.Fprintln(stdout, "Binaries (ham, hamd) are still installed.")
	fmt.Fprintln(stdout, "  To reconfigure:    ham setup")
	fmt.Fprintln(stdout, "  To fully remove:   brew uninstall ham")
	return nil
}

func runDetach(ctx context.Context, client *ipc.Client, args []string) error {
	agentID, asJSON, err := parseStopInput(args)
	if err != nil {
		return err
	}
	if err := client.RemoveAgent(ctx, agentID); err != nil {
		return err
	}
	return renderDetachResult(os.Stdout, agentID, asJSON)
}

func runLogs(ctx context.Context, client *ipc.Client, args []string) error {
	agentID, limit, asJSON, exportPath, err := parseLogsInput(args)
	if err != nil {
		return err
	}

	events, err := client.Events(ctx, agentLogFetchLimit(limit))
	if err != nil {
		return err
	}
	filtered := eventsForAgent(events, agentID, limit)
	if exportPath == "" {
		return printEvents(os.Stdout, filtered, asJSON)
	}
	file, err := os.Create(exportPath)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := printEvents(file, filtered, asJSON); err != nil {
		return err
	}
	_, err = fmt.Fprintf(os.Stdout, "exported logs to %s\n", exportPath)
	return err
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

	if err := startDetachedProcess(detachedLaunchTarget{Executable: target.Executable}); err != nil {
		return fmt.Errorf("launch ham ui: %w", err)
	}
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

	if options.graph {
		graph, snapshot, err := client.StatusWithGraph(ctx)
		if err != nil {
			return err
		}
		filteredAgents, err := filterAgentsForQuery(ctx, client, snapshot.Agents, options.teamRef, options.workspaceRef)
		if err != nil {
			return err
		}
		if options.teamRef != "" || options.workspaceRef != "" {
			// Rebuild graph from filtered agent set.
			rebuiltGraph := buildFilteredGraph(filteredAgents, graph.GeneratedAt)
			return renderSessionGraph(os.Stdout, rebuiltGraph)
		}
		return renderSessionGraph(os.Stdout, graph)
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
	return chooseAttachableSessionWithPrompt(in, out, sessions, "iTerm session")
}

func chooseAttachableSessionWithPrompt(in io.Reader, out io.Writer, sessions []core.AttachableSession, promptLabel string) (core.AttachableSession, error) {
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
	if _, err := fmt.Fprintf(out, "Select %s: ", promptLabel); err != nil {
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
