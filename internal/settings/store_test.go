package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenCreatesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config", FileName)
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got := store.Snapshot()
	if got.Shortcuts.OpenClipboard != "Alt+C" {
		t.Errorf("open clipboard = %q", got.Shortcuts.OpenClipboard)
	}
	if got.Clipboard.Capture.MaxTextMB != 4 {
		t.Errorf("max text = %d", got.Clipboard.Capture.MaxTextMB)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("settings file missing: %v", err)
	}
}

func TestLoadKeepsDefaultsForMissingFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)
	if err := os.WriteFile(path, []byte(`{"general":{"autoStart":true}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got := store.Snapshot()
	if !got.General.AutoStart {
		t.Error("autoStart not loaded")
	}
	if !got.General.TrayIcon {
		t.Error("trayIcon default lost")
	}
	if got.Update.Frequency != UpdateDaily {
		t.Errorf("update frequency = %q", got.Update.Frequency)
	}
}

func TestUpdateDeepMergesObjectsAndReplacesArrays(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), FileName))
	if err != nil {
		t.Fatal(err)
	}
	next, err := store.Update(map[string]any{
		"general": map[string]any{"autoStart": true},
		"clipboard": map[string]any{
			"content": map[string]any{"itemActions": []any{"copy", "pastePlain"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !next.General.AutoStart || !next.General.TrayIcon {
		t.Errorf("general = %+v", next.General)
	}
	if len(next.Clipboard.Content.ItemActions) != 2 || next.Clipboard.Content.ItemActions[1] != ActionPastePlain {
		t.Errorf("item actions = %v", next.Clipboard.Content.ItemActions)
	}
}

func TestUpdateRejectsDuplicateShortcuts(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), FileName))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(map[string]any{"shortcuts": map[string]any{"openPreference": " alt+c "}}); err == nil {
		t.Fatal("expected an error")
	}
}

func TestWriteIsAtomicAndReadable(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(map[string]any{"appearance": map[string]any{"theme": "dark"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Errorf("temporary file left behind: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var parsed Settings
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Appearance.Theme != ThemeDark {
		t.Errorf("theme = %q", parsed.Appearance.Theme)
	}
}
