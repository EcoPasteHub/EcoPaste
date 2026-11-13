package app

import (
	"log"
	"os"
	"path/filepath"
	"strings"

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
	if paths, err := storage.Default(); err != nil {
		log.Printf("resolve storage: %v", err)
	} else {
		log.Printf("data root %s", paths.Root)
		if store, err := settings.Open(filepath.Join(paths.Config, settings.FileName)); err != nil {
			log.Printf("open settings: %v", err)
		} else if configured := store.Snapshot().Shortcuts.OpenClipboard; configured != "" {
			hotKey = strings.ToLower(configured)
		}
	}
	if custom := os.Getenv("ECOPASTE_HOTKEY"); custom != "" {
		hotKey = custom
	}

	panel := desktop.OpenPanel(app, desktop.PanelOptions{
		Width:      420,
		Height:     560,
		HideOnBlur: true,
		Material:   desktop.WindowMaterialPopover,
	}, ui.ClipboardPanel)

	desktop.NewTray(app, desktop.TrayOptions{
		Icon:     trayIcon(),
		Template: true,
		Tooltip:  "EcoPaste",
		Panel:    panel,
		Menu: func() []desktop.MenuItem {
			return []desktop.MenuItem{
				{Label: "Show EcoPaste", Shortcut: hotKey, OnSelect: panel.Show},
				{Separator: true},
				{Label: "Quit EcoPaste", Shortcut: "mod+q", OnSelect: app.Quit},
			}
		},
	})

	desktop.RegisterHotKey(app, hotKey, panel.Toggle)

	if os.Getenv("ECOPASTE_SHOW_ON_LAUNCH") == "1" {
		panel.Show()
	}
}
