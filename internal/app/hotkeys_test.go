package app

import (
	"path/filepath"
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/settings"
)

func TestHotKeySetHonorsClearedShortcuts(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	keys := newHotKeySet(nil, nil, store)
	if keys.openClipboard != "alt+c" || keys.openPreference != "alt+x" {
		t.Fatalf("defaults = %q, %q", keys.openClipboard, keys.openPreference)
	}

	if _, err := store.Update(map[string]any{
		"shortcuts": map[string]any{"openClipboard": ""},
	}); err != nil {
		t.Fatal(err)
	}
	keys.applyStored()
	if keys.openClipboard != "" || keys.openPreference != "alt+x" {
		t.Fatalf("after clearing clipboard = %q, %q", keys.openClipboard, keys.openPreference)
	}

	if _, err := store.Update(map[string]any{
		"shortcuts": map[string]any{"openPreference": ""},
	}); err != nil {
		t.Fatal(err)
	}
	keys.applyStored()
	if keys.openClipboard != "" || keys.openPreference != "" {
		t.Fatalf("after clearing both = %q, %q", keys.openClipboard, keys.openPreference)
	}

	if _, err := store.Update(map[string]any{
		"shortcuts": map[string]any{"openPreference": "Alt+C"},
	}); err != nil {
		t.Fatal(err)
	}
	keys.applyStored()
	if keys.openClipboard != "" || keys.openPreference != "alt+c" {
		t.Fatalf("after reassigning = %q, %q", keys.openClipboard, keys.openPreference)
	}
}
