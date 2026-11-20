package app

import (
	"log/slog"
	"os"
	"strings"

	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
	"github.com/nsgo-dev/nsgo/core"
)

// hotKeySet owns the configurable global shortcuts and the manager that
// registers them.
type hotKeySet struct {
	app            *core.Scope
	manager        *macos.HotKeyManager
	store          *settings.Store
	openClipboard  string
	openPreference string
	togglePanel    func()
	openSettings   func()
}

func newHotKeySet(app *core.Scope, manager *macos.HotKeyManager, store *settings.Store) *hotKeySet {
	keys := &hotKeySet{
		app:            app,
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
		// An empty value disables the binding. Falling back to defaults here
		// would register a shortcut the user already removed.
		k.openClipboard = strings.ToLower(snapshot.Shortcuts.OpenClipboard)
		k.openPreference = strings.ToLower(snapshot.Shortcuts.OpenPreference)
	}
	if custom := os.Getenv("ECOPASTE_HOTKEY"); custom != "" {
		k.openClipboard = strings.ToLower(custom)
	}
}

func (k *hotKeySet) reload() error {
	k.applyStored()
	var bindings []macos.HotKeyBinding
	if k.openClipboard != "" && k.togglePanel != nil {
		bindings = append(bindings, macos.HotKeyBinding{Key: k.openClipboard, Action: k.onGraph(k.togglePanel)})
	}
	if k.openPreference != "" && k.openSettings != nil {
		bindings = append(bindings, macos.HotKeyBinding{Key: k.openPreference, Action: k.onGraph(k.openSettings)})
	}
	if err := k.manager.Set(bindings); err != nil {
		slog.Error("register global shortcuts", "module", "app.hotkey", "error", err)
		return err
	}
	return nil
}

func (k *hotKeySet) shortcuts() (string, string) {
	return k.openClipboard, k.openPreference
}

// onGraph hands fn to the reactive graph's goroutine. Carbon fires hotkeys on
// the AppKit main thread, where NSGo window signals belong to another
// goroutine, so a shortcut that opened or hid a window would panic there.
func (k *hotKeySet) onGraph(fn func()) func() {
	return func() {
		if k.app == nil {
			fn()
			return
		}
		k.app.Post(fn)
	}
}
