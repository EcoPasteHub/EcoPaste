package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/backup"
	"github.com/EcoPasteHub/EcoPaste/internal/clipboard"
	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/EcoPasteHub/EcoPaste/internal/logging"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
	"github.com/EcoPasteHub/EcoPaste/internal/storage"
	"github.com/EcoPasteHub/EcoPaste/internal/ui"
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
	var settingsWindow *desktop.Window
	var rebuildPanel func()
	hotKeys := macos.NewHotKeyManager()
	app.OnQuit(func(context.Context) { hotKeys.Close() })
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
			openSettings = func() {
				model := newSettingsModel(store)
				model.dataDir = func() string {
					if runtime == nil {
						return ""
					}
					return runtime.paths.Root
				}
				model.changeDataDir = func(parent string) error {
					if runtime == nil {
						return fmt.Errorf("%s", i18n.T(i18n.ClipboardServiceDown))
					}
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
					defer cancel()
					_, err := runtime.changeDataDir(ctx, parent)
					return err
				}
				model.canRestoreDataDir = func() bool {
					if runtime == nil {
						return false
					}
					paths, err := storage.Resolve(runtime.paths.Base, storage.Mode())
					return err == nil && paths.Root != paths.Bootstrap
				}
				model.restoreDataDir = func() error {
					if runtime == nil {
						return fmt.Errorf("%s", i18n.T(i18n.ClipboardServiceDown))
					}
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
					defer cancel()
					_, err := runtime.restoreDefaultDataDir(ctx)
					return err
				}
				model.listApps = func() ([]database.App, error) {
					if runtime == nil || runtime.apps == nil {
						return nil, fmt.Errorf("%s", i18n.T(i18n.ClipboardServiceDown))
					}
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					return runtime.apps.List(ctx)
				}
				model.listGroups = func() ([]database.Group, error) {
					if runtime == nil {
						return nil, fmt.Errorf("%s", i18n.T(i18n.ClipboardServiceDown))
					}
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					return runtime.db.ListGroups(ctx)
				}
				model.addApp = func(path string) (database.App, error) {
					if runtime == nil || runtime.apps == nil {
						return database.App{}, fmt.Errorf("%s", i18n.T(i18n.ClipboardServiceDown))
					}
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()
					return runtime.apps.Add(ctx, path)
				}
				model.deleteApps = func(ids []string) error {
					if runtime == nil || runtime.apps == nil {
						return fmt.Errorf("%s", i18n.T(i18n.ClipboardServiceDown))
					}
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()
					_, err := runtime.apps.DeleteUnreferenced(ctx, ids)
					return err
				}
				model.appIconPath = func(name string) string {
					if runtime == nil {
						return ""
					}
					return runtime.appIcons.Path(name)
				}
				model.onShortcutsChanged = func(changed string) error {
					if reloadHotkeys == nil {
						return nil
					}
					err := reloadHotkeys()
					if err == nil || changed == "" || !hotKeys.HasErrors() {
						return err
					}
					key := hotKey
					if changed == "openPreference" {
						key = preferenceHotKey
					}
					return hotKeys.ErrorFor(key)
				}
				model.onShortcutsSuspended = func(suspend bool) {
					if suspend {
						_ = hotKeys.Set(nil)
						return
					}
					if reloadHotkeys != nil {
						_ = reloadHotkeys()
					}
				}
				model.onDisplayChanged = func() {
					if runtime != nil {
						runtime.Signal()
					}
				}
				model.onActionsChanged = func() {
					if runtime != nil {
						runtime.Signal()
					}
				}
				model.setLanguage = func(language string) error {
					i18n.SetLanguage(language)
					app.SetLocale(language)
					app.Post(func() {
						if settingsWindow != nil && !settingsWindow.Closed() {
							settingsWindow.Close()
							settingsWindow = nil
						}
						if rebuildPanel != nil {
							rebuildPanel()
						}
						macos.SelectSettingsTab("general")
						if openSettings != nil {
							openSettings()
						}
					})
					return nil
				}
				model.setTrayIcon = func(visible bool) error {
					if tray == nil {
						return fmt.Errorf("%s", i18n.T(i18n.ErrorTrayNotReady))
					}
					icon := []byte(nil)
					if visible {
						icon = trayIcon()
					}
					tray.SetIcon(icon, true)
					return nil
				}
				model.exportBackup = func() {
					exportBackupWithUI(app, runtime)
				}
				model.importBackup = func() {
					importBackupWithUI(app, runtime, store, reloadHotkeys)
				}
				model.restart = func() {
					app.Post(func() {
						macos.ShowMessage(i18n.T(i18n.DataDirChangedTitle), i18n.T(i18n.DataDirChangedBody))
						desktop.Restart(app)
					})
				}
				settingsWindow = desktop.OpenWindow(app, desktop.WindowOptions{
					Title:       i18n.T(i18n.SettingsWindowTitle),
					Width:       560,
					Height:      600,
					Key:         "settings",
					Constraints: core.WindowConstraints{NoResize: true, NoMaximize: true},
				}, ui.SettingsWindow(model))
			}
			openSettingsTab := func(tab string) {
				macos.SelectSettingsTab(tab)
				openSettings()
			}
			runtime, err = startClipboard(app, paths, store, &paused)
			if err != nil {
				log.Printf("start clipboard: %v", err)
			} else {
				history = newHistoryModel(runtime, store)
				if model, ok := history.(*historyModel); ok {
					model.openSettings = openSettingsTab
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
			Material: desktop.WindowMaterialPopover,
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
		if runtime != nil {
			runtime.Signal()
		}
		placePanel()
		panel.Show()
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
		Menu: func() []desktop.MenuItem {
			pauseLabel := i18n.T(i18n.TrayPause)
			if paused.Load() {
				pauseLabel = i18n.T(i18n.TrayResume)
			}
			return []desktop.MenuItem{
				{Label: i18n.T(i18n.TrayShow), Shortcut: hotKey, OnSelect: showPanel},
				{Label: pauseLabel, OnSelect: func() { paused.Store(!paused.Load()) }},
				{Label: i18n.T(i18n.TrayReadClipboard), OnSelect: func() {
					if runtime == nil {
						return
					}
					go func() {
						ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
						defer cancel()
						if err := runtime.readClipboard(ctx); err != nil {
							log.Printf("read clipboard: %v", err)
							return
						}
						runtime.Signal()
					}()
				}},
				{Label: i18n.T(i18n.TraySettings), Shortcut: preferenceHotKey, OnSelect: func() {
					if openSettings != nil {
						openSettings()
					}
				}},
				{Separator: true},
				{Label: i18n.T(i18n.TrayExportBackup), OnSelect: func() {
					exportBackupWithUI(app, runtime)
				}},
				{Label: i18n.T(i18n.TrayImportBackup), OnSelect: func() {
					importBackupWithUI(app, runtime, store, reloadHotkeys)
				}},
				{Label: i18n.T(i18n.TrayClearHistory), OnSelect: func() {
					choice := macos.ConfirmClearHistory()
					if choice == macos.ClearHistoryCancel || runtime == nil {
						return
					}
					_, err := runtime.service.Clear(context.Background(), choice == macos.ClearHistoryAll, choice == macos.ClearHistoryAll)
					if err != nil {
						log.Printf("clear history: %v", err)
						return
					}
					runtime.Signal()
				}},
				{Separator: true},
				{Label: i18n.T(i18n.TrayStorageUsage), OnSelect: func() {
					if runtime == nil {
						return
					}
					usage, err := runtime.storageUsage()
					if err != nil {
						macos.ShowMessage(i18n.T(i18n.StorageUsageFailed), err.Error())
						return
					}
					macos.ShowMessage(i18n.T(i18n.StorageUsageTitle),
						i18n.Tf(i18n.StorageUsageBody,
							formatBytes(usage.Database), formatBytes(usage.Resource),
							formatBytes(usage.Config), formatBytes(usage.Total)))
				}},
				{Label: i18n.T(i18n.TrayCleanCache), OnSelect: func() {
					if runtime == nil || !macos.Confirm(i18n.T(i18n.CleanCacheTitle),
						i18n.T(i18n.CleanCacheBody), i18n.T(i18n.CleanCacheConfirm)) {
						return
					}
					files, freed, err := runtime.cleanCache(context.Background())
					if err != nil {
						macos.ShowMessage(i18n.T(i18n.CleanCacheFailed), err.Error())
						return
					}
					if files == 0 {
						macos.ShowMessage(i18n.T(i18n.CleanCacheEmptyTitle), i18n.T(i18n.CleanCacheEmptyBody))
						return
					}
					macos.ShowMessage(i18n.T(i18n.CleanCacheDoneTitle),
						i18n.Tf(i18n.CleanCacheDoneBody, files, formatBytes(freed)))
				}},
				{Label: i18n.T(i18n.TrayOpenDataDir), OnSelect: func() {
					if runtime == nil {
						return
					}
					if err := macos.OpenPath(runtime.paths.Root); err != nil {
						macos.ShowMessage(i18n.T(i18n.OpenDataDirFailed), err.Error())
					}
				}},
				{Label: i18n.T(i18n.TrayOpenLogs), OnSelect: func() {
					if runtime == nil {
						return
					}
					if err := macos.OpenPath(runtime.paths.Logs); err != nil {
						macos.ShowMessage(i18n.T(i18n.OpenLogsFailed), err.Error())
					}
				}},
				{Separator: true},
				{Label: i18n.T(i18n.TraySource), OnSelect: func() {
					if err := macos.OpenURL(sourceURL); err != nil {
						log.Printf("open source: %v", err)
					}
				}},
				{Label: i18n.Tf(i18n.TrayVersion, macos.AppVersion()), Disabled: true},
				{Label: i18n.T(i18n.TrayRestart), OnSelect: func() { desktop.Restart(app) }},
				{Separator: true},
				{Label: i18n.T(i18n.TrayQuit), Shortcut: "mod+q", OnSelect: app.Quit},
			}
		},
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
	_ = reloadHotkeys()
	if os.Getenv("ECOPASTE_SHOW_SETTINGS") == "1" && openSettings != nil {
		openSettings()
	}

	if store != nil && !store.Snapshot().Onboarding.Completed {
		showPanel()
	} else if os.Getenv("ECOPASTE_SHOW_ON_LAUNCH") == "1" {
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

type clipboardRuntime struct {
	db            *database.DB
	service       *clipboard.Service
	images        *storage.ImageStore
	appIcons      *storage.AppIconStore
	fileIcons     *storage.FileIconStore
	fileIconCache *fileIconCache
	paths         *storage.Paths
	apps          *appRegistry
	watcher       *clipboard.Watcher
	changed       chan struct{}
}

func (r *clipboardRuntime) Signal() {
	select {
	case r.changed <- struct{}{}:
	default:
	}
}

// readClipboard captures what is on the pasteboard now, even while the
// automatic watcher is paused.
func (r *clipboardRuntime) readClipboard(ctx context.Context) error {
	_, err := r.watcher.Poll(ctx, true)
	return err
}

func startClipboard(app *core.Scope, paths *storage.Paths, store *settings.Store, paused *atomic.Bool) (*clipboardRuntime, error) {
	db, err := database.Open(filepath.Join(paths.DB, "clipboard.db"))
	if err != nil {
		return nil, err
	}
	images := storage.NewImageStore(paths.Resources)
	appIcons := storage.NewAppIconStore(paths.Resources)
	pruneAppIcons(db, appIcons)
	fileIcons := storage.NewFileIconStore(paths.Resources)
	pruneFileIcons(db, fileIcons)
	iconCache := newAppIconCache(appIcons)
	registry := newAppRegistry(db, appIcons, iconCache)
	fileIconCache := newFileIconCache(db, fileIcons)
	changed := make(chan struct{}, 1)
	service := &clipboard.Service{
		DB:       db,
		Images:   images,
		Settings: func() settings.Clipboard { return store.Snapshot().Clipboard },
		Guard:    clipboard.NewGuard(2 * time.Second),
		Writer:   pasteboardWriter{},
	}
	watcher := &clipboard.Watcher{
		Source: pasteboardSource{icons: iconCache},
		Paused: paused.Load,
		Capture: func(ctx context.Context, payload clipboard.Payload, source clipboard.SourceApp) {
			_, captured, err := service.Capture(ctx, payload, source)
			if err != nil {
				log.Printf("capture clipboard: %v", err)
			} else if captured {
				if store.Snapshot().Clipboard.Feedback.CopySound {
					macos.PlayCopySound()
				}
				select {
				case changed <- struct{}{}:
				default:
				}
			}
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		watcher.Run(ctx)
		close(done)
	}()
	runtime := &clipboardRuntime{
		db: db, service: service, images: images, appIcons: appIcons,
		fileIcons: fileIcons, fileIconCache: fileIconCache, paths: paths,
		apps: registry, watcher: watcher, changed: changed,
	}
	startCleanup(ctx, runtime, store)
	app.OnQuit(func(context.Context) {
		cancel()
		select {
		case <-done:
		case <-time.After(150 * time.Millisecond):
		}
		db.Close()
	})
	return runtime, nil
}

func pruneAppIcons(db *database.DB, store *storage.AppIconStore) {
	referenced, err := db.AppIconFiles(context.Background())
	if err != nil {
		log.Printf("list app icons: %v", err)
		return
	}
	if _, _, err := store.RemoveUnreferenced(referenced); err != nil {
		log.Printf("clean app icons: %v", err)
	}
}

func pruneFileIcons(db *database.DB, store *storage.FileIconStore) {
	referenced, err := db.FileIconFiles(context.Background())
	if err != nil {
		log.Printf("list file icons: %v", err)
		return
	}
	if _, _, err := store.RemoveUnreferenced(referenced); err != nil {
		log.Printf("clean file icons: %v", err)
	}
}
