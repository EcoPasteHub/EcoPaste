package storage

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"sync"
)

const windowStateName = "window-state.json"

// WindowFrame is one window's frame in screen points. The file keeps the
// published map of labels to frames so older state files stay readable.
type WindowFrame struct {
	X      int64 `json:"x"`
	Y      int64 `json:"y"`
	Width  int64 `json:"width"`
	Height int64 `json:"height"`
}

// WindowState stores window frames between launches. Writes replace the
// file atomically so a crash cannot leave a truncated map behind.
type WindowState struct {
	path   string
	mu     sync.Mutex
	frames map[string]WindowFrame
}

// OpenWindowState reads the frames under stateDir, dropping the whole
// map when the file is unreadable and starting empty when it is missing.
func OpenWindowState(stateDir string) (*WindowState, error) {
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return nil, err
	}
	state := &WindowState{
		path:   filepath.Join(stateDir, windowStateName),
		frames: map[string]WindowFrame{},
	}
	data, err := os.ReadFile(state.path)
	if errors.Is(err, os.ErrNotExist) {
		return state, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &state.frames); err != nil {
		state.frames = map[string]WindowFrame{}
	}
	return state, nil
}

// Frame returns a window's saved frame.
func (s *WindowState) Frame(label string) (WindowFrame, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	frame, ok := s.frames[label]
	return frame, ok
}

// SaveFrame records a window's frame and writes the map.
func (s *WindowState) SaveFrame(label string, frame WindowFrame) error {
	frame.X = int64(math.Round(float64(frame.X)))
	frame.Y = int64(math.Round(float64(frame.Y)))
	frame.Width = int64(math.Round(float64(frame.Width)))
	frame.Height = int64(math.Round(float64(frame.Height)))
	s.mu.Lock()
	defer s.mu.Unlock()
	next := make(map[string]WindowFrame, len(s.frames)+1)
	for key, value := range s.frames {
		next[key] = value
	}
	next[label] = frame
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		os.Remove(tmp)
		return err
	}
	s.frames = next
	return nil
}
