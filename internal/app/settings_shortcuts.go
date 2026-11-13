package app

import (
	"errors"
	"fmt"
	"strings"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
)

var shortcutModifiers = map[string]struct{}{
	"command": {}, "control": {}, "alt": {}, "shift": {},
}

// normalizeShortcut validates one recorded shortcut and returns it in the
// stored form. An empty value disables the binding.
func normalizeShortcut(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	parts := strings.Split(trimmed, "+")
	key := parts[len(parts)-1]
	if key == "" {
		return "", errors.New(i18n.T(i18n.ShortcutMissingKey))
	}
	primary := strings.ToLower(key)
	if len(parts) == 1 {
		if supportedShortcutKey(primary) && standaloneFunctionKey(primary) {
			return trimmed, nil
		}
		return "", errors.New(i18n.T(i18n.ShortcutMissingModifier))
	}
	if !supportedShortcutKey(primary) {
		return "", fmt.Errorf(i18n.T(i18n.ShortcutUnsupportedKey), key)
	}
	hasPrimaryModifier := false
	for _, modifier := range parts[:len(parts)-1] {
		name := strings.ToLower(strings.TrimSpace(modifier))
		if _, ok := shortcutModifiers[name]; !ok {
			return "", fmt.Errorf(i18n.T(i18n.ShortcutUnsupportedMod), modifier)
		}
		if name != "shift" {
			hasPrimaryModifier = true
		}
	}
	if !hasPrimaryModifier && !standaloneFunctionKey(primary) {
		return "", errors.New(i18n.T(i18n.ShortcutNeedsPrimaryMod))
	}
	return trimmed, nil
}

func standaloneFunctionKey(key string) bool {
	if len(key) < 2 || key[0] != 'f' {
		return false
	}
	switch key {
	case "f1", "f2", "f3", "f4", "f5", "f6", "f7", "f8", "f9", "f10", "f11", "f12":
		return true
	}
	return false
}

func supportedShortcutKey(key string) bool {
	if len(key) == 1 && strings.Contains("abcdefghijklmnopqrstuvwxyz0123456789-=[]\\;',./`", key) {
		return true
	}
	switch key {
	case "space", "tab", "enter", "backspace", "delete", "esc", "escape",
		"home", "end", "pageup", "pagedown", "arrowup", "arrowdown", "arrowleft", "arrowright":
		return true
	}
	for i := 1; i <= 20; i++ {
		if key == fmt.Sprintf("f%d", i) {
			return true
		}
	}
	return false
}

func shortcutsEqual(left, right string) bool {
	return strings.EqualFold(strings.TrimSpace(left), strings.TrimSpace(right))
}

// applyShortcut persists one global shortcut and asks the app to rebind.
func (m *settingsModel) applyShortcut(path string, value any) error {
	text, _ := value.(string)
	shortcut, err := normalizeShortcut(text)
	if err != nil {
		return err
	}
	key := "openClipboard"
	if path == "shortcuts.openPreference" {
		key = "openPreference"
	}
	snapshot := m.store.Snapshot().Shortcuts
	other := snapshot.OpenPreference
	previous := snapshot.OpenClipboard
	if key == "openPreference" {
		other = snapshot.OpenClipboard
		previous = snapshot.OpenPreference
	}
	if shortcut != "" && shortcutsEqual(shortcut, other) {
		return errors.New(i18n.T(i18n.ShortcutInUse))
	}
	if _, err := m.store.Update(map[string]any{
		"shortcuts": map[string]any{key: shortcut},
	}); err != nil {
		return fmt.Errorf(i18n.T(i18n.ShortcutSave), err)
	}
	if m.onShortcutsChanged != nil {
		if err := m.onShortcutsChanged(key); err != nil {
			restore := map[string]any{"shortcuts": map[string]any{key: previous}}
			if _, rollbackErr := m.store.Update(restore); rollbackErr != nil {
				return fmt.Errorf(i18n.T(i18n.ShortcutRestore), err, rollbackErr)
			}
			if rollbackErr := m.onShortcutsChanged(key); rollbackErr != nil {
				return fmt.Errorf(i18n.T(i18n.ShortcutRestoreRegister), err, rollbackErr)
			}
			return err
		}
	}
	return nil
}
