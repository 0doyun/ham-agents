package ipc

import (
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	SocketPath string
}

func DefaultSocketPath() (string, error) {
	if path := os.Getenv("HAM_AGENTS_SOCKET"); path != "" {
		return path, nil
	}

	if root := os.Getenv("HAM_AGENTS_HOME"); root != "" {
		return filepath.Join(root, "hamd.sock"), nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}

	return filepath.Join(homeDir, "Library", "Application Support", "ham-agents", "hamd.sock"), nil
}

func DefaultConfig() (Config, error) {
	socketPath, err := DefaultSocketPath()
	if err != nil {
		return Config{}, err
	}

	return Config{SocketPath: socketPath}, nil
}
