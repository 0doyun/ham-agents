package ipc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/ham-agents/ham-agents/go/internal/core"
	hamruntime "github.com/ham-agents/ham-agents/go/internal/runtime"
)

type Server struct {
	socketPath string
	registry   *hamruntime.Registry

	listener net.Listener
}

func NewServer(socketPath string, registry *hamruntime.Registry) *Server {
	return &Server{
		socketPath: socketPath,
		registry:   registry,
	}
}

func (s *Server) Serve(ctx context.Context) error {
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
		agent, err := s.registry.RegisterManaged(ctx, hamruntime.RegisterManagedInput{
			Provider:    request.Provider,
			DisplayName: request.DisplayName,
			ProjectPath: request.ProjectPath,
			Role:        request.Role,
		})
		if err != nil {
			return Response{}, err
		}
		return Response{Agent: &agent}, nil
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
	case CommandSetNotificationPolicy:
		agent, err := s.registry.UpdateNotificationPolicy(ctx, request.AgentID, core.NotificationPolicy(request.Policy))
		if err != nil {
			return Response{}, err
		}
		return Response{Agent: &agent}, nil
	default:
		return Response{}, fmt.Errorf("unsupported command %q", request.Command)
	}
}
