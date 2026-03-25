package adapters

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

type TranscriptSource struct {
	Path        string
	DisplayName string
}

type TranscriptAdapter struct{}

func NewTranscriptAdapter() TranscriptAdapter {
	return TranscriptAdapter{}
}

func (TranscriptAdapter) Discover(dirs []string) ([]TranscriptSource, error) {
	seen := map[string]struct{}{}
	sources := []TranscriptSource{}

	for _, dir := range dirs {
		trimmed := strings.TrimSpace(dir)
		if trimmed == "" {
			continue
		}
		err := filepath.WalkDir(trimmed, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".log" && ext != ".txt" && ext != ".jsonl" && ext != ".md" {
				return nil
			}
			if _, ok := seen[path]; ok {
				return nil
			}
			seen[path] = struct{}{}
			sources = append(sources, TranscriptSource{
				Path:        path,
				DisplayName: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
			})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	sort.SliceStable(sources, func(i, j int) bool { return sources[i].Path < sources[j].Path })
	return sources, nil
}
