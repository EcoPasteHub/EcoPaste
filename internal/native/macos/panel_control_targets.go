//go:build darwin

package macos

import (
	"github.com/ebitengine/purego/objc"
)

func registerPanelControlTargetClass() error {
	target, err := objc.RegisterClass("EcoPasteSearchTarget", class("NSObject"), nil, nil, panelControlMethods())
	if err != nil {
		return err
	}
	searchTarget = objc.ID(target).Send(sel("new"))
	return nil
}

func panelControlMethods() []objc.MethodDef {
	return []objc.MethodDef{
		{Cmd: sel("searchChanged:"), Fn: panelSearchChanged},
		{Cmd: sel("pinPressed:"), Fn: panelPinPressed},
		{Cmd: sel("filterPressed:"), Fn: panelFilterPressed},
		{Cmd: sel("filterMenuPressed:"), Fn: panelFilterMenuPressed},
		{Cmd: sel("settingsPressed:"), Fn: panelSettingsPressed},
		{Cmd: sel("clearPressed:"), Fn: panelClearPressed},
		{Cmd: sel("elevateDialog:"), Fn: panelElevateDialog},
		{Cmd: sel("shortcutHelpPressed:"), Fn: panelShortcutHelpPressed},
		{Cmd: sel("capsulePressed:"), Fn: panelCapsulePressed},
		{Cmd: sel("navPlusPressed:"), Fn: panelNavPlusPressed},
		{Cmd: sel("hoverActionPressed:"), Fn: panelHoverActionPressed},
		{Cmd: sel("menuNewGroup:"), Fn: panelMenuNewGroup},
		{Cmd: sel("menuRenameGroup:"), Fn: panelMenuRenameGroup},
		{Cmd: sel("menuDeleteGroup:"), Fn: panelMenuDeleteGroup},
	}
}

func panelSearchChanged(_ objc.ID, _ objc.SEL, sender objc.ID) {
	if p := panelsBySearch[sender]; p != nil {
		next := goString(sender.Send(sel("stringValue")))
		if next != p.query {
			p.query = next
			p.markSelectionFirst()
		}
		p.cx.Emit("query", next)
	}
}

func panelPinPressed(_ objc.ID, _ objc.SEL, sender objc.ID) {
	if p := panelsByControl[sender]; p != nil {
		p.togglePinned()
	}
}

func panelFilterPressed(_ objc.ID, _ objc.SEL, sender objc.ID) {
	if p := panelsByControl[sender]; p != nil {
		p.showFilterMenu(sender)
	}
}

func panelFilterMenuPressed(_ objc.ID, _ objc.SEL, sender objc.ID) {
	if p := menuPanel(sender); p != nil {
		p.applyFilterMenuItem(int(objc.Send[int64](sender, sel("tag"))))
	}
}

func panelSettingsPressed(_ objc.ID, _ objc.SEL, sender objc.ID) {
	if p := panelsByControl[sender]; p != nil {
		p.cx.Emit("openSettings", "")
	}
}

func panelClearPressed(_ objc.ID, _ objc.SEL, sender objc.ID) {
	if p := panelsByControl[sender]; p != nil {
		p.cx.Emit("clearHistory", nil)
	}
}

func panelElevateDialog(_ objc.ID, _ objc.SEL, timer objc.ID) {
	window := objc.Send[objc.ID](timer, sel("userInfo"))
	if window == 0 {
		return
	}
	window.Send(sel("setLevel:"), int64(1000))
	placeDialogOverPanel(window)
	window.Send(sel("orderFront:"), objc.ID(0))
}

func panelShortcutHelpPressed(_ objc.ID, _ objc.SEL, sender objc.ID) {
	if p := panelsByControl[sender]; p != nil {
		p.toggleShortcutHelp()
	}
}

func panelCapsulePressed(_ objc.ID, _ objc.SEL, sender objc.ID) {
	nav := navsByCapsule[sender]
	if nav == nil {
		return
	}
	index := int(objc.Send[int64](sender, sel("tag")))
	if index < 0 || index >= len(panelNavFilters) {
		return
	}
	nav.applyFilter(panelNavFilters[index])
	nav.panel.cx.Emit("filter", panelNavFilters[index])
}

func panelNavPlusPressed(_ objc.ID, _ objc.SEL, sender objc.ID) {
	if nav := navsByPlus[sender]; nav != nil && nav.panel != nil {
		nav.panel.showNewGroupPopover(nav.plus)
	}
}

func panelHoverActionPressed(_ objc.ID, _ objc.SEL, sender objc.ID) {
	if p := panelsByControl[sender]; p != nil {
		p.runHoverAction(int(objc.Send[int64](sender, sel("tag"))))
	}
}

func panelMenuNewGroup(_ objc.ID, _ objc.SEL, sender objc.ID) {
	if p := menuPanel(sender); p != nil {
		p.showNewGroupPopover(p.filterButton)
	}
}

func panelMenuRenameGroup(_ objc.ID, _ objc.SEL, sender objc.ID) {
	if p := menuPanel(sender); p != nil && p.groupID != "" {
		p.cx.Emit("renameGroup", p.groupID)
	}
}

func panelMenuDeleteGroup(_ objc.ID, _ objc.SEL, sender objc.ID) {
	if p := menuPanel(sender); p != nil && p.groupID != "" {
		p.cx.Emit("deleteGroup", p.groupID)
	}
}
