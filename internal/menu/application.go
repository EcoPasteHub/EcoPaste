// Package menu builds EcoPaste's native application menu bar.
package menu

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/nsgo-dev/nsgo/desktop"
	"github.com/nsgo-dev/nsgo/desktop/locales"
)

// Application returns the menu bar in language. Settings opens the
// preference window and Check for Updates starts an update check; each
// is left out when nil.
func Application(language string, settings, checkUpdates func()) []desktop.MenuItem {
	msgs := locales.In(language)
	appItems := []desktop.MenuItem{
		{Role: desktop.MenuAbout},
		{Separator: true},
	}
	if checkUpdates != nil {
		appItems = append(appItems,
			desktop.MenuItem{Label: i18n.T(i18n.TrayCheckUpdates), OnSelect: checkUpdates})
	}
	if settings != nil {
		appItems = append(appItems,
			desktop.MenuItem{Label: i18n.T(i18n.TraySettings), Shortcut: "meta+,", OnSelect: settings},
		)
	}
	if checkUpdates != nil || settings != nil {
		appItems = append(appItems, desktop.MenuItem{Separator: true})
	}
	appItems = append(appItems,
		desktop.MenuItem{Role: desktop.MenuHide},
		desktop.MenuItem{Role: desktop.MenuHideOthers},
		desktop.MenuItem{Role: desktop.MenuShowAll},
		desktop.MenuItem{Separator: true},
		desktop.MenuItem{Role: desktop.MenuQuit},
	)
	return []desktop.MenuItem{
		{Items: appItems},
		{Label: msgs.Edit(), Items: []desktop.MenuItem{
			{Role: desktop.MenuUndo},
			{Role: desktop.MenuRedo},
			{Separator: true},
			{Role: desktop.MenuCut},
			{Role: desktop.MenuCopy},
			{Role: desktop.MenuPaste},
			{Role: desktop.MenuSelectAll},
		}},
		{Label: msgs.Window(), Items: []desktop.MenuItem{
			{Role: desktop.MenuMinimize},
			{Role: desktop.MenuZoom},
			{Separator: true},
			{Role: desktop.MenuToggleFullScreen},
			{Separator: true},
			{Role: desktop.MenuClose},
		}},
	}
}
