package ipc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/core"
	hamruntime "github.com/ham-agents/ham-agents/go/internal/runtime"
	"github.com/ham-agents/ham-agents/go/internal/store"
)

type SessionLister interface {
	ListSessions() ([]core.AttachableSession, error)
}

type Server struct {
	socketPath         string
	registry           *hamruntime.Registry
	managed            *hamruntime.ManagedService
	settings           *hamruntime.SettingsService
	teams              *hamruntime.TeamService
	inbox              *hamruntime.InboxManager
	itermSessionLister SessionLister
	tmuxSessionLister  SessionLister
	costStore          store.CostStore

	listener   net.Listener
	cancelFunc context.CancelFunc
}

// SetCostStore wires a CostStore so the server can serve CommandCostSummary
// requests. Optional — when nil the cost.summary command returns an empty
// response. Must be called before Serve.
func (s *Server) SetCostStore(costStore store.CostStore) {
	s.costStore = costStore
}

func NewServer(socketPath string, registry *hamruntime.Registry, managed *hamruntime.ManagedService, settings *hamruntime.SettingsService, teams *hamruntime.TeamService, inbox *hamruntime.InboxManager, itermSessionLister SessionLister, tmuxSessionLister SessionLister) *Server {
	return &Server{
		socketPath:         socketPath,
		registry:           registry,
		managed:            managed,
		settings:           settings,
		teams:              teams,
		inbox:              inbox,
		itermSessionLister: itermSessionLister,
		tmuxSessionLister:  tmuxSessionLister,
	}
}

func (s *Server) Serve(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	s.cancelFunc = cancel

	if err := os.MkdirAll(filepath.Dir(s.socketPath), 0o755); err != nil {
		return fmt.Errorf("create socket directory: %w", err)
	}

	if err := removeStaleSocket(s.socketPath); err != nil {
		return err
	}

	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("listen on socket: %w", err)
	}
	s.listener = listener
	defer func() {
		_ = listener.Close()
		_ = os.Remove(s.socketPath)
	}()

	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	for {
		connection, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("accept connection: %w", err)
		}

		go s.handleConnection(ctx, connection)
	}
}

func removeStaleSocket(socketPath string) error {
	info, err := os.Lstat(socketPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect socket path: %w", err)
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("socket path exists and is not a unix socket: %s", socketPath)
	}
	if err := os.Remove(socketPath); err != nil {
		return fmt.Errorf("remove stale socket: %w", err)
	}
	return nil
}

func (s *Server) handleConnection(ctx context.Context, connection net.Conn) {
	defer connection.Close()

	// Prevent slow or malicious clients from holding a goroutine indefinitely.
	_ = connection.SetReadDeadline(time.Now().Add(10 * time.Second))

	var request Request
	if err := json.NewDecoder(io.LimitReader(connection, 1<<20)).Decode(&request); err != nil {
		_ = json.NewEncoder(connection).Encode(Response{Error: fmt.Sprintf("decode request: %v", err)})
		closeWrite(connection)
		return
	}

	response, err := s.dispatch(ctx, request)
	if err != nil {
		_ = json.NewEncoder(connection).Encode(Response{Error: err.Error()})
		closeWrite(connection)
		return
	}

	_ = json.NewEncoder(connection).Encode(response)
	closeWrite(connection)
}

// closeWrite signals the client that no more data will be sent,
// so the client's read loop can terminate instead of blocking forever.
func closeWrite(conn net.Conn) {
	if uc, ok := conn.(*net.UnixConn); ok {
		_ = uc.CloseWrite()
	}
}

func (s *Server) dispatch(ctx context.Context, request Request) (Response, error) {
	switch request.Command {
	case CommandRunManaged:
		if s.managed == nil {
			return Response{}, fmt.Errorf("managed service is not configured")
		}
		agent, err := s.managed.Start(ctx, hamruntime.RegisterManagedInput{
			Provider:    request.Provider,
			DisplayName: request.DisplayName,
			ProjectPath: request.ProjectPath,
			Role:        request.Role,
		})
		if err != nil {
			return Response{}, err
		}
		return Response{Agent: &agent}, nil
	case CommandRegisterManaged:
		agent, err := s.registry.RegisterManaged(ctx, hamruntime.RegisterManagedInput{
			Provider:    request.Provider,
			DisplayName: request.DisplayName,
			ProjectPath: request.ProjectPath,
			Role:        request.Role,
			SessionRef:  request.SessionRef,
		})
		if err != nil {
			return Response{}, err
		}
		return Response{Agent: &agent}, nil
	case CommandRecordOutput:
		if err := s.registry.RecordManagedOutput(ctx, request.AgentID, request.OutputLine, false, true); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandNotifyManagedExited:
		var exitErr error
		if request.ExitError != "" {
			exitErr = fmt.Errorf("%s", request.ExitError)
		}
		if err := s.registry.RecordManagedExit(ctx, request.AgentID, exitErr); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandStopManaged:
		if s.managed == nil {
			return Response{}, fmt.Errorf("managed service is not configured")
		}
		if err := s.managed.Stop(ctx, request.AgentID); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandAttachSession:
		agent, err := s.registry.RegisterAttached(ctx, hamruntime.RegisterAttachedInput{
			Provider:    request.Provider,
			DisplayName: request.DisplayName,
			ProjectPath: request.ProjectPath,
			Role:        request.Role,
			SessionRef:  request.SessionRef,
		})
		if err != nil {
			return Response{}, err
		}
		return Response{Agent: &agent}, nil
	case CommandObserveSource:
		agent, err := s.registry.RegisterObserved(ctx, hamruntime.RegisterObservedInput{
			Provider:    request.Provider,
			DisplayName: request.DisplayName,
			ProjectPath: request.ProjectPath,
			Role:        request.Role,
			SessionRef:  request.SessionRef,
		})
		if err != nil {
			return Response{}, err
		}
		return Response{Agent: &agent}, nil
	case CommandCreateTeam:
		if s.teams == nil {
			return Response{}, fmt.Errorf("team service is not configured")
		}
		team, err := s.teams.Create(ctx, request.DisplayName)
		if err != nil {
			return Response{}, err
		}
		return Response{Team: &team}, nil
	case CommandAddTeamMember:
		if s.teams == nil {
			return Response{}, fmt.Errorf("team service is not configured")
		}
		team, err := s.teams.AddMember(ctx, request.TeamRef, request.MemberAgentID)
		if err != nil {
			return Response{}, err
		}
		return Response{Team: &team}, nil
	case CommandListTeams:
		if s.teams == nil {
			return Response{Teams: []core.Team{}}, nil
		}
		teams, err := s.teams.List(ctx)
		if err != nil {
			return Response{}, err
		}
		return Response{Teams: teams}, nil
	case CommandOpenTarget:
		target, err := s.registry.OpenTarget(ctx, request.AgentID)
		if err != nil {
			return Response{}, err
		}
		return Response{OpenTarget: &target}, nil
	case CommandListItermSessions:
		if s.itermSessionLister == nil {
			return Response{AttachableSessions: []core.AttachableSession{}}, nil
		}
		sessions, err := s.itermSessionLister.ListSessions()
		if err != nil {
			return Response{}, err
		}
		return Response{AttachableSessions: sessions}, nil
	case CommandListTmuxSessions:
		if s.tmuxSessionLister == nil {
			return Response{AttachableSessions: []core.AttachableSession{}}, nil
		}
		sessions, err := s.tmuxSessionLister.ListSessions()
		if err != nil {
			return Response{}, err
		}
		return Response{AttachableSessions: sessions}, nil
	case CommandListAgents:
		agents, err := s.registry.List(ctx)
		if err != nil {
			return Response{}, err
		}
		return Response{Agents: agents}, nil
	case CommandStatus:
		snapshot, err := s.registry.Snapshot(ctx)
		if err != nil {
			return Response{}, err
		}
		resp := Response{Snapshot: &snapshot}
		if request.Graph {
			graph := core.BuildSessionGraph(snapshot.Agents)
			resp.SessionGraph = &graph
		}
		return resp, nil
	case CommandEvents:
		events, err := s.registry.Events(ctx, request.Limit)
		if err != nil {
			return Response{}, err
		}
		return Response{Events: events}, nil
	case CommandFollowEvents:
		events, err := s.registry.FollowEvents(ctx, request.AfterEventID, request.Limit, time.Duration(request.WaitMillis)*time.Millisecond)
		if err != nil {
			return Response{}, err
		}
		return Response{Events: events}, nil
	case CommandSetNotificationPolicy:
		agent, err := s.registry.UpdateNotificationPolicy(ctx, request.AgentID, core.NotificationPolicy(request.Policy))
		if err != nil {
			return Response{}, err
		}
		return Response{Agent: &agent}, nil
	case CommandSetRole:
		agent, err := s.registry.UpdateRole(ctx, request.AgentID, request.Role)
		if err != nil {
			return Response{}, err
		}
		return Response{Agent: &agent}, nil
	case CommandRenameAgent:
		agent, err := s.registry.Rename(ctx, request.AgentID, request.DisplayName)
		if err != nil {
			return Response{}, err
		}
		return Response{Agent: &agent}, nil
	case CommandRemoveAgent:
		if err := s.registry.Remove(ctx, request.AgentID); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandGetSettings:
		settings, err := s.settings.Get(ctx)
		if err != nil {
			return Response{}, err
		}
		return Response{Settings: &settings}, nil
	case CommandUpdateSettings:
		if request.Settings == nil {
			return Response{}, fmt.Errorf("settings payload is required")
		}
		settings, err := s.settings.Update(ctx, *request.Settings)
		if err != nil {
			return Response{}, err
		}
		return Response{Settings: &settings}, nil
	case CommandHookToolStart:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			if errors.Is(err, errNoAgent) {
				return Response{}, nil
			}
			return Response{}, err
		}
		if err := s.registry.RecordHookToolStart(ctx, request.AgentID, request.ToolName, request.ToolInputPreview, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookToolDone:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			if errors.Is(err, errNoAgent) {
				return Response{}, nil
			}
			return Response{}, err
		}
		if err := s.registry.RecordHookToolDone(ctx, request.AgentID, request.ToolName, request.ToolInputPreview, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookNotification:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			if errors.Is(err, errNoAgent) {
				return Response{}, nil
			}
			return Response{}, err
		}
		if err := s.registry.RecordHookNotification(ctx, request.AgentID, request.NotificationType, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookStopFailure:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			if errors.Is(err, errNoAgent) {
				return Response{}, nil
			}
			return Response{}, err
		}
		if err := s.registry.RecordHookStopFailure(ctx, request.AgentID, request.ErrorType, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookSessionStart:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			if errors.Is(err, errNoAgent) {
				return Response{}, nil
			}
			return Response{}, err
		}
		if err := s.registry.RecordHookSessionStart(ctx, request.AgentID, request.SessionID, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookSessionEnd:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			if errors.Is(err, errNoAgent) {
				return Response{}, nil
			}
			return Response{}, err
		}
		if err := s.registry.RecordHookSessionEnd(ctx, request.AgentID, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookAgentSpawned:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			if errors.Is(err, errNoAgent) {
				return Response{}, nil
			}
			return Response{}, err
		}
		if err := s.registry.RecordHookAgentSpawned(ctx, request.AgentID, request.Description, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookAgentFinished:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			if errors.Is(err, errNoAgent) {
				return Response{}, nil
			}
			return Response{}, err
		}
		if err := s.registry.RecordHookAgentFinished(ctx, request.AgentID, request.Description, request.LastMessage, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookStop:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			if errors.Is(err, errNoAgent) {
				return Response{}, nil
			}
			return Response{}, err
		}
		if err := s.registry.RecordHookStop(ctx, request.AgentID, request.LastMessage, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookTeammateIdle:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			if errors.Is(err, errNoAgent) {
				return Response{}, nil
			}
			return Response{}, err
		}
		if err := s.registry.RecordHookTeammateIdle(ctx, request.AgentID, request.TeammateName, request.TeamRole, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookTaskCreated:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			if errors.Is(err, errNoAgent) {
				return Response{}, nil
			}
			return Response{}, err
		}
		if err := s.registry.RecordHookTaskCreated(ctx, request.AgentID, request.TaskName, request.TaskDescription, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookTaskCompleted:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			if errors.Is(err, errNoAgent) {
				return Response{}, nil
			}
			return Response{}, err
		}
		if err := s.registry.RecordHookTaskCompleted(ctx, request.AgentID, request.TaskName, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookToolFailed:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			if errors.Is(err, errNoAgent) {
				return Response{}, nil
			}
			return Response{}, err
		}
		if err := s.registry.RecordHookToolFailed(ctx, request.AgentID, request.ToolName, request.Description, request.IsInterrupt, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookUserPrompt:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			if errors.Is(err, errNoAgent) {
				return Response{}, nil
			}
			return Response{}, err
		}
		if err := s.registry.RecordHookUserPrompt(ctx, request.AgentID, request.Prompt, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookPermissionReq:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			if errors.Is(err, errNoAgent) {
				return Response{}, nil
			}
			return Response{}, err
		}
		if err := s.registry.RecordHookPermissionRequest(ctx, request.AgentID, request.ToolName, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookPermissionDenied:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			if errors.Is(err, errNoAgent) {
				return Response{}, nil
			}
			return Response{}, err
		}
		if err := s.registry.RecordHookPermissionDenied(ctx, request.AgentID, request.ToolName, request.Description, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookPreCompact:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			if errors.Is(err, errNoAgent) {
				return Response{}, nil
			}
			return Response{}, err
		}
		if err := s.registry.RecordHookPreCompact(ctx, request.AgentID, request.CompactTrigger, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookPostCompact:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			if errors.Is(err, errNoAgent) {
				return Response{}, nil
			}
			return Response{}, err
		}
		if err := s.registry.RecordHookPostCompact(ctx, request.AgentID, request.CompactTrigger, request.CompactSummary, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookSetup:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			if errors.Is(err, errNoAgent) {
				return Response{}, nil
			}
			return Response{}, err
		}
		if err := s.registry.RecordHookSetup(ctx, request.AgentID, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookElicitation:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			if errors.Is(err, errNoAgent) {
				return Response{}, nil
			}
			return Response{}, err
		}
		if err := s.registry.RecordHookElicitation(ctx, request.AgentID, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookElicitationResult:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			if errors.Is(err, errNoAgent) {
				return Response{}, nil
			}
			return Response{}, err
		}
		if err := s.registry.RecordHookElicitationResult(ctx, request.AgentID, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookConfigChange:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			if errors.Is(err, errNoAgent) {
				return Response{}, nil
			}
			return Response{}, err
		}
		if err := s.registry.RecordHookConfigChange(ctx, request.AgentID, request.Description, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookWorktreeCreate:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			if errors.Is(err, errNoAgent) {
				return Response{}, nil
			}
			return Response{}, err
		}
		if err := s.registry.RecordHookWorktreeCreate(ctx, request.AgentID, request.WorktreeName, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookWorktreeRemove:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			if errors.Is(err, errNoAgent) {
				return Response{}, nil
			}
			return Response{}, err
		}
		if err := s.registry.RecordHookWorktreeRemove(ctx, request.AgentID, request.WorktreePath, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookInstructions:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			if errors.Is(err, errNoAgent) {
				return Response{}, nil
			}
			return Response{}, err
		}
		if err := s.registry.RecordHookInstructionsLoaded(ctx, request.AgentID, request.FilePath, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookCwdChanged:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			if errors.Is(err, errNoAgent) {
				return Response{}, nil
			}
			return Response{}, err
		}
		if err := s.registry.RecordHookCwdChanged(ctx, request.AgentID, request.OldCwd, request.NewCwd, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookFileChanged:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			if errors.Is(err, errNoAgent) {
				return Response{}, nil
			}
			return Response{}, err
		}
		if err := s.registry.RecordHookFileChanged(ctx, request.AgentID, request.FilePath, request.FileEvent, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandInboxList:
		if s.inbox == nil {
			return Response{InboxItems: []core.InboxItem{}, UnreadCount: 0}, nil
		}
		items := s.inbox.List(core.InboxItemType(request.TypeFilter), request.UnreadOnly)
		if items == nil {
			items = []core.InboxItem{}
		}
		return Response{InboxItems: items, UnreadCount: s.inbox.UnreadCount()}, nil
	case CommandInboxMarkRead:
		if s.inbox != nil {
			if request.InboxItemID == "" {
				s.inbox.MarkAllRead()
			} else {
				s.inbox.MarkRead(request.InboxItemID)
			}
		}
		unread := 0
		if s.inbox != nil {
			unread = s.inbox.UnreadCount()
		}
		return Response{UnreadCount: unread}, nil
	case CommandCostSummary:
		return s.handleCostSummary(ctx, request)
	case CommandShutdown:
		if s.managed != nil {
			s.managed.StopAll(ctx)
		}
		go func() {
			// Small delay so the response can be sent before the server shuts down.
			time.Sleep(100 * time.Millisecond)
			if s.cancelFunc != nil {
				s.cancelFunc()
			}
		}()
		return Response{}, nil
	default:
		return Response{}, fmt.Errorf("unsupported command %q", request.Command)
	}
}

func (s *Server) prepareHookRequest(ctx context.Context, request *Request) error {
	if request == nil {
		return fmt.Errorf("hook request is required")
	}
	resolvedAgentID, err := s.resolveHookAgentID(ctx, *request)
	if err != nil {
		return err
	}
	if resolvedAgentID == "" {
		return errNoAgent
	}
	request.AgentID = resolvedAgentID
	if strings.TrimSpace(request.SessionID) != "" {
		if err := s.registry.RecordHookSessionSeen(ctx, request.AgentID, request.SessionID); err != nil {
			return err
		}
	}
	if strings.TrimSpace(request.SessionRef) != "" {
		if err := s.registry.RecordHookSessionRefSeen(ctx, request.AgentID, request.SessionRef); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) resolveHookAgentID(ctx context.Context, request Request) (string, error) {
	if sessionID := strings.TrimSpace(request.SessionID); sessionID != "" {
		agent, err := s.registry.FindAgentBySessionID(ctx, sessionID)
		if err == nil {
			return agent.ID, nil
		}
		// Auto-register a new agent when SessionStart fires with an unknown session.
		// This lets plain `claude` sessions get tracked without `ham run claude`.
		if request.Command == CommandHookSessionStart {
			displayName := autoDisplayName(request.ProjectPath, s.registry, ctx)
			newAgent, regErr := s.registry.RegisterManaged(ctx, hamruntime.RegisterManagedInput{
				Provider:    "claude",
				DisplayName: displayName,
				ProjectPath: request.ProjectPath,
				SessionRef:  request.SessionRef,
			})
			if regErr != nil {
				return "", fmt.Errorf("auto-register agent for session %q: %w", sessionID, regErr)
			}
			return newAgent.ID, nil
		}
	}
	if agentID := strings.TrimSpace(request.AgentID); agentID != "" {
		return agentID, nil
	}
	// For non-SessionStart hooks, silently ignore if no agent is found.
	// This avoids errors when e.g. a Stop hook fires after the agent was removed.
	return "", errNoAgent
}

// autoDisplayName derives a display name from the project path (e.g. "/Users/gong/projects/ham-agents" → "ham-agents").
// If an agent with the same name already exists, appends a number suffix.
func autoDisplayName(projectPath string, registry *hamruntime.Registry, ctx context.Context) string {
	base := "claude"
	if projectPath != "" {
		parts := strings.Split(strings.TrimRight(projectPath, "/"), "/")
		if len(parts) > 0 && parts[len(parts)-1] != "" {
			base = parts[len(parts)-1]
		}
	}

	agents, err := registry.List(ctx)
	if err != nil {
		return base
	}

	taken := make(map[string]bool)
	for _, a := range agents {
		taken[a.DisplayName] = true
	}

	if !taken[base] {
		return base
	}
	for i := 2; i <= 99; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if !taken[candidate] {
			return candidate
		}
	}
	return base
}

// errNoAgent is returned when a hook fires but no matching agent exists.
// The server treats this as a no-op rather than an error.
var errNoAgent = fmt.Errorf("no matching agent")

// handleCostSummary loads cost records from the configured CostStore and
// builds the response payload. When no store is wired (e.g. in tests) it
// returns an empty response so callers can still hit the daemon without
// crashing.
func (s *Server) handleCostSummary(ctx context.Context, request Request) (Response, error) {
	if s.costStore == nil {
		return Response{
			CostRecords: []core.CostRecord{},
			ByModel:     map[string]float64{},
			ByDay:       map[string]float64{},
			ByAgent:     map[string]float64{},
		}, nil
	}

	filter := store.CostFilter{AgentID: strings.TrimSpace(request.AgentIDFilter)}
	if request.SinceDays > 0 {
		filter.Since = time.Now().UTC().AddDate(0, 0, -request.SinceDays)
	}

	records, err := s.costStore.Load(ctx, filter)
	if err != nil {
		return Response{}, err
	}
	if records == nil {
		records = []core.CostRecord{}
	}

	response := Response{
		CostRecords: records,
		ByModel:     map[string]float64{},
		ByDay:       map[string]float64{},
		ByAgent:     map[string]float64{},
	}
	for _, record := range records {
		response.TotalUSD += record.EstimatedUSD
		if record.Model != "" {
			response.ByModel[record.Model] += record.EstimatedUSD
		}
		if !record.RecordedAt.IsZero() {
			day := record.RecordedAt.UTC().Format("2006-01-02")
			response.ByDay[day] += record.EstimatedUSD
		}
		if record.AgentID != "" {
			response.ByAgent[record.AgentID] += record.EstimatedUSD
		} else {
			response.ByAgent["(orphan)"] += record.EstimatedUSD
		}
	}
	return response, nil
}
