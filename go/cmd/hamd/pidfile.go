package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

// acquirePIDFileLock creates a PID file next to the socket and holds an
// exclusive advisory lock (flock) on it. If another daemon already holds
// the lock, an error is returned. The returned *os.File must be kept open
// for the lifetime of the process — closing it releases the lock.
func acquirePIDFileLock(socketPath string) (*os.File, error) {
	pidPath := pidPathForSocket(socketPath)

	if err := os.MkdirAll(filepath.Dir(pidPath), 0o755); err != nil {
		return nil, fmt.Errorf("create pid directory: %w", err)
	}

	file, err := os.OpenFile(pidPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open pid file: %w", err)
	}

	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close()
		return nil, fmt.Errorf("another hamd instance is already running (pid file: %s)", pidPath)
	}

	_ = file.Truncate(0)
	_, _ = file.Seek(0, 0)
	_, _ = fmt.Fprintf(file, "%d\n", os.Getpid())
	_ = file.Sync()

	return file, nil
}

func pidPathForSocket(socketPath string) string {
	return socketPath + ".pid"
}

func removePIDFile(socketPath string) {
	_ = os.Remove(pidPathForSocket(socketPath))
}

// isProcessAlive checks if a process with the given PID is still running.
func isProcessAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// cleanStalePIDFile removes a PID file if the recorded process is no longer alive.
func cleanStalePIDFile(socketPath string) {
	pidPath := pidPathForSocket(socketPath)
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(string(data[:len(data)-1]))
	if err != nil {
		_ = os.Remove(pidPath)
		return
	}
	if !isProcessAlive(pid) {
		_ = os.Remove(pidPath)
	}
}
