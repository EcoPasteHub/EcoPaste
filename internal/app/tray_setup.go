package app

import (
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
)

func newAppTray(app *core.Scope, panel *desktop.Window, runtime *clipboardRuntime,
	store *settings.Store, showPanel, openSettings func(),
	hotKeys *hotKeySet) *desktop.Tray {
	visible := settings.Default().General.TrayIcon
	if store != nil {
		visible = store.Snapshot().General.TrayIcon
	}
	var image []byte
	if visible {
		image = trayIcon()
	}
	return desktop.NewTray(app, desktop.TrayOptions{
		Icon:     image, // the pre-refactor "p" mark, tinted like system icons
		Template: true,
		Tooltip:  "EcoPaste",
		Panel:    panel,
		Menu: trayMenu{
			app:       app,
			runtime:   runtime,
			store:     store,
			showPanel: showPanel,
			openSettings: func() {
				if openSettings != nil {
					openSettings()
				}
			},
			shortcuts:     hotKeys.shortcuts,
			reloadHotkeys: hotKeys.reload,
		}.items,
	})
}
