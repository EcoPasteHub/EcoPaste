package app

import (
	"path/filepath"
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/settings"
)

func TestOnboardingDefaultsAndPersistence(t *testing.T) {
	root := t.TempDir()
	store, err := settings.Open(filepath.Join(root, settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	model := &historyModel{store: store}
	data := model.Onboarding()
	if !data.Active || data.Step != 0 {
		t.Fatalf("onboarding = %+v", data)
	}
	if err := model.SaveOnboarding(2, false); err != nil {
		t.Fatal(err)
	}
	data = model.Onboarding()
	if !data.Active || data.Step != 2 {
		t.Errorf("onboarding = %+v", data)
	}
	if got := store.Snapshot().Onboarding.LastStep; got != 2 {
		t.Errorf("last step = %d", got)
	}
}

func TestOnboardingFinishAndClamp(t *testing.T) {
	root := t.TempDir()
	store, err := settings.Open(filepath.Join(root, settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	model := &historyModel{store: store}
	if err := model.SaveOnboarding(99, true); err != nil {
		t.Fatal(err)
	}
	snapshot := store.Snapshot().Onboarding
	if !snapshot.Completed || snapshot.LastStep != 5 {
		t.Errorf("onboarding = %+v", snapshot)
	}
	if data := model.Onboarding(); data.Active || data.Step != 5 {
		t.Errorf("onboarding = %+v", data)
	}
}

func TestOnboardingReportsShortcutsAndExcludedApps(t *testing.T) {
	root := t.TempDir()
	store, err := settings.Open(filepath.Join(root, settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(map[string]any{
		"clipboard": map[string]any{"filters": map[string]any{
			"excludedAppIds": []any{"com.apple.Passwords", "com.apple.keychainaccess"},
		}},
	}); err != nil {
		t.Fatal(err)
	}
	data := (&historyModel{store: store}).Onboarding()
	if data.ExcludedApps != 2 {
		t.Errorf("excluded apps = %d", data.ExcludedApps)
	}
	if data.OpenShortcut != "Alt+C" || data.PreferenceShortcut != "Alt+X" {
		t.Errorf("shortcuts = %q / %q", data.OpenShortcut, data.PreferenceShortcut)
	}
}
