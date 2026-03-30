package ipc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/core"
	hamruntime "github.com/ham-agents/ham-agents/go/internal/runtime"
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
	itermSessionLister SessionLister
	tmuxSessionLister  SessionLister

	listener   net.Listener
	cancelFunc context.CancelFunc
}

func NewServer(socketPath string, registry *hamruntime.Registry, managed *hamruntime.ManagedService, settings *hamruntime.SettingsService, teams *hamruntime.TeamService, itermSessionLister SessionLister, tmuxSessionLister SessionLister) *Server {
	return &Server{
		socketPath:         socketPath,
		registry:           registry,
		managed:            managed,
		settings:           settings,
		teams:              teams,
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

	var request Request
	if err := json.NewDecoder(connection).Decode(&request); err != nil {
		_ = json.NewEncoder(connection).Encode(Response{Error: fmt.Sprintf("decode request: %v", err)})
		return
	}

	response, err := s.dispatch(ctx, request)
	if err != nil {
		_ = json.NewEncoder(connection).Encode(Response{Error: err.Error()})
		return
	}

	_ = json.NewEncoder(connection).Encode(response)
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
		return Response{Snapshot: &snapshot}, nil
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
			return Response{}, err
		}
		if err := s.registry.RecordHookToolStart(ctx, request.AgentID, request.ToolName, request.ToolInputPreview, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookToolDone:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			return Response{}, err
		}
		if err := s.registry.RecordHookToolDone(ctx, request.AgentID, request.ToolName, request.ToolInputPreview, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookNotification:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			return Response{}, err
		}
		if err := s.registry.RecordHookNotification(ctx, request.AgentID, request.NotificationType, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookStopFailure:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			return Response{}, err
		}
		if err := s.registry.RecordHookStopFailure(ctx, request.AgentID, request.ErrorType, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookSessionStart:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			return Response{}, err
		}
		if err := s.registry.RecordHookSessionStart(ctx, request.AgentID, request.SessionID, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookSessionEnd:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			return Response{}, err
		}
		if err := s.registry.RecordHookSessionEnd(ctx, request.AgentID, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookAgentSpawned:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			return Response{}, err
		}
		if err := s.registry.RecordHookAgentSpawned(ctx, request.AgentID, request.Description, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookAgentFinished:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			return Response{}, err
		}
		if err := s.registry.RecordHookAgentFinished(ctx, request.AgentID, request.Description, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookTeammateIdle:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			return Response{}, err
		}
		if err := s.registry.RecordHookTeammateIdle(ctx, request.AgentID, request.TeammateName, request.TeamRole, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookTaskCreated:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			return Response{}, err
		}
		if err := s.registry.RecordHookTaskCreated(ctx, request.AgentID, request.TaskName, request.TaskDescription, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
	case CommandHookTaskCompleted:
		if err := s.prepareHookRequest(ctx, &request); err != nil {
			return Response{}, err
		}
		if err := s.registry.RecordHookTaskCompleted(ctx, request.AgentID, request.TaskName, request.OmcMode); err != nil {
			return Response{}, err
		}
		return Response{}, nil
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
	if strings.TrimSpace(request.SessionID) != "" && strings.TrimSpace(request.AgentID) != "" {
		if err := s.registry.RecordHookSessionSeen(ctx, request.AgentID, request.SessionID); err != nil {
			return err
		}
	}
	resolvedAgentID, err := s.resolveHookAgentID(ctx, *request)
	if err != nil {
		return err
	}
	request.AgentID = resolvedAgentID
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
			newAgent, regErr := s.registry.RegisterManaged(ctx, hamruntime.RegisterManagedInput{
				Provider: "claude",
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
	return "", fmt.Errorf("hook request requires HAM_AGENT_ID or a known session_id")
}
