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
	if _, ok := state.Frame("clipboard"); ok {
		t.Fatal("new state has a frame")
	}
	frame := WindowFrame{X: 120, Y: 240, Width: 420, Height: 560}
	if err := state.SaveFrame("clipboard", frame); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenWindowState(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := reopened.Frame("clipboard")
	if !ok || got != frame {
		t.Fatalf("frame = %+v, %v, want %+v", got, ok, frame)
	}
}

func TestWindowStateKeepsOtherWindows(t *testing.T) {
	dir := t.TempDir()
	legacy := `{"settings":{"x":10,"y":20,"width":560,"height":600}}`
	if err := os.WriteFile(filepath.Join(dir, windowStateName), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	state, err := OpenWindowState(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := state.SaveFrame("clipboard", WindowFrame{X: 1, Y: 2, Width: 3, Height: 4}); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenWindowState(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := reopened.Frame("settings"); !ok {
		t.Fatal("settings frame dropped")
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
	if _, ok := state.Frame("clipboard"); ok {
		t.Fatal("broken file kept a frame")
	}
}
