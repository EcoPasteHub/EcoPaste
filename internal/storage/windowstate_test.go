package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWindowStateRoundTrip(t *testing.T) {
	dir := t.TempDir()
	state, err := OpenWindowState(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := state.Frame(); ok {
		t.Fatal("new state has a frame")
	}
	frame := WindowFrame{X: 120, Y: 240, Width: 420, Height: 560}
	if err := state.SaveFrame(frame); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenWindowState(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := reopened.Frame()
	if !ok || got != frame {
		t.Fatalf("frame = %+v, %v, want %+v", got, ok, frame)
	}
}

func TestWindowStateDropsBrokenFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, windowStateName), []byte("{broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	state, err := OpenWindowState(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := state.Frame(); ok {
		t.Fatal("broken file kept a frame")
	}
}

func TestSavingSameWindowFrameKeepsPersistedFile(t *testing.T) {
	state, err := OpenWindowState(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	frame := WindowFrame{X: 120, Y: 240, Width: 420, Height: 560}
	if err := state.SaveFrame(frame); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(state.path)
	if err != nil {
		t.Fatal(err)
	}
	if err := state.SaveFrame(frame); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(state.path)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) {
		t.Error("saving the same frame replaced the state file")
	}
}
