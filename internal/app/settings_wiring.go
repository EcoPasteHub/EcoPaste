package app

import (
	"context"
	"log"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
	"github.com/EcoPasteHub/EcoPaste/internal/storage"
	"github.com/EcoPasteHub/EcoPaste/internal/ui"
	"github.com/EcoPasteHub/EcoPaste/internal/update"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
)

// settingsWiring connects the settings UI to the running app. The runtime
// and tray are fetched lazily because the settings window can be created
// before either service finishes starting.
type settingsWiring struct {
	app                  *core.Scope
	store                *settings.Store
	hotKeys              *macos.HotKeyManager
	runtime              func() *clipboardRuntime
	tray                 func() *desktop.Tray
	openClipboardKey     func() string
	openPreferenceKey    func() string
	applyApplicationMenu func()
	reloadHotkeys        func() error
}

func (w *settingsWiring) open() {
	desktop.OpenWindow(w.app, desktop.WindowOptions{
		Title:       i18n.T(i18n.SettingsWindowTitle),
		Width:       480,
		Height:      640,
		Key:         "settings",
		Constraints: core.WindowConstraints{NoResize: true, NoMaximize: true},
	}, ui.SettingsWindow(w.newModel()))
}

func (w *settingsWiring) openTab(tab string) {
	macos.SelectSettingsTab(tab)
	w.open()
}

func (w *settingsWiring) newModel() *settingsModel {
	model := newSettingsModel(w.store)
	model.dataDir = w.dataDir
	model.changeDataDir = w.changeDataDir
	model.canRestoreDataDir = w.canRestoreDataDir
	model.restoreDataDir = w.restoreDataDir
	model.listApps = w.listApps
	model.listGroups = w.listGroups
	model.addApp = w.addApp
	model.deleteApps = w.deleteApps
	model.appIconPath = w.appIconPath
	model.onShortcutsChanged = w.onShortcutsChanged
	model.onShortcutsSuspended = w.onShortcutsSuspended
	model.onChanged = w.signalHistory
	model.onHistoryChanged = w.cleanupHistory
	model.setLanguage = w.setLanguage
	model.setTrayIcon = w.setTrayIcon
	model.exportBackup = w.exportBackup
	model.importBackup = w.importBackup
	model.checkUpdates = w.checkUpdates
	model.restart = w.restart
	return model
}

func (w *settingsWiring) dataDir() string {
	if runtime := w.runtime(); runtime != nil {
		return runtime.paths.Root
	}
	return ""
}

func (w *settingsWiring) changeDataDir(parent string) error {
	return runWithTimeout(w.runtime(), 10*time.Minute, func(ctx context.Context, rt *clipboardRuntime) error {
		_, err := rt.changeDataDir(ctx, parent)
		return err
	})
}

func (w *settingsWiring) canRestoreDataDir() bool {
	runtime := w.runtime()
	if runtime == nil {
		return false
	}
	paths, err := storage.Resolve(runtime.paths.Base, storage.Mode())
	return err == nil && paths.Root != paths.Bootstrap
}

func (w *settingsWiring) restoreDataDir() error {
	return runWithTimeout(w.runtime(), 10*time.Minute, func(ctx context.Context, rt *clipboardRuntime) error {
		_, err := rt.restoreDefaultDataDir(ctx)
		return err
	})
}

func (w *settingsWiring) listApps() ([]database.App, error) {
	return runWithTimeoutValue(w.runtime(), 5*time.Second, func(ctx context.Context, rt *clipboardRuntime) ([]database.App, error) {
		if rt.apps == nil {
			return nil, clipboardServiceDown()
		}
		return rt.apps.List(ctx)
	})
}

func (w *settingsWiring) listGroups() ([]database.Group, error) {
	return runWithTimeoutValue(w.runtime(), 5*time.Second, func(ctx context.Context, rt *clipboardRuntime) ([]database.Group, error) {
		return rt.db.ListGroups(ctx)
	})
}

func (w *settingsWiring) addApp(path string) (database.App, error) {
	return runWithTimeoutValue(w.runtime(), 10*time.Second, func(ctx context.Context, rt *clipboardRuntime) (database.App, error) {
		if rt.apps == nil {
			return database.App{}, clipboardServiceDown()
		}
		return rt.apps.Add(ctx, path)
	})
}

func (w *settingsWiring) deleteApps(ids []string) error {
	return runWithTimeout(w.runtime(), 10*time.Second, func(ctx context.Context, rt *clipboardRuntime) error {
		if rt.apps == nil {
			return clipboardServiceDown()
		}
		_, err := rt.apps.DeleteUnreferenced(ctx, ids)
		return err
	})
}

func (w *settingsWiring) appIconPath(name string) string {
	if runtime := w.runtime(); runtime != nil {
		return runtime.appIcons.Path(name)
	}
	return ""
}

func (w *settingsWiring) onShortcutsChanged(changed string) error {
	if w.reloadHotkeys == nil {
		return nil
	}
	err := w.reloadHotkeys()
	if err == nil || changed == "" || !w.hotKeys.HasErrors() {
		return err
	}
	key := w.openClipboardKey()
	if changed == "openPreference" {
		key = w.openPreferenceKey()
	}
	return w.hotKeys.ErrorFor(key)
}

func (w *settingsWiring) onShortcutsSuspended(suspend bool) {
	if suspend {
		_ = w.hotKeys.Set(nil)
		return
	}
	if w.reloadHotkeys != nil {
		_ = w.reloadHotkeys()
	}
}

func (w *settingsWiring) signalHistory() {
	if runtime := w.runtime(); runtime != nil {
		runtime.Signal()
	}
}

// cleanupHistory applies retention and count limits immediately after their
// settings change. It runs off the graph goroutine and refreshes the list.
func (w *settingsWiring) cleanupHistory() {
	runtime := w.runtime()
	if runtime == nil || runtime.service == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		result, err := runtime.service.Cleanup(ctx, time.Now())
		if err != nil {
			log.Printf("cleanup after settings change: %v", err)
			return
		}
		if result.Removed > 0 {
			runtime.Signal()
		}
	}()
}

func (w *settingsWiring) setLanguage(language string) error {
	w.app.Post(func() {
		applyAppLanguage(w.app, language)
		w.applyApplicationMenu()
	})
	return nil
}

func (w *settingsWiring) setTrayIcon(visible bool) error {
	tray := w.tray()
	if tray == nil {
		return i18n.Error(i18n.ErrorTrayNotReady)
	}
	icon := []byte(nil)
	if visible {
		icon = trayIcon()
	}
	// Settings edits run on a worker goroutine; a tray change mutates the
	// reactive tree, so it must run on the graph's goroutine. Post it instead
	// of touching the tree here.
	w.app.Post(func() { tray.SetIcon(icon, true) })
	return nil
}

func (w *settingsWiring) exportBackup() {
	exportBackupWithUI(w.app, w.runtime())
}

func (w *settingsWiring) importBackup() {
	importBackupWithUI(w.app, w.runtime(), w.store, w.reloadHotkeys)
}

func (w *settingsWiring) checkUpdates() {
	update.CheckNow(w.app)
}

func (w *settingsWiring) restart() {
	w.app.Post(func() {
		macos.ShowMessage(i18n.T(i18n.DataDirChangedTitle), i18n.T(i18n.DataDirChangedBody))
		desktop.Restart(w.app)
	})
}
