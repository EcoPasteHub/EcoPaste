package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
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

func TestOpenFallsBackToDefaultsForUnreadableSettings(t *testing.T) {
	cases := map[string]string{
		"empty":           "",
		"truncated":       `{"general":`,
		"wrong type":      `{"general":{"autoStart":"yes"}}`,
		"unknown enum":    `{"appearance":{"theme":"sepia"}}`,
		"invalid preview": `{"clipboard":{"preview":{"hoverDelayMs":200}}}`,
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), FileName)
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			store, err := Open(path)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := store.Snapshot(), Default(); !reflect.DeepEqual(got, want) {
				t.Errorf("settings = %+v, want defaults", got)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != content {
				t.Error("unreadable settings file was overwritten during startup")
			}
		})
	}
}

func TestSettingsJSONRoundTripsPublishedCamelCase(t *testing.T) {
	want := Default()
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		`"autoStart"`,
		`"maxTextMb"`,
		`"showOriginalPreview"`,
		`"itemActionOrder"`,
		`"deleteFavoriteItemsOnlyInFavoriteGroup"`,
		`"lastCheckedAt"`,
		`"importedAt"`,
	} {
		if !strings.Contains(string(data), key) {
			t.Errorf("encoded settings missing %s", key)
		}
	}
	var got Settings
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round trip = %+v, want %+v", got, want)
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

func TestUpdateRejectsInvalidValuesWithoutChangingSettings(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), FileName))
	if err != nil {
		t.Fatal(err)
	}
	before := store.Snapshot()
	path := store.Path()
	beforeData, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	patches := []map[string]any{
		{"appearance": map[string]any{"theme": "sepia"}},
		{"clipboard": map[string]any{"capture": map[string]any{
			"order": []any{"text", "unknown"},
		}}},
		{"clipboard": map[string]any{"content": map[string]any{
			"itemActions": []any{"copy", "unknown"},
		}}},
		{"clipboard": map[string]any{"preview": map[string]any{"hoverDelayMs": 200}}},
		{"clipboard": map[string]any{"window": map[string]any{"selectGroupOnOpen": "invalid"}}},
		{"update": map[string]any{"frequency": "hourly"}},
	}
	for _, patch := range patches {
		if _, err := store.Update(patch); err == nil {
			t.Errorf("invalid patch accepted: %v", patch)
		}
		if got := store.Snapshot(); !reflect.DeepEqual(got, before) {
			t.Errorf("settings changed after rejected patch %v", patch)
		}
	}
	afterData, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(afterData) != string(beforeData) {
		t.Error("settings file changed after rejected patches")
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

func TestSnapshotAndUpdateResultsDoNotAliasStore(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), FileName))
	if err != nil {
		t.Fatal(err)
	}
	snapshot := store.Snapshot()
	snapshot.Clipboard.Capture.Order[0] = CaptureText
	snapshot.Clipboard.Content.ItemActions[0] = ActionDelete
	snapshot.Clipboard.Filters.ExcludedAppIDs[0] = "changed"
	if got := store.Snapshot().Clipboard.Capture.Order[0]; got != CaptureFiles {
		t.Errorf("capture order aliased store: %q", got)
	}
	if got := store.Snapshot().Clipboard.Content.ItemActions[0]; got != ActionCopy {
		t.Errorf("item actions aliased store: %q", got)
	}
	if got := store.Snapshot().Clipboard.Filters.ExcludedAppIDs[0]; got != "com.apple.keychainaccess" {
		t.Errorf("excluded apps aliased store: %q", got)
	}

	updated, err := store.Update(map[string]any{"appearance": map[string]any{"theme": "dark"}})
	if err != nil {
		t.Fatal(err)
	}
	updated.Clipboard.Capture.Order[0] = CaptureHTML
	if got := store.Snapshot().Clipboard.Capture.Order[0]; got != CaptureFiles {
		t.Errorf("update result aliased store: %q", got)
	}
}

func TestConcurrentUpdatesKeepCompleteSettings(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), FileName))
	if err != nil {
		t.Fatal(err)
	}
	patches := []map[string]any{
		{"general": map[string]any{"autoStart": true}},
		{"general": map[string]any{"runAsAdmin": true}},
		{"general": map[string]any{"trayIcon": false}},
		{"general": map[string]any{"dockIcon": true}},
		{"clipboard": map[string]any{"content": map[string]any{"copyPlain": true}}},
		{"clipboard": map[string]any{"content": map[string]any{"pastePlain": true}}},
		{"clipboard": map[string]any{"content": map[string]any{"autoFavorite": true}}},
		{"clipboard": map[string]any{"feedback": map[string]any{"copySound": true}}},
	}
	var wg sync.WaitGroup
	errs := make(chan error, len(patches))
	for _, patch := range patches {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := store.Update(patch)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	got := store.Snapshot()
	if !got.General.AutoStart || !got.General.RunAsAdmin || got.General.TrayIcon || !got.General.DockIcon ||
		!got.Clipboard.Content.CopyPlain || !got.Clipboard.Content.PastePlain ||
		!got.Clipboard.Content.AutoFavorite || !got.Clipboard.Feedback.CopySound {
		t.Errorf("concurrent updates lost fields: %+v", got)
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

func TestCrashResidueDoesNotAffectSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(map[string]any{"appearance": map[string]any{"theme": "dark"}}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".tmp", []byte(`{"appearance":`), 0o644); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.Snapshot().Appearance.Theme; got != ThemeDark {
		t.Errorf("theme after crash residue = %q", got)
	}
	if _, err := reopened.Update(map[string]any{"general": map[string]any{"autoStart": true}}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Errorf("temporary file left behind after recovery: %v", err)
	}
}
