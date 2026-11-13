package app

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/clipboard"
	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
	"github.com/EcoPasteHub/EcoPaste/internal/storage"
	"github.com/EcoPasteHub/EcoPaste/internal/ui"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
)

const panelHotKey = "alt+c"

// Run opens EcoPaste's panel, tray icon and global shortcut. The app
// keeps running without windows until the user quits it.
func Run(app *core.Scope) {
	desktop.SetShutdownMode(app, desktop.ShutdownOnExplicit)
	hotKey := panelHotKey
	var paused atomic.Bool
	var history ui.HistoryModel
	var runtime *clipboardRuntime
	if paths, err := storage.Default(); err != nil {
		log.Printf("resolve storage: %v", err)
	} else {
		log.Printf("data root %s", paths.Root)
		store, err := settings.Open(filepath.Join(paths.Config, settings.FileName))
		if err != nil {
			log.Printf("open settings: %v", err)
		} else {
			if configured := store.Snapshot().Shortcuts.OpenClipboard; configured != "" {
				hotKey = strings.ToLower(configured)
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
	content := ui.ClipboardPanel(history, func() { panel.Hide() })
	panel = desktop.OpenPanel(app, desktop.PanelOptions{
		Width:      420,
		Height:     560,
		HideOnBlur: true,
		Material:   desktop.WindowMaterialPopover,
	}, content)

	showPanel := func() {
		if runtime != nil {
			runtime.Signal()
		}
		panel.Show()
	}
	togglePanel := func() {
		if !panel.Visible() && runtime != nil {
			runtime.Signal()
		}
		panel.Toggle()
	}

	desktop.NewTray(app, desktop.TrayOptions{
		Icon:     trayIcon(),
		Template: true,
		Tooltip:  "EcoPaste",
		Panel:    panel,
		Menu: func() []desktop.MenuItem {
			pauseLabel := "Pause Capture"
			if paused.Load() {
				pauseLabel = "Resume Capture"
			}
			return []desktop.MenuItem{
				{Label: "Show EcoPaste", Shortcut: hotKey, OnSelect: showPanel},
				{Label: pauseLabel, OnSelect: func() { paused.Store(!paused.Load()) }},
				{Separator: true},
				{Label: "Quit EcoPaste", Shortcut: "mod+q", OnSelect: app.Quit},
			}
		},
	})

	desktop.RegisterHotKey(app, hotKey, togglePanel)

	if os.Getenv("ECOPASTE_SHOW_ON_LAUNCH") == "1" {
		panel.Show()
	}
}

type clipboardRuntime struct {
	db      *database.DB
	service *clipboard.Service
	images  *storage.ImageStore
	changed chan struct{}
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
	changed := make(chan struct{}, 1)
	service := &clipboard.Service{
		DB:       db,
		Images:   images,
		Settings: func() settings.Clipboard { return store.Snapshot().Clipboard },
		Guard:    clipboard.NewGuard(2 * time.Second),
		Writer:   pasteboardWriter{},
	}
	watcher := &clipboard.Watcher{
		Source: pasteboardSource{},
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
	app.OnQuit(func(context.Context) {
		cancel()
		select {
		case <-done:
		case <-time.After(150 * time.Millisecond):
		}
		db.Close()
	})
	return &clipboardRuntime{db: db, service: service, images: images, changed: changed}, nil
}
