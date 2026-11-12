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

const filename = "settings.json"

// Store owns the in-memory settings snapshot and its atomic on-disk copy.
type Store struct {
	mu      sync.RWMutex
	path    string
	current Settings
}

// Open loads settings from configDir, creating defaults on first run.
func Open(configDir string) (*Store, error) {
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return nil, fmt.Errorf("create settings directory: %w", err)
	}
	path := filepath.Join(configDir, filename)
	current, firstRun, err := load(path)
	if err != nil {
		return nil, err
	}
	store := &Store{path: path, current: current}
	if firstRun {
		if err := store.write(current); err != nil {
			return nil, err
		}
	}
	return store, nil
}

// Snapshot returns a deep copy of the current settings.
func (s *Store) Snapshot() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return clone(s.current)
}

// Update applies a JSON object patch and persists the merged result.
func (s *Store) Update(patch map[string]any) (Settings, error) {
	if patch == nil {
		return Settings{}, fmt.Errorf("settings patch must be a JSON object")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	base, err := toMap(s.current)
	if err != nil {
		return Settings{}, err
	}
	deepMerge(base, patch)

	var next Settings
	raw, err := json.Marshal(base)
	if err != nil {
		return Settings{}, err
	}
	next = Default()
	if err := json.Unmarshal(raw, &next); err != nil {
		return Settings{}, fmt.Errorf("invalid settings patch: %w", err)
	}
	if err := validate(next); err != nil {
		return Settings{}, err
	}
	if err := s.write(next); err != nil {
		return Settings{}, err
	}
	s.current = next
	return clone(next), nil
}

// Replace validates and installs a full settings document.
func (s *Store) Replace(next Settings) (Settings, error) {
	if err := validate(next); err != nil {
		return Settings{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.write(next); err != nil {
		return Settings{}, err
	}
	s.current = clone(next)
	return clone(next), nil
}

// Reset restores defaults and persists them.
func (s *Store) Reset() (Settings, error) {
	next := Default()
	if err := validate(next); err != nil {
		return Settings{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.write(next); err != nil {
		return Settings{}, err
	}
	s.current = next
	return clone(next), nil
}

func load(path string) (Settings, bool, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Default(), true, nil
	}
	if err != nil {
		return Settings{}, false, fmt.Errorf("read settings: %w", err)
	}

	next := Default()
	if err := json.Unmarshal(raw, &next); err != nil {
		// The Rust store treats a corrupt file as defaults, not first run.
		return Default(), false, nil
	}
	return next, false, nil
}

func (s *Store) write(value Settings) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("serialize settings: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return fmt.Errorf("write settings temporary file: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("replace settings file: %w", err)
	}
	return nil
}

func validate(value Settings) error {
	if err := validateOpenGroup(value.Clipboard.Window.SelectGroupOnOpen); err != nil {
		return err
	}
	left := normalizeShortcut(value.Shortcuts.OpenClipboard)
	right := normalizeShortcut(value.Shortcuts.OpenPreference)
	if left != "" && left == right {
		return fmt.Errorf("global shortcuts must be unique")
	}
	return nil
}

func validateOpenGroup(value string) error {
	if value == WindowOpenSelectionPreserve || value == WindowOpenSelectionAll {
		return nil
	}
	id, ok := strings.CutPrefix(value, WindowOpenGroupPrefix)
	if !ok || strings.TrimSpace(id) == "" {
		return fmt.Errorf("open group selection is invalid")
	}
	return nil
}

func normalizeShortcut(value string) string {
	parts := strings.Split(value, "+")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.ToLower(strings.TrimSpace(part))
		if part != "" {
			out = append(out, part)
		}
	}
	return strings.Join(out, "+")
}

func toMap(value Settings) (map[string]any, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func deepMerge(base, patch map[string]any) {
	for key, value := range patch {
		if patchObject, ok := value.(map[string]any); ok {
			if baseObject, ok := base[key].(map[string]any); ok {
				deepMerge(baseObject, patchObject)
				continue
			}
		}
		base[key] = value
	}
}

func clone(value Settings) Settings {
	raw, _ := json.Marshal(value)
	var out Settings
	_ = json.Unmarshal(raw, &out)
	return out
}
