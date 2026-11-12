package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultsMatchPersistedContract(t *testing.T) {
	value := Default()
	if value.Update.AutoCheck {
		t.Fatal("autoCheck must default to false")
	}
	if value.Update.Frequency != "daily" {
		t.Fatalf("frequency = %q", value.Update.Frequency)
	}
	if got := value.Clipboard.Capture.Order; len(got) != 5 || got[0] != "files" || got[4] != "text" {
		t.Fatalf("capture order = %v", got)
	}
	if value.Clipboard.Content.AutoPaste != "doubleClickPaste" {
		t.Fatalf("autoPaste = %q", value.Clipboard.Content.AutoPaste)
	}
}

func TestUpdateDeepMergesAndReplacesArrays(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	next, err := store.Update(map[string]any{
		"general": map[string]any{"autoStart": true},
		"clipboard": map[string]any{
			"content": map[string]any{"itemActions": []any{"paste", "copy"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !next.General.AutoStart || !next.General.TrayIcon {
		t.Fatalf("general merge lost defaults: %+v", next.General)
	}
	if len(next.Clipboard.Content.ItemActions) != 2 {
		t.Fatalf("arrays must be replaced: %v", next.Clipboard.Content.ItemActions)
	}
}

func TestCorruptSettingsFallBackWithoutOverwriting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte("{broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if store.Snapshot().Shortcuts.OpenClipboard != "Alt+C" {
		t.Fatal("expected defaults")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "{broken" {
		t.Fatal("corrupt file must not be overwritten on load")
	}
}

func TestUpdateRejectsDuplicateShortcuts(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Update(map[string]any{
		"shortcuts": map[string]any{"openPreference": "Alt+C"},
	})
	if err == nil {
		t.Fatal("expected duplicate shortcut error")
	}
}

func TestSavedJSONUsesCamelCase(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(store.path)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	clipboard := document["clipboard"].(map[string]any)
	if _, ok := clipboard["capture"]; !ok {
		t.Fatal("missing clipboard.capture")
	}
}

func TestRealSettingsCompatibility(t *testing.T) {
	path := os.Getenv("ECOPASTE_TEST_SETTINGS")
	if path == "" {
		t.Skip("set ECOPASTE_TEST_SETTINGS to a real settings.json")
	}
	dir := filepath.Dir(path)
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	value := store.Snapshot()
	if value.Appearance.Language == "" || value.Shortcuts.OpenClipboard == "" {
		t.Fatalf("settings did not parse: %+v", value)
	}
	if len(value.Clipboard.Capture.Order) != 5 {
		t.Fatalf("capture order = %v", value.Clipboard.Capture.Order)
	}
}
