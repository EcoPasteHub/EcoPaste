package app

import (
	"context"
	"log"
	"os"
	"strings"
	"sync/atomic"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/EcoPasteHub/EcoPaste/internal/menu"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
	"github.com/EcoPasteHub/EcoPaste/internal/ui"
	"github.com/EcoPasteHub/EcoPaste/internal/update"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
)

const panelHotKey = "alt+c"

const panelStateLabel = "clipboard"

const sourceURL = "https://github.com/EcoPasteHub/EcoPaste"

// Run opens EcoPaste's panel, tray icon and global shortcut. The app
// keeps running without windows until the user quits it.
func Run(app *core.Scope) {
	if err := developmentIdentityError(core.App()); err != nil {
		log.Print(err)
		os.Exit(1)
	}
	desktop.SetShutdownMode(app, desktop.ShutdownOnExplicit)
	activation := instanceRequests(app)
	hotKey := panelHotKey
	preferenceHotKey := "alt+x"
	var paused atomic.Bool
	var history ui.HistoryModel
	var runtime *clipboardRuntime
	var openSettings func()
	var reloadHotkeys func() error
	var tray *desktop.Tray
	var wiring *settingsWiring
	hotKeys := macos.NewHotKeyManager()
	app.OnQuit(func(context.Context) { hotKeys.Close() })
	applyApplicationMenu := func() {
		desktop.SetApplicationMenu(app, menu.Application(string(i18n.Current()), func() {
			if openSettings != nil {
				openSettings()
			}
		}, func() { update.CheckNow(app) }))
	}
	applyApplicationMenu()
	paths, store, windowState := loadStorage(app)
	if store != nil {
		syncStoredSettings(app, store)
		if configured := store.Snapshot().Shortcuts.OpenClipboard; configured != "" {
			hotKey = strings.ToLower(configured)
		}
		if configured := store.Snapshot().Shortcuts.OpenPreference; configured != "" {
			preferenceHotKey = strings.ToLower(configured)
		}
		wiring = &settingsWiring{
			app:                  app,
			store:                store,
			hotKeys:              hotKeys,
			runtime:              func() *clipboardRuntime { return runtime },
			tray:                 func() *desktop.Tray { return tray },
			openClipboardKey:     func() string { return hotKey },
			openPreferenceKey:    func() string { return preferenceHotKey },
			applyApplicationMenu: applyApplicationMenu,
		}
		openSettings = wiring.open
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
	if wiring != nil {
		wiring.rebuildPanel = panelHost.Rebuild
	}

	showPanel := panelHost.Show
	togglePanel := panelHost.Toggle

	trayVisible := settings.Default().General.TrayIcon
	if store != nil {
		trayVisible = store.Snapshot().General.TrayIcon
	}
	var trayImage []byte
	if trayVisible {
		trayImage = trayIcon()
	}
	tray = desktop.NewTray(app, desktop.TrayOptions{
		Icon:     trayImage,
		Template: true,
		Tooltip:  "EcoPaste",
		Panel:    panelHost.Window(),
		Menu: trayMenu{
			app:       app,
			runtime:   runtime,
			store:     store,
			paused:    &paused,
			showPanel: showPanel,
			openSettings: func() {
				if openSettings != nil {
					openSettings()
				}
			},
			shortcuts: func() (string, string) {
				return hotKey, preferenceHotKey
			},
			reloadHotkeys: func() error {
				if reloadHotkeys == nil {
					return nil
				}
				return reloadHotkeys()
			},
		}.items,
	})

	reloadHotkeys = func() error {
		if store != nil {
			hotKey = strings.ToLower(store.Snapshot().Shortcuts.OpenClipboard)
			preferenceHotKey = strings.ToLower(store.Snapshot().Shortcuts.OpenPreference)
		}
		if custom := os.Getenv("ECOPASTE_HOTKEY"); custom != "" {
			hotKey = strings.ToLower(custom)
		}
		var bindings []macos.HotKeyBinding
		if hotKey != "" {
			bindings = append(bindings, macos.HotKeyBinding{Key: hotKey, Action: togglePanel})
		}
		if openSettings != nil && preferenceHotKey != "" {
			bindings = append(bindings, macos.HotKeyBinding{Key: preferenceHotKey, Action: openSettings})
		}
		if err := hotKeys.Set(bindings); err != nil {
			log.Printf("register hotkeys: %v", err)
			return err
		}
		return nil
	}
	if wiring != nil {
		wiring.reloadHotkeys = reloadHotkeys
	}
	_ = reloadHotkeys()
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
			runBackupImport(app, runtime, path, store, reloadHotkeys)
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
