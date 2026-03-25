package runtime

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ham-agents/ham-agents/go/internal/core"
)

type managedProcess struct {
	cmd      *exec.Cmd
	stopping bool
}

type ManagedService struct {
	registry  *Registry
	mu        sync.Mutex
	processes map[string]*managedProcess
}

func NewManagedService(registry *Registry) *ManagedService {
	return &ManagedService{registry: registry, processes: map[string]*managedProcess{}}
}

func (s *ManagedService) Start(ctx context.Context, input RegisterManagedInput) (core.Agent, error) {
	agent, err := s.registry.RegisterManaged(ctx, input)
	if err != nil {
		return core.Agent{}, err
	}

	cmd, commandLine, err := buildManagedCommand(agent)
	if err != nil {
		_ = s.registry.RecordManagedStartFailure(ctx, agent.ID, err.Error())
		return core.Agent{}, err
	}
	cmd.Dir = agent.ProjectPath
	cmd.Env = append(os.Environ(),
		"HAM_AGENT_ID="+agent.ID,
		"HAM_AGENT_ROLE="+agent.Role,
		"HAM_AGENT_PROJECT_PATH="+agent.ProjectPath,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = s.registry.RecordManagedStartFailure(ctx, agent.ID, err.Error())
		return core.Agent{}, fmt.Errorf("capture stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = s.registry.RecordManagedStartFailure(ctx, agent.ID, err.Error())
		return core.Agent{}, fmt.Errorf("capture stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		_ = s.registry.RecordManagedStartFailure(ctx, agent.ID, err.Error())
		return core.Agent{}, fmt.Errorf("start managed process: %w", err)
	}

	agent, err = s.registry.RecordManagedStarted(ctx, agent.ID, cmd.Process.Pid, commandLine)
	if err != nil {
		return core.Agent{}, err
	}

	s.mu.Lock()
	s.processes[agent.ID] = &managedProcess{cmd: cmd}
	s.mu.Unlock()

	go s.consumeOutput(agent.ID, stdout, false)
	go s.consumeOutput(agent.ID, stderr, true)
	go s.waitForExit(agent.ID, cmd)

	return agent, nil
}

func (s *ManagedService) Stop(ctx context.Context, agentID string) error {
	s.mu.Lock()
	proc := s.processes[agentID]
	if proc != nil {
		proc.stopping = true
	}
	s.mu.Unlock()
	if proc == nil || proc.cmd.Process == nil {
		return fmt.Errorf("managed agent %q is not running", agentID)
	}
	if err := proc.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("stop managed process: %w", err)
	}
	go func(processRef *managedProcess) {
		time.Sleep(2 * time.Second)
		s.mu.Lock()
		stillRunning := s.processes[agentID] == processRef
		s.mu.Unlock()
		if stillRunning && processRef.cmd.Process != nil {
			_ = processRef.cmd.Process.Kill()
		}
	}(proc)
	return nil
}

func (s *ManagedService) consumeOutput(agentID string, reader io.Reader, isStderr bool) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		_ = s.registry.RecordManagedOutput(context.Background(), agentID, line, isStderr)
	}
}

func (s *ManagedService) waitForExit(agentID string, cmd *exec.Cmd) {
	err := cmd.Wait()
	s.mu.Lock()
	stopping := false
	if proc, ok := s.processes[agentID]; ok {
		stopping = proc.stopping
	}
	delete(s.processes, agentID)
	s.mu.Unlock()
	if stopping {
		_ = s.registry.RecordManagedStopped(context.Background(), agentID)
		return
	}
	_ = s.registry.RecordManagedExit(context.Background(), agentID, err)
}

func buildManagedCommand(agent core.Agent) (*exec.Cmd, string, error) {
	provider := strings.TrimSpace(agent.Provider)
	if provider == "" {
		return nil, "", fmt.Errorf("provider is required")
	}
	if shell := strings.TrimSpace(os.Getenv("HAM_MANAGED_PROVIDER_" + strings.ToUpper(provider) + "_SHELL")); shell != "" {
		return exec.Command("/bin/sh", "-lc", shell), "/bin/sh -lc " + shell, nil
	}
	if scriptPath, err := exec.LookPath("script"); err == nil {
		return exec.Command(scriptPath, "-q", "/dev/null", provider), scriptPath + " -q /dev/null " + provider, nil
	}
	return exec.Command(provider), provider, nil
}
