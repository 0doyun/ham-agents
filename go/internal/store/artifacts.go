package store

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// ArtifactStore persists large event artifact blobs outside the event log.
type ArtifactStore interface {
	Save(agentID, eventID string, data []byte) (ref string, err error)
	Load(ref string) ([]byte, error)
	// Prune removes oldest files (by mtime) until total size <= maxBytesTotal.
	Prune(maxBytesTotal int64) error
}

// FileArtifactStore stores artifacts as raw files under a root directory.
// Layout: {root}/{agentID}/{eventID}.bin
type FileArtifactStore struct {
	root string
}

// NewFileArtifactStore creates a FileArtifactStore rooted at root.
// root is typically {HAM_AGENTS_HOME}/artifacts or
// ~/Library/Application Support/ham-agents/artifacts.
func NewFileArtifactStore(root string) *FileArtifactStore {
	return &FileArtifactStore{root: root}
}

// DefaultArtifactStorePath returns the default artifact root directory,
// mirroring the same logic as DefaultEventLogPath.
func DefaultArtifactStorePath() (string, error) {
	if root := os.Getenv("HAM_AGENTS_HOME"); root != "" {
		return filepath.Join(root, "artifacts"), nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}
	return filepath.Join(homeDir, "Library", "Application Support", "ham-agents", "artifacts"), nil
}

// Save writes data to {root}/{agentID}/{eventID}.bin using an atomic
// write (tmp file + rename). Returns the absolute path as the ref.
func (s *FileArtifactStore) Save(agentID, eventID string, data []byte) (string, error) {
	dir := filepath.Join(s.root, agentID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create artifact dir: %w", err)
	}

	dest := filepath.Join(dir, eventID+".bin")
	tmp := dest + ".tmp"

	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return "", fmt.Errorf("write artifact tmp: %w", err)
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("rename artifact: %w", err)
	}
	return dest, nil
}

// Load reads and returns the artifact at the given ref (absolute path).
func (s *FileArtifactStore) Load(ref string) ([]byte, error) {
	data, err := os.ReadFile(ref)
	if err != nil {
		return nil, fmt.Errorf("load artifact %q: %w", ref, err)
	}
	return data, nil
}

// Prune removes the oldest artifact files (by mtime) until the total
// size of all artifacts is <= maxBytesTotal. Directories are not removed.
func (s *FileArtifactStore) Prune(maxBytesTotal int64) error {
	type fileEntry struct {
		path  string
		size  int64
		mtime int64 // UnixNano
	}

	var entries []fileEntry
	var totalSize int64

	err := filepath.WalkDir(s.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		entries = append(entries, fileEntry{
			path:  path,
			size:  info.Size(),
			mtime: info.ModTime().UnixNano(),
		})
		totalSize += info.Size()
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk artifact store: %w", err)
	}

	if totalSize <= maxBytesTotal {
		return nil
	}

	// Sort oldest first (ascending mtime).
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].mtime < entries[j].mtime
	})

	for _, e := range entries {
		if totalSize <= maxBytesTotal {
			break
		}
		if err := os.Remove(e.path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove artifact %q: %w", e.path, err)
		}
		totalSize -= e.size
	}
	return nil
}
