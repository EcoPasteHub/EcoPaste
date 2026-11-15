package app

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/EcoPasteHub/EcoPaste/internal/backup"
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/EcoPasteHub/EcoPaste/internal/logging"
	"github.com/EcoPasteHub/EcoPaste/internal/menu"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
	"github.com/EcoPasteHub/EcoPaste/internal/storage"
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
	var rebuildPanel func()
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
	var store *settings.Store
	var windowState *storage.WindowState
	if paths, err := storage.Default(); err != nil {
		log.Printf("resolve storage: %v", err)
	} else {
		lock, running, err := lockInstance(paths.State)
		switch {
		case err != nil:
			log.Printf("lock instance: %v", err)
		case running:
			if err := signalInstance(paths.State); err != nil {
				log.Printf("signal instance: %v", err)
			}
			os.Exit(0)
		default:
			app.Cleanup(func() { lock.Close() })
		}
		if logger, err := logging.Open(paths.Logs); err != nil {
			log.Printf("open logs: %v", err)
		} else {
			app.Cleanup(func() { logger.Close() })
		}
		if result, applied, err := backup.ApplyPendingImport(paths); err != nil {
			log.Printf("apply pending import: %v", err)
		} else if applied {
			log.Printf("imported backup: %d items", result.Items)
		}
		log.Printf("data root %s", paths.Root)
		if state, err := storage.OpenWindowState(paths.State); err != nil {
			log.Printf("open window state: %v", err)
		} else {
			windowState = state
		}
		store, err = settings.Open(filepath.Join(paths.Config, settings.FileName))
		if err != nil {
			log.Printf("open settings: %v", err)
		} else {
			i18n.SetLanguage(string(store.Snapshot().Appearance.Language))
			app.SetLocale(string(store.Snapshot().Appearance.Language))
			if err := macos.SetAppearance(string(store.Snapshot().Appearance.Theme)); err != nil {
				log.Printf("sync appearance: %v", err)
			}
			if err := macos.SetDockIconVisible(store.Snapshot().General.DockIcon); err != nil {
				log.Printf("sync dock icon: %v", err)
			}
			if err := macos.SyncAutostart(store.Snapshot().General.AutoStart); err != nil {
				log.Printf("sync autostart: %v", err)
			}
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
			runtime, err = startClipboard(app, paths, store, &paused)
			if err != nil {
				log.Printf("start clipboard: %v", err)
			} else {
				history = newHistoryModel(runtime, store)
				if model, ok := history.(*historyModel); ok {
					model.openSettings = wiring.openTab
				}
			}
		}
	}
	var panel *desktop.Window
	placePanel := func() {
		mode := string(settings.Default().Clipboard.Window.Position)
		if store != nil {
			mode = string(store.Snapshot().Clipboard.Window.Position)
		}
		placement := macos.PanelPlacement{Mode: mode}
		if windowState != nil {
			if frame, ok := windowState.Frame(panelStateLabel); ok {
				placement.Frame = macos.PanelFrame{
					X: float64(frame.X), Y: float64(frame.Y),
					Width: float64(frame.Width), Height: float64(frame.Height),
				}
				placement.HasFrame = true
			}
		}
		macos.PlacePanelOnNextShow(placement)
	}
	savePanelFrame := func() {
		if windowState == nil {
			return
		}
		frame, ok := macos.PanelFrameOf()
		if !ok {
			return
		}
		err := windowState.SaveFrame(panelStateLabel, storage.WindowFrame{
			X: int64(frame.X), Y: int64(frame.Y),
			Width: int64(frame.Width), Height: int64(frame.Height),
		})
		if err != nil {
			log.Printf("save window state: %v", err)
		}
	}
	hidePanel := func() {
		savePanelFrame()
		panel.Hide()
		// Window commands issued from a Carbon callback need a tree update
		// before NSGo applies them.
		if runtime != nil {
			runtime.Signal()
		}
	}
	content := ui.ClipboardPanel(history, hidePanel)
	openPanel := func() {
		panel = desktop.OpenPanel(app, desktop.PanelOptions{
			Width:    420,
			Height:   560,
			Material: desktop.WindowMaterialSidebar,
		}, content)
	}
	openPanel()
	rebuildPanel = func() {
		if panel != nil && !panel.Closed() {
			if panel.Visible() {
				savePanelFrame()
			}
			panel.Close()
		}
		openPanel()
	}
	if wiring != nil {
		wiring.rebuildPanel = rebuildPanel
	}

	showPanel := func() {
		if runtime != nil {
			runtime.Signal()
		}
		placePanel()
		panel.Show()
	}
	togglePanel := func() {
		if panel.Visible() {
			hidePanel()
			return
		}
		showPanel()
	}

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
		Panel:    panel,
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
