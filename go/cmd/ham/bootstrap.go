package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	daemonBootstrapTimeout      = 3 * time.Second
	daemonBootstrapPollInterval = 100 * time.Millisecond
)

type detachedLaunchTarget struct {
	Executable string
	Args       []string
	Dir        string
}

type daemonBootstrapDependencies struct {
	executablePath func() (string, error)
	lookupEnv      func(string) (string, bool)
	getwd          func() (string, error)
	lookPath       func(string) (string, error)
	dial           func(string, time.Duration) (net.Conn, error)
	start          func(detachedLaunchTarget) error
	sleep          func(time.Duration)
	now            func() time.Time
}

func defaultDaemonBootstrapDependencies() daemonBootstrapDependencies {
	return daemonBootstrapDependencies{
		executablePath: os.Executable,
		lookupEnv:      os.LookupEnv,
		getwd:          os.Getwd,
		lookPath:       exec.LookPath,
		dial: func(socketPath string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", socketPath, timeout)
		},
		start: startDetachedProcess,
		sleep: time.Sleep,
		now:   time.Now,
	}
}

func ensureDaemon(socketPath string) error {
	return ensureDaemonWithDependencies(socketPath, defaultDaemonBootstrapDependencies())
}

func ensureDaemonWithDependencies(socketPath string, deps daemonBootstrapDependencies) error {
	if err := checkDaemonReachable(socketPath, deps.dial); err == nil {
		return nil
	}

	target, err := resolveDaemonLaunchTarget(deps.executablePath, deps.lookupEnv, deps.getwd, deps.lookPath)
	if err != nil {
		return fmt.Errorf("resolve daemon bootstrap target: %w", err)
	}
	if err := deps.start(target); err != nil {
		return fmt.Errorf("launch hamd: %w", err)
	}

	deadline := deps.now().Add(daemonBootstrapTimeout)
	var lastErr error
	for {
		lastErr = checkDaemonReachable(socketPath, deps.dial)
		if lastErr == nil {
			return nil
		}
		if !deps.now().Before(deadline) {
			break
		}
		deps.sleep(daemonBootstrapPollInterval)
	}

	return fmt.Errorf("timed out waiting for daemon socket %s after auto-bootstrap: %w", socketPath, lastErr)
}

func checkDaemonReachable(socketPath string, dial func(string, time.Duration) (net.Conn, error)) error {
	conn, err := dial(socketPath, 200*time.Millisecond)
	if err != nil {
		return err
	}
	return conn.Close()
}

func resolveDaemonLaunchTarget(
	executablePath func() (string, error),
	lookupEnv func(string) (string, bool),
	getwd func() (string, error),
	lookPath func(string) (string, error),
) (detachedLaunchTarget, error) {
	for _, key := range []string{"HAM_DAEMON_EXECUTABLE", "HAMD_EXECUTABLE"} {
		if override, ok := lookupEnv(key); ok && strings.TrimSpace(override) != "" {
			return detachedLaunchTarget{Executable: strings.TrimSpace(override), Args: []string{"serve"}}, nil
		}
	}

	if currentExecutable, err := executablePath(); err == nil {
		sibling := filepath.Join(filepath.Dir(currentExecutable), "hamd")
		if fileExists(sibling) {
			return detachedLaunchTarget{Executable: sibling, Args: []string{"serve"}}, nil
		}
	}

	if found, err := lookPath("hamd"); err == nil {
		return detachedLaunchTarget{Executable: found, Args: []string{"serve"}}, nil
	}

	if workingDirectory, err := getwd(); err == nil {
		if repoRoot, ok := findRepoRoot(workingDirectory); ok {
			goBinary, lookErr := lookPath("go")
			if lookErr != nil {
				return detachedLaunchTarget{}, fmt.Errorf("resolve go for hamd fallback: %w", lookErr)
			}
			return detachedLaunchTarget{
				Executable: goBinary,
				Args:       []string{"run", "./go/cmd/hamd", "serve"},
				Dir:        repoRoot,
			}, nil
		}
	}

	return detachedLaunchTarget{}, fmt.Errorf("hamd executable could not be resolved")
}

func findRepoRoot(start string) (string, bool) {
	current := filepath.Clean(start)
	for {
		if fileExists(filepath.Join(current, "go.mod")) && fileExists(filepath.Join(current, "go", "cmd", "hamd", "main.go")) {
			return current, true
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}
		current = parent
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func startDetachedProcess(target detachedLaunchTarget) error {
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("open %s: %w", os.DevNull, err)
	}
	defer devNull.Close()

	cmd := exec.Command(target.Executable, target.Args...)
	cmd.Stdin = devNull
	cmd.Stdout = devNull
	cmd.Stderr = devNull
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if target.Dir != "" {
		cmd.Dir = target.Dir
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func commandRequiresDaemon(command string) bool {
	switch command {
	case "run", "attach", "observe", "open", "ask", "stop", "detach", "logs", "team", "settings", "list", "status", "events":
		return true
	default:
		return false
	}
}
