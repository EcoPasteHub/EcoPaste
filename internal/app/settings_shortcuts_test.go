package app

import (
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
	model.onShortcutsChanged = func() { changed++ }
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
