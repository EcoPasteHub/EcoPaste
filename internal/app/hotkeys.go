package app

import (
	"log"
	"os"
	"strings"

	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
)

// hotKeySet owns the configurable global shortcuts and the manager that
// registers them.
type hotKeySet struct {
	manager        *macos.HotKeyManager
	store          *settings.Store
	openClipboard  string
	openPreference string
	togglePanel    func()
	openSettings   func()
}

func newHotKeySet(manager *macos.HotKeyManager, store *settings.Store) *hotKeySet {
	keys := &hotKeySet{
		manager:        manager,
		store:          store,
		openClipboard:  panelHotKey,
		openPreference: "alt+x",
	}
	keys.applyStored()
	return keys
}

func (k *hotKeySet) applyStored() {
	if k.store != nil {
		snapshot := k.store.Snapshot()
		if configured := snapshot.Shortcuts.OpenClipboard; configured != "" {
			k.openClipboard = strings.ToLower(configured)
		}
		if configured := snapshot.Shortcuts.OpenPreference; configured != "" {
			k.openPreference = strings.ToLower(configured)
		}
	}
	if custom := os.Getenv("ECOPASTE_HOTKEY"); custom != "" {
		k.openClipboard = strings.ToLower(custom)
	}
}

func (k *hotKeySet) reload() error {
	k.applyStored()
	var bindings []macos.HotKeyBinding
	if k.openClipboard != "" && k.togglePanel != nil {
		bindings = append(bindings, macos.HotKeyBinding{Key: k.openClipboard, Action: k.togglePanel})
	}
	if k.openPreference != "" && k.openSettings != nil {
		bindings = append(bindings, macos.HotKeyBinding{Key: k.openPreference, Action: k.openSettings})
	}
	if err := k.manager.Set(bindings); err != nil {
		log.Printf("register hotkeys: %v", err)
		return err
	}
	return nil
}

func (k *hotKeySet) shortcuts() (string, string) {
	return k.openClipboard, k.openPreference
}
