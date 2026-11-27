package app

import (
	"path/filepath"
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
)

func TestSettingsModelAppliesSourceAppModeAndKeepsOneRule(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	model := newSettingsModel(store)
	icon := "icon.png"
	model.listApps = func() ([]database.App, error) {
		return []database.App{
			{ID: "com.example.a", Name: "A", IconFile: &icon},
			{ID: "com.example.b", Name: "B"},
			{ID: "com.apple.Passwords", Name: "Passwords"},
		}, nil
	}
	model.appIconPath = func(name string) string { return "/icons/" + name }
	var ensured []string
	model.ensureApp = func(id, name string) error {
		ensured = append(ensured, id+":"+name)
		return nil
	}

	apply := func(id, name, mode string) {
		t.Helper()
		if err := model.Apply("sourceApps.mode", macos.SettingsAppSelection{ID: id, Name: name, Mode: mode}); err != nil {
			t.Fatal(err)
		}
	}
	apply("com.example.a", "A", string(settings.SourceAppBlockCapture))
	apply("com.example.b", "B", string(settings.SourceAppBlockWake))
	// Changing A's mode overwrites its single rule instead of adding another.
	apply("com.example.a", "A", string(settings.SourceAppBlockCaptureAndWake))

	rules := store.Snapshot().Clipboard.SourceApps.Rules
	if len(rules) != 2 {
		t.Fatalf("rules = %+v, want two apps", rules)
	}
	if rules[0].ID != "com.example.a" || rules[0].Mode != settings.SourceAppBlockCaptureAndWake {
		t.Errorf("rule for A = %+v", rules[0])
	}
	if rules[1].ID != "com.example.b" || rules[1].Mode != settings.SourceAppBlockWake {
		t.Errorf("rule for B = %+v", rules[1])
	}

	// Choosing the default removes the record.
	apply("com.example.a", "A", string(settings.SourceAppAllowed))
	rules = store.Snapshot().Clipboard.SourceApps.Rules
	if len(rules) != 1 || rules[0].ID != "com.example.b" {
		t.Fatalf("rules after reset = %+v", rules)
	}

	values := model.Load()
	if len(values.Apps) != 3 {
		t.Fatalf("apps = %+v", values.Apps)
	}
	byID := map[string]macos.SettingsApp{}
	for _, app := range values.Apps {
		byID[app.ID] = app
	}
	if got := byID["com.example.a"].Mode; got != string(settings.SourceAppAllowed) {
		t.Errorf("A mode = %q", got)
	}
	if got := byID["com.example.b"].Mode; got != string(settings.SourceAppBlockWake) {
		t.Errorf("B mode = %q", got)
	}
	if got := byID["com.apple.Passwords"].Mode; got != string(settings.SourceAppBlockCapture) {
		t.Errorf("Passwords default mode = %q", got)
	}

	// Defaults are not persisted; only the app the user changed was ensured.
	if len(ensured) != 3 || ensured[2] != "com.example.a:A" {
		t.Errorf("ensured = %v", ensured)
	}
}

func TestSettingsModelCanExplicitlyAllowPasswordApp(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	model := newSettingsModel(store)
	model.ensureApp = func(id, name string) error { return nil }
	if err := model.Apply("sourceApps.mode", macos.SettingsAppSelection{
		ID: "com.apple.Passwords", Name: "Passwords", Mode: string(settings.SourceAppAllowed),
	}); err != nil {
		t.Fatal(err)
	}
	rules := store.Snapshot().Clipboard.SourceApps.Rules
	if len(rules) != 1 || rules[0].ID != "com.apple.Passwords" || rules[0].Mode != settings.SourceAppAllowed {
		t.Fatalf("rules = %+v", rules)
	}
}

func TestSettingsModelRejectsUnknownSourceAppMode(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	model := newSettingsModel(store)
	if err := model.Apply("sourceApps.mode", macos.SettingsAppSelection{ID: "com.example.a", Mode: "silent"}); err == nil {
		t.Fatal("expected an invalid mode error")
	}
	if rules := store.Snapshot().Clipboard.SourceApps.Rules; len(rules) != 0 {
		t.Fatalf("rules = %+v", rules)
	}
}
