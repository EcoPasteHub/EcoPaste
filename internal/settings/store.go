package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const FileName = "settings.json"

// Store keeps the current settings in memory and writes every change
// atomically to config/settings.json.
type Store struct {
	mu      sync.RWMutex
	path    string
	current Settings
}

// Open loads the settings file, creating it with defaults on first run.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create settings dir: %w", err)
	}
	current, exists, err := load(path)
	if err != nil {
		return nil, err
	}
	s := &Store{path: path, current: current}
	if !exists {
		if err := s.write(current); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// Path returns the settings file's location.
func (s *Store) Path() string { return s.path }

// Snapshot returns a copy of the current settings.
func (s *Store) Snapshot() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current
}

// Update deep-merges a JSON object patch into the current settings,
// validates the result and persists it. Objects merge recursively;
// arrays and scalars replace the previous value.
func (s *Store) Update(patch map[string]any) (Settings, error) {
	if patch == nil {
		return Settings{}, errors.New("settings patch must be a JSON object")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	merged, err := merge(s.current, patch)
	if err != nil {
		return Settings{}, err
	}
	if err := validate(merged); err != nil {
		return Settings{}, err
	}
	if err := s.write(merged); err != nil {
		return Settings{}, err
	}
	s.current = merged
	return merged, nil
}

// Replace swaps the whole settings for next, validating and persisting
// it. It is used by backup imports.
func (s *Store) Replace(next Settings) error {
	if err := validate(next); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.write(next); err != nil {
		return err
	}
	s.current = next
	return nil
}

// Reset restores the published defaults.
func (s *Store) Reset() (Settings, error) {
	next := Default()
	if err := s.Replace(next); err != nil {
		return Settings{}, err
	}
	return next, nil
}

func (s *Store) write(next Settings) error {
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	tmp := s.path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("create settings temp file: %w", err)
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return fmt.Errorf("write settings temp file: %w", err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return fmt.Errorf("sync settings temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close settings temp file: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("replace settings file: %w", err)
	}
	return nil
}

func load(path string) (Settings, bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Default(), false, nil
	}
	if err != nil {
		return Settings{}, false, fmt.Errorf("read settings: %w", err)
	}
	next := Default()
	if err := json.Unmarshal(data, &next); err != nil {
		return Settings{}, true, fmt.Errorf("parse settings: %w", err)
	}
	return next, true, nil
}

func merge(current Settings, patch map[string]any) (Settings, error) {
	data, err := json.Marshal(current)
	if err != nil {
		return Settings{}, fmt.Errorf("encode current settings: %w", err)
	}
	var base map[string]any
	if err := json.Unmarshal(data, &base); err != nil {
		return Settings{}, fmt.Errorf("decode current settings: %w", err)
	}
	deepMerge(base, patch)

	merged, err := json.Marshal(base)
	if err != nil {
		return Settings{}, fmt.Errorf("encode merged settings: %w", err)
	}
	next := Default()
	if err := json.Unmarshal(merged, &next); err != nil {
		return Settings{}, fmt.Errorf("invalid settings patch: %w", err)
	}
	return next, nil
}

func deepMerge(base, patch map[string]any) {
	for key, value := range patch {
		if existing, ok := base[key]; ok {
			if existingMap, ok := existing.(map[string]any); ok {
				if patchMap, ok := value.(map[string]any); ok {
					deepMerge(existingMap, patchMap)
					continue
				}
			}
		}
		base[key] = value
	}
}

func validate(s Settings) error {
	if value := s.Clipboard.Window.SelectGroupOnOpen; value != "preserve" && value != "all" {
		if !strings.HasPrefix(value, "group:") || strings.TrimSpace(strings.TrimPrefix(value, "group:")) == "" {
			return errors.New("open group selection is invalid")
		}
	}
	clipboard := normalizeShortcut(s.Shortcuts.OpenClipboard)
	preference := normalizeShortcut(s.Shortcuts.OpenPreference)
	if clipboard != "" && preference != "" && clipboard == preference {
		return errors.New("global shortcuts must be unique")
	}
	return nil
}

func normalizeShortcut(value string) string {
	parts := strings.Split(value, "+")
	keys := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			keys = append(keys, strings.ToLower(part))
		}
	}
	return strings.Join(keys, "+")
}
