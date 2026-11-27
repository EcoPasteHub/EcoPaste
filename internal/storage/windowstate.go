package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/renameio/v2"
)

const windowStateName = "panel-frame.json"

// WindowFrame is the clipboard panel's frame in AppKit screen points.
type WindowFrame struct {
	X      int64 `json:"x"`
	Y      int64 `json:"y"`
	Width  int64 `json:"width"`
	Height int64 `json:"height"`
}

// WindowState stores the clipboard panel's frame between launches.
type WindowState struct {
	path  string
	mu    sync.Mutex
	frame *WindowFrame
}

// OpenWindowState reads the saved panel frame, if present.
func OpenWindowState(stateDir string) (*WindowState, error) {
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return nil, err
	}
	state := &WindowState{
		path: filepath.Join(stateDir, windowStateName),
	}
	data, err := os.ReadFile(state.path)
	if errors.Is(err, os.ErrNotExist) {
		return state, nil
	}
	if err != nil {
		return nil, err
	}
	var frame WindowFrame
	if err := json.Unmarshal(data, &frame); err == nil && frame.Width > 0 && frame.Height > 0 {
		state.frame = &frame
	}
	return state, nil
}

// Frame returns a window's saved frame.
func (s *WindowState) Frame() (WindowFrame, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.frame == nil {
		return WindowFrame{}, false
	}
	return *s.frame, true
}

// SaveFrame records a changed panel frame atomically.
func (s *WindowState) SaveFrame(frame WindowFrame) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.frame != nil && *s.frame == frame {
		return nil
	}
	data, err := json.MarshalIndent(frame, "", "  ")
	if err != nil {
		return err
	}
	if err := renameio.WriteFile(s.path, data, 0o644); err != nil {
		return err
	}
	s.frame = &frame
	return nil
}
