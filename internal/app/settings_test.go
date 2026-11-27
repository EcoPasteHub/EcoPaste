package app

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
)

func TestSettingsModelAppliesPaths(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	model := newSettingsModel(store)
	if err := model.Apply("copyPlain", true); err != nil {
		t.Fatal(err)
	}
	if err := model.Apply("capture.maxTextMb", uint32(8)); err != nil {
		t.Fatal(err)
	}
	if err := model.Apply("history.retention", "7d"); err != nil {
		t.Fatal(err)
	}
	if err := model.Apply("sensitive.redactSecrets", false); err != nil {
		t.Fatal(err)
	}
	if err := model.Apply("window.position", "remember"); err != nil {
		t.Fatal(err)
	}
	if err := model.Apply("feedback.copySound", true); err != nil {
		t.Fatal(err)
	}
	if err := model.Apply("content.autoPaste", "singleClickCopy"); err != nil {
		t.Fatal(err)
	}
	if err := model.Apply("content.autoFavorite", true); err != nil {
		t.Fatal(err)
	}
	if err := model.Apply("content.showOriginalPreview", false); err != nil {
		t.Fatal(err)
	}
	if err := model.Apply("content.deleteNormal", "direct"); err != nil {
		t.Fatal(err)
	}
	if err := model.Apply("content.deleteFavorite", "confirm"); err != nil {
		t.Fatal(err)
	}
	if err := model.Apply("search.clearOnHide", false); err != nil {
		t.Fatal(err)
	}

	values := model.Load()
	if !values.CopyPlain || values.MaxTextMB != 8 || values.Retention != "7d" || values.RedactSecrets ||
		values.WindowPosition != "remember" || !values.CopySound ||
		values.AutoPaste != "singleClickCopy" ||
		!values.AutoFavorite || values.ShowOriginalPreview ||
		values.DeleteNormal != "direct" || values.DeleteFavorite != "confirm" ||
		values.SearchClear {
		t.Errorf("values = %+v", values)
	}
	cfg := store.Snapshot().Clipboard
	if cfg.History.Retention.Value != 7 || cfg.History.Retention.Unit != settings.RetentionDays {
		t.Errorf("retention = %+v", cfg.History.Retention)
	}
	if cfg.Window.Position != settings.PositionRemember {
		t.Errorf("window position = %q", cfg.Window.Position)
	}
}

func TestSettingsModelAppliesAndOpensLogs(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	model := newSettingsModel(store)
	var applied settings.Logging
	model.onLoggingChanged = func(config settings.Logging) error {
		applied = config
		return nil
	}
	opened := false
	model.openLogs = func() error {
		opened = true
		return nil
	}
	if err := model.Apply("logging.level", "debug"); err != nil {
		t.Fatal(err)
	}
	if err := model.Apply("logging.level", "all"); err != nil {
		t.Fatal(err)
	}
	if applied.Level != settings.LogLevelAll {
		t.Fatalf("applied logging settings = %+v", applied)
	}
	if err := model.Apply("logs.open", true); err != nil {
		t.Fatal(err)
	}
	if !opened {
		t.Fatal("open log directory action was not invoked")
	}
	if got := model.Load(); got.LogLevel != "all" {
		t.Fatalf("view logging values = %+v", got)
	}
}

func TestSettingsModelSignalsListRefresh(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	model := newSettingsModel(store)
	changed := 0
	model.onChanged = func() { changed++ }
	for _, change := range []struct {
		path  string
		value any
	}{
		{"sort", "useCountDesc"},
		{"sensitive.redactSecrets", false},
		{"content.showOriginalPreview", true},
		{"content.deleteNormal", "direct"},
		{"preview.spaceEnabled", false},
	} {
		if err := model.Apply(change.path, change.value); err != nil {
			t.Fatal(err)
		}
	}
	if changed != 5 {
		t.Errorf("refresh signals = %d, want 5", changed)
	}
}

func TestLoadFallsBackWhenOpenGroupWasDeleted(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	if err := newSettingsModel(store).Apply("window.selectGroupOnOpen", "group:gone"); err != nil {
		t.Fatal(err)
	}
	model := newSettingsModel(store)
	model.listGroups = func() ([]database.Group, error) {
		return []database.Group{{ID: "kept", Name: "Kept"}}, nil
	}
	if got := model.Load().GroupOnOpen; got != "preserve" {
		t.Errorf("open group = %q, want preserve", got)
	}
	model.listGroups = func() ([]database.Group, error) {
		return []database.Group{{ID: "gone", Name: "Gone"}}, nil
	}
	if got := model.Load().GroupOnOpen; got != "group:gone" {
		t.Errorf("open group = %q, want the existing group", got)
	}
}

func TestRetentionEncoding(t *testing.T) {
	cases := map[settings.Retention]string{
		{Unit: settings.RetentionForever}:          "forever",
		{Value: 2, Unit: settings.RetentionHours}:  "2h",
		{Value: 3, Unit: settings.RetentionDays}:   "3d",
		{Value: 2, Unit: settings.RetentionWeeks}:  "2w",
		{Value: 1, Unit: settings.RetentionMonths}: "1mo",
	}
	for retention, want := range cases {
		if got := encodeRetention(retention); got != want {
			t.Errorf("encodeRetention(%+v) = %q, want %q", retention, got, want)
		}
		decoded, err := decodeRetention(want)
		if err != nil {
			t.Fatal(err)
		}
		if decoded != retention {
			t.Errorf("decodeRetention(%q) = %+v, want %+v", want, decoded, retention)
		}
	}
}

func TestSettingsModelRejectsUnknownPath(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	if err := newSettingsModel(store).Apply("missing", true); err == nil {
		t.Error("unknown path was accepted")
	}
}

func TestSettingsModelAppliesCaptureOrder(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	model := newSettingsModel(store)
	if err := model.Apply("capture.updateOrder", []string{"text", "files", "text"}); err != nil {
		t.Fatal(err)
	}
	want := []settings.CaptureKind{
		settings.CaptureText,
		settings.CaptureFiles,
		settings.CaptureImage,
		settings.CaptureHTML,
		settings.CaptureRTF,
	}
	got := store.Snapshot().Clipboard.Capture.Order
	if len(got) != len(want) {
		t.Fatalf("capture order = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("capture order = %v, want %v", got, want)
		}
	}
	values := model.Load().CaptureOrder
	if len(values) != len(want) || values[0] != "text" {
		t.Errorf("loaded capture order = %v", values)
	}
}

func TestSettingsModelRejectsUnknownCaptureKind(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	err = newSettingsModel(store).Apply("capture.updateOrder", []string{"text", "missing"})
	if err == nil {
		t.Fatal("unknown capture kind accepted")
	}
}

func TestSettingsModelAppliesTrayIcon(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	model := newSettingsModel(store)
	var applied []bool
	model.setTrayIcon = func(visible bool) error {
		applied = append(applied, visible)
		return nil
	}
	if err := model.Apply("general.trayIcon", false); err != nil {
		t.Fatal(err)
	}
	if store.Snapshot().General.TrayIcon {
		t.Error("tray icon setting was not persisted")
	}
	if len(applied) != 1 || applied[0] {
		t.Errorf("tray icon applications = %v", applied)
	}
	if model.Load().TrayIcon {
		t.Error("tray icon setting was not loaded")
	}
}

func TestSettingsModelAppliesTheme(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	model := newSettingsModel(store)
	var applied []string
	model.setAppearance = func(theme string) error {
		applied = append(applied, theme)
		return nil
	}
	if err := model.Apply("appearance.theme", "dark"); err != nil {
		t.Fatal(err)
	}
	if store.Snapshot().Appearance.Theme != settings.ThemeDark {
		t.Errorf("theme = %q", store.Snapshot().Appearance.Theme)
	}
	if model.Load().Theme != "dark" {
		t.Errorf("loaded theme = %q", model.Load().Theme)
	}
	if len(applied) != 1 || applied[0] != "dark" {
		t.Errorf("appearance applications = %v", applied)
	}
}

func TestSettingsModelAppliesLanguage(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	model := newSettingsModel(store)
	var applied []string
	model.setLanguage = func(language string) error {
		applied = append(applied, language)
		return nil
	}
	if err := model.Apply("appearance.language", "en-US"); err != nil {
		t.Fatal(err)
	}
	if store.Snapshot().Appearance.Language != settings.LanguageEnUS {
		t.Errorf("language = %q", store.Snapshot().Appearance.Language)
	}
	if model.Load().Language != "en-US" {
		t.Errorf("loaded language = %q", model.Load().Language)
	}
	if len(applied) != 1 || applied[0] != "en-US" {
		t.Errorf("language applications = %v", applied)
	}
}

func TestSettingsModelRunsBackupActions(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	model := newSettingsModel(store)
	var actions []string
	model.exportBackup = func() { actions = append(actions, "export") }
	model.importBackup = func() { actions = append(actions, "import") }
	if err := model.Apply("backup.export", true); err != nil {
		t.Fatal(err)
	}
	if err := model.Apply("backup.import", true); err != nil {
		t.Fatal(err)
	}
	if len(actions) != 2 || actions[0] != "export" || actions[1] != "import" {
		t.Errorf("backup actions = %v", actions)
	}

	missing := newSettingsModel(store)
	if err := missing.Apply("backup.export", true); err == nil {
		t.Error("missing export callback did not fail")
	}
}

func TestSettingsModelChecksUpdates(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	model := newSettingsModel(store)
	checked := 0
	model.checkUpdates = func() { checked++ }
	if err := model.Apply("update.check", true); err != nil {
		t.Fatal(err)
	}
	if checked != 1 {
		t.Errorf("update checked %d times", checked)
	}

	if err := newSettingsModel(store).Apply("update.check", true); err == nil {
		t.Error("missing update callback did not fail")
	}
}

func TestSettingsModelChangesDataDir(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	model := newSettingsModel(store)
	var changed []string
	restarted := false
	model.changeDataDir = func(parent string) error {
		changed = append(changed, parent)
		return nil
	}
	model.restart = func() { restarted = true }
	if err := model.Apply("general.dataDir", "/tmp/parent"); err != nil {
		t.Fatal(err)
	}
	if len(changed) != 1 || changed[0] != "/tmp/parent" || !restarted {
		t.Errorf("changed = %v, restarted = %t", changed, restarted)
	}
}

func TestSettingsModelRestoresDefaultDataDir(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	model := newSettingsModel(store)
	restored := false
	restarted := false
	model.restoreDataDir = func() error {
		restored = true
		return nil
	}
	model.restart = func() { restarted = true }
	if err := model.Apply("general.defaultDataDir", true); err != nil {
		t.Fatal(err)
	}
	if !restored || !restarted {
		t.Errorf("restored = %t, restarted = %t", restored, restarted)
	}
}

func TestSettingsModelAppliesAutostart(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	model := newSettingsModel(store)
	var applied []bool
	model.setAutoStart = func(enabled bool) error {
		applied = append(applied, enabled)
		return nil
	}
	if err := model.Apply("general.autoStart", true); err != nil {
		t.Fatal(err)
	}
	if !store.Snapshot().General.AutoStart {
		t.Error("autostart setting was not persisted")
	}
	if len(applied) != 1 || !applied[0] {
		t.Errorf("autostart applications = %v", applied)
	}
	if !model.Load().AutoStart {
		t.Error("autostart setting was not loaded")
	}
}

func TestSettingsModelKeepsAutostartWhenNativeApplyFails(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	model := newSettingsModel(store)
	model.setAutoStart = func(bool) error { return errors.New("not permitted") }
	if err := model.Apply("general.autoStart", true); err == nil {
		t.Fatal("native failure was ignored")
	}
	if store.Snapshot().General.AutoStart {
		t.Error("autostart setting was persisted after native failure")
	}
}

func TestResetSettingsRestoresDefaultsAndSideEffects(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(map[string]any{
		"general": map[string]any{"autoStart": true},
		"appearance": map[string]any{
			"theme":    "dark",
			"language": "en-US",
		},
		"clipboard": map[string]any{
			"content": map[string]any{"copyPlain": true},
			"history": map[string]any{"maxCount": 20},
		},
	}); err != nil {
		t.Fatal(err)
	}
	model := newSettingsModel(store)
	var trayIcon, autoStart bool
	var theme, language string
	rebound, redisplayed := false, false
	model.setTrayIcon = func(on bool) error { trayIcon = on; return nil }
	model.setAutoStart = func(on bool) error { autoStart = on; return nil }
	model.setAppearance = func(name string) error { theme = name; return nil }
	model.setLanguage = func(name string) error { language = name; return nil }
	model.onShortcutsChanged = func(string) error { rebound = true; return nil }
	model.onChanged = func() { redisplayed = true }

	if err := model.Apply("settings.reset", true); err != nil {
		t.Fatal(err)
	}
	defaults := settings.Default()
	if store.Snapshot().Clipboard.Content.CopyPlain {
		t.Error("copyPlain was not reset")
	}
	if trayIcon != defaults.General.TrayIcon || autoStart != defaults.General.AutoStart {
		t.Errorf("tray icon = %v, autostart = %v", trayIcon, autoStart)
	}
	if theme != string(defaults.Appearance.Theme) {
		t.Errorf("theme = %q", theme)
	}
	if language != string(defaults.Appearance.Language) {
		t.Errorf("language = %q", language)
	}
	if !rebound || !redisplayed {
		t.Errorf("rebound = %v, redisplayed = %v", rebound, redisplayed)
	}
}

func TestSettingsModelAcceptsEveryVisiblePath(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	model := newSettingsModel(store)
	// The native login item, appearance and tray calls need AppKit's main
	// queue; their own tests use fakes, so leave them out here.
	model.setAutoStart = nil
	model.setAppearance = nil
	cases := []struct {
		path  string
		value any
	}{
		{"general.autoStart", true},
		{"general.trayIcon", false},
		{"shortcuts.openClipboard", "Command+Shift+K"},
		{"shortcuts.openPreference", "Command+Shift+J"},
		{"appearance.theme", "dark"},
		{"appearance.language", "en-US"},
		{"copyThenHideWindow", true},
		{"feedback.copySound", true},
		{"sort", "createdAtDesc"},
		{"window.position", "center"},
		{"display.textMaxLines", uint32(5)},
		{"display.imageMaxHeight", uint32(100)},
		{"display.fileMaxCount", uint32(5)},
		{"window.scrollToTopOnOpen", false},
		{"window.selectRangeOnOpen", "all"},
		{"window.selectCategoryOnOpen", "image"},
		{"window.selectGroupOnOpen", "all"},
		{"search.clearOnHide", false},
		{"content.autoPaste", "singleClickPaste"},
		{"content.autoFavorite", false},
		{"content.showOriginalPreview", false},
		{"copyPlain", true},
		{"preview.spaceEnabled", false},
		{"capture.text", false},
		{"capture.html", false},
		{"capture.rtf", false},
		{"capture.image", false},
		{"capture.files", true},
		{"capture.maxTextMb", uint32(8)},
		{"capture.maxImageMb", uint32(50)},
		{"history.retention", "3d"},
		{"history.maxCount", uint32(500)},
		{"sensitive.collectSecrets", false},
		{"sensitive.redactSecrets", false},
		{"content.deleteNormal", "confirm"},
		{"content.deleteFavorite", "direct"},
		{"content.deletePinned", "forbidden"},
	}
	for _, tc := range cases {
		if err := model.Apply(tc.path, tc.value); err != nil {
			t.Errorf("apply %s: %v", tc.path, err)
		}
	}
	reopened, err := settings.Open(store.Path())
	if err != nil {
		t.Fatal(err)
	}
	snapshot := reopened.Snapshot()
	if snapshot.Clipboard.Search.ClearOnHide ||
		snapshot.Clipboard.Preview.SpaceEnabled || snapshot.Clipboard.History.Retention.Value != 3 ||
		snapshot.Clipboard.History.MaxCount != 500 {
		t.Errorf("reopened settings = %+v", snapshot.Clipboard)
	}
}
