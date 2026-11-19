package app

import (
	"context"
	"log"
	"os"
	"sync/atomic"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/EcoPasteHub/EcoPaste/internal/menu"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/EcoPasteHub/EcoPaste/internal/ui"
	"github.com/EcoPasteHub/EcoPaste/internal/update"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
)

const panelHotKey = "alt+c"

const panelStateLabel = "clipboard"

// Run opens EcoPaste's panel, tray icon and global shortcut. The app
// keeps running without windows until the user quits it.
func Run(app *core.Scope) {
	if err := developmentIdentityError(core.App()); err != nil {
		log.Print(err)
		os.Exit(1)
	}
	i18n.Bind(app)
	desktop.SetShutdownMode(app, desktop.ShutdownOnExplicit)
	activation := instanceRequests(app)
	var paused atomic.Bool
	var history ui.HistoryModel
	var runtime *clipboardRuntime
	var openSettings func()
	var hotKeys *hotKeySet
	var tray *desktop.Tray
	var wiring *settingsWiring
	manager := macos.NewHotKeyManager()
	app.OnQuit(func(context.Context) { manager.Close() })
	applyApplicationMenu := func() {
		desktop.SetApplicationMenu(app, menu.Application(string(i18n.Current()), func() {
			if openSettings != nil {
				openSettings()
			}
		}, func() { update.CheckNow(app) }))
	}
	applyApplicationMenu()
	paths, store, windowState := loadStorage(app)
	hotKeys = newHotKeySet(app, manager, store)
	if store != nil {
		syncStoredSettings(app, store)
		wiring = &settingsWiring{
			app:                  app,
			store:                store,
			hotKeys:              manager,
			runtime:              func() *clipboardRuntime { return runtime },
			tray:                 func() *desktop.Tray { return tray },
			openClipboardKey:     func() string { return hotKeys.openClipboard },
			openPreferenceKey:    func() string { return hotKeys.openPreference },
			applyApplicationMenu: applyApplicationMenu,
		}
		openSettings = wiring.open
		hotKeys.openSettings = openSettings
		var startErr error
		runtime, startErr = startClipboard(app, paths, store, &paused)
		if startErr != nil {
			log.Printf("start clipboard: %v", startErr)
		} else {
			history = newHistoryModel(runtime, store)
			if model, ok := history.(*historyModel); ok {
				model.openSettings = wiring.openTab
			}
		}
	}
	panelHost := newPanelHost(app, store, windowState, func() *clipboardRuntime { return runtime }, history)
	hotKeys.togglePanel = panelHost.Toggle

	showPanel := panelHost.Show

	tray = newAppTray(app, panelHost.Window(), runtime, store, &paused, showPanel, openSettings, hotKeys)

	if wiring != nil {
		wiring.reloadHotkeys = hotKeys.reload
	}
	_ = hotKeys.reload()
	if os.Getenv("ECOPASTE_SHOW_SETTINGS") == "1" && openSettings != nil {
		openSettings()
	}

	if os.Getenv("ECOPASTE_SHOW_ON_LAUNCH") == "1" {
		showPanel()
	}

	desktop.OnReopen(app, func() {
		if openSettings != nil {
			openSettings()
		}
	})
	desktop.OnOpenFiles(app, func(paths []string) {
		if path, ok := firstBackupPath(paths); ok {
			runBackupImport(app, runtime, path, store, hotKeys.reload)
		}
	})
	go func() {
		for {
			select {
			case <-app.Context().Done():
				return
			case <-activation:
				if openSettings != nil {
					app.Post(openSettings)
				}
			}
		}
	}()
}
