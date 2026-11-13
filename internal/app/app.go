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

	"github.com/EcoPasteHub/EcoPaste/internal/clipboard"
	"github.com/EcoPasteHub/EcoPaste/internal/database"
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
	desktop.SetShutdownMode(app, desktop.ShutdownOnExplicit)
	activation := instanceRequests(app)
	hotKey := panelHotKey
	preferenceHotKey := "alt+x"
	var paused atomic.Bool
	var history ui.HistoryModel
	var runtime *clipboardRuntime
	var openSettings func()
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
				desktop.OpenWindow(app, desktop.WindowOptions{
					Title:       "EcoPaste 设置",
					Width:       560,
					Height:      560,
					Key:         "settings",
					Constraints: core.WindowConstraints{NoResize: true, NoMaximize: true},
				}, ui.SettingsWindow(newSettingsModel(store)))
			}
			runtime, err = startClipboard(app, paths, store, &paused)
			if err != nil {
				log.Printf("start clipboard: %v", err)
			} else {
				history = newHistoryModel(runtime, store)
			}
		}
	}
	if custom := os.Getenv("ECOPASTE_HOTKEY"); custom != "" {
		hotKey = custom
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
	content := ui.ClipboardPanel(history, func() {
		savePanelFrame()
		panel.Hide()
	})
	panel = desktop.OpenPanel(app, desktop.PanelOptions{
		Width:    420,
		Height:   560,
		Material: desktop.WindowMaterialPopover,
	}, content)

	showPanel := func() {
		if runtime != nil {
			runtime.Signal()
		}
		placePanel()
		panel.Show()
	}
	togglePanel := func() {
		if !panel.Visible() {
			if runtime != nil {
				runtime.Signal()
			}
			placePanel()
		}
		panel.Toggle()
	}

	desktop.NewTray(app, desktop.TrayOptions{
		Icon:     trayIcon(),
		Template: true,
		Tooltip:  "EcoPaste",
		Panel:    panel,
		Menu: func() []desktop.MenuItem {
			pauseLabel := "暂停记录"
			if paused.Load() {
				pauseLabel = "继续记录"
			}
			return []desktop.MenuItem{
				{Label: "显示 EcoPaste", Shortcut: hotKey, OnSelect: showPanel},
				{Label: pauseLabel, OnSelect: func() { paused.Store(!paused.Load()) }},
				{Label: "设置…", Shortcut: preferenceHotKey, OnSelect: func() {
					if openSettings != nil {
						openSettings()
					}
				}},
				{Separator: true},
				{Label: "清除历史…", OnSelect: func() {
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
				{Label: "存储占用…", OnSelect: func() {
					if runtime == nil {
						return
					}
					usage, err := runtime.storageUsage()
					if err != nil {
						macos.ShowMessage("无法读取存储占用", err.Error())
						return
					}
					macos.ShowMessage("存储占用",
						fmt.Sprintf("数据库 %s · 资源 %s · 设置 %s\n总计 %s",
							formatBytes(usage.Database), formatBytes(usage.Resource),
							formatBytes(usage.Config), formatBytes(usage.Total)))
				}},
				{Label: "清理未引用缓存…", OnSelect: func() {
					if runtime == nil || !macos.Confirm("清理未引用缓存？",
						"仅删除未被历史记录引用的图片和来源应用图标。", "清理") {
						return
					}
					files, freed, err := runtime.cleanCache(context.Background())
					if err != nil {
						macos.ShowMessage("清理缓存失败", err.Error())
						return
					}
					if files == 0 {
						macos.ShowMessage("缓存已是最新", "没有需要清理的文件。")
						return
					}
					macos.ShowMessage("清理完成",
						fmt.Sprintf("已删除 %d 个文件，释放 %s。", files, formatBytes(freed)))
				}},
				{Label: "打开数据目录", OnSelect: func() {
					if runtime == nil {
						return
					}
					if err := macos.OpenPath(runtime.paths.Root); err != nil {
						macos.ShowMessage("无法打开数据目录", err.Error())
					}
				}},
				{Label: "打开日志目录", OnSelect: func() {
					if runtime == nil {
						return
					}
					if err := macos.OpenPath(runtime.paths.Logs); err != nil {
						macos.ShowMessage("无法打开日志目录", err.Error())
					}
				}},
				{Separator: true},
				{Label: "开源地址", OnSelect: func() {
					if err := macos.OpenURL(sourceURL); err != nil {
						log.Printf("open source: %v", err)
					}
				}},
				{Label: "版本 " + macos.AppVersion(), Disabled: true},
				{Label: "重新启动", OnSelect: func() { desktop.Restart(app) }},
				{Separator: true},
				{Label: "退出 EcoPaste", Shortcut: "mod+q", OnSelect: app.Quit},
			}
		},
	})

	desktop.RegisterHotKey(app, hotKey, togglePanel)
	if openSettings != nil && preferenceHotKey != "" {
		desktop.RegisterHotKey(app, preferenceHotKey, openSettings)
	}
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
	changed       chan struct{}
}

func (r *clipboardRuntime) Signal() {
	select {
	case r.changed <- struct{}{}:
	default:
	}
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
		Capture: func(ctx context.Context, payload clipboard.Payload, source clipboard.SourceApp) {
			if paused.Load() {
				return
			}
			_, captured, err := service.Capture(ctx, payload, source)
			if err != nil {
				log.Printf("capture clipboard: %v", err)
			} else if captured {
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
		fileIcons: fileIcons, fileIconCache: fileIconCache, paths: paths, changed: changed,
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
