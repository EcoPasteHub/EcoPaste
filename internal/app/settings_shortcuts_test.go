package app

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/settings"
)

func TestNormalizeShortcut(t *testing.T) {
	valid := []string{"Alt+C", "Command+Shift+K", "Control+Alt+Delete", "F5", "Alt+ArrowUp"}
	for _, value := range valid {
		if got, err := normalizeShortcut(value); err != nil || got != value {
			t.Errorf("normalizeShortcut(%q) = %q, %v", value, got, err)
		}
	}
	invalid := []string{"C", "Shift+C", "Hyper+C", "Alt+Nope", "Command+Shift+"}
	for _, value := range invalid {
		if _, err := normalizeShortcut(value); err == nil {
			t.Errorf("normalizeShortcut(%q) accepted an invalid shortcut", value)
		}
	}
	if got, err := normalizeShortcut("  "); err != nil || got != "" {
		t.Errorf("empty shortcut = %q, %v", got, err)
	}
}

func TestSettingsModelAppliesShortcutAndRejectsDuplicate(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	model := newSettingsModel(store)
	changed := 0
	model.onShortcutsChanged = func(string) error { changed++; return nil }
	if err := model.Apply("shortcuts.openClipboard", "Command+Shift+K"); err != nil {
		t.Fatal(err)
	}
	if got := store.Snapshot().Shortcuts.OpenClipboard; got != "Command+Shift+K" {
		t.Errorf("open clipboard = %q", got)
	}
	if changed != 1 {
		t.Errorf("changed = %d", changed)
	}
	if err := model.Apply("shortcuts.openPreference", " command+shift+k "); err == nil {
		t.Fatal("duplicate shortcut was accepted")
	}
	if err := model.Apply("shortcuts.openClipboard", ""); err != nil {
		t.Fatal(err)
	}
	if got := store.Snapshot().Shortcuts.OpenClipboard; got != "" {
		t.Errorf("cleared shortcut = %q", got)
	}
}

func TestSettingsModelRollsBackShortcutWhenRebindFails(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	model := newSettingsModel(store)
	attempts := make([]string, 0, 2)
	model.onShortcutsChanged = func(changed string) error {
		if changed != "openClipboard" {
			t.Errorf("changed key = %q", changed)
		}
		current := store.Snapshot().Shortcuts.OpenClipboard
		attempts = append(attempts, current)
		if current == "Command+Shift+K" {
			return errors.New("快捷键已被占用")
		}
		return nil
	}
	before := store.Snapshot().Shortcuts.OpenClipboard
	if err := model.Apply("shortcuts.openClipboard", "Command+Shift+K"); err == nil {
		t.Fatal("rebind failure was ignored")
	}
	if got := store.Snapshot().Shortcuts.OpenClipboard; got != before {
		t.Errorf("shortcut = %q, want rollback to %q", got, before)
	}
	if len(attempts) != 2 || attempts[0] != "Command+Shift+K" || attempts[1] != before {
		t.Errorf("rebind attempts = %v", attempts)
	}
}
