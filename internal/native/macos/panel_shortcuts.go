//go:build darwin

package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

const (
	shortcutHelpWidth   = 300.0
	shortcutHelpPadding = 16.0
	shortcutHelpHeader  = 32.0
	shortcutHelpRow     = 24.0
)

type shortcutHelpItem struct {
	Label string
	Keys  string
}

func shortcutHelpItems() []shortcutHelpItem {
	return []shortcutHelpItem{
		{i18n.T(i18n.MenuPaste), "↵"},
		{i18n.T(i18n.MenuPastePlain), "⌘↩"},
		{i18n.T(i18n.PanelShortcutPasteNth), "⌘1–0"},
		{i18n.T(i18n.PanelShortcutPreview), "Space"},
		{i18n.T(i18n.MenuCopy), "⌘C"},
		{i18n.T(i18n.PanelShortcutOpen), "⌘O"},
		{i18n.T(i18n.PanelShortcutFavorite), "⌘D"},
		{i18n.T(i18n.MenuPin), "⌘T"},
		{i18n.T(i18n.PanelShortcutNote), "⌘M"},
		{i18n.T(i18n.CommonDelete), "⌫"},
		{i18n.T(i18n.PanelShortcutNavigate), "↑ ↓"},
		{i18n.T(i18n.PanelShortcutSearch), "⌘F"},
		{i18n.T(i18n.PanelPinWindow), "⌘P"},
		{i18n.T(i18n.PanelNewGroup), "⌘N"},
		{i18n.T(i18n.PanelShortcutClose), "Esc"},
	}
}

func shortcutHelpHeight() float64 {
	return shortcutHelpHeader + shortcutHelpRow*float64(len(shortcutHelpItems())) + shortcutHelpPadding
}

// buildShortcutHelp creates the keyboard cheat sheet shown by Command-K
// and the footer keyboard button.
func (p *Panel) buildShortcutHelp() {
	view := objc.ID(class("NSView")).Send(sel("new"))
	view.Send(sel("setAutoresizesSubviews:"), false)
	p.shortcutView = view

	title := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(i18n.T(i18n.PanelShortcutsTitle)))
	title.Send(sel("setFont:"), objc.ID(classNSFont).Send(sel("boldSystemFontOfSize:"), 13.0))
	title.Send(sel("setLineBreakMode:"), int64(4))
	title.Send(sel("setMaximumNumberOfLines:"), int64(1))
	view.Send(sel("addSubview:"), title)
	p.shortcutTitle = title

	for range shortcutHelpItems() {
		label := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
		label.Send(sel("setFont:"), systemFont(12))
		label.Send(sel("setLineBreakMode:"), int64(4))
		label.Send(sel("setMaximumNumberOfLines:"), int64(1))
		view.Send(sel("addSubview:"), label)
		p.shortcutLabels = append(p.shortcutLabels, label)

		keys := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
		keys.Send(sel("setFont:"), systemFont(11))
		keys.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
		keys.Send(sel("setAlignment:"), int64(2))
		keys.Send(sel("setLineBreakMode:"), int64(4))
		keys.Send(sel("setMaximumNumberOfLines:"), int64(1))
		view.Send(sel("addSubview:"), keys)
		p.shortcutKeys = append(p.shortcutKeys, keys)
	}

	controller := objc.ID(class("NSViewController")).Send(sel("new"))
	controller.Send(sel("setView:"), view)
	p.shortcutController = controller

	p.shortcutPopover = objc.ID(class("NSPopover")).Send(sel("new"))
	p.shortcutPopover.Send(sel("setBehavior:"), int64(1))
	p.shortcutPopover.Send(sel("setAnimates:"), true)
	p.shortcutPopover.Send(sel("setContentViewController:"), controller)
}

func (p *Panel) refreshShortcutHelp() {
	items := shortcutHelpItems()
	width := shortcutHelpWidth
	height := shortcutHelpHeight()
	setFrame(p.shortcutView, 0, 0, width, height)
	setFrame(p.shortcutTitle, shortcutHelpPadding, height-shortcutHelpPadding-17,
		width-2*shortcutHelpPadding, 17)
	for index, item := range items {
		y := height - shortcutHelpHeader - shortcutHelpRow*float64(index+1)
		setFrame(p.shortcutLabels[index], shortcutHelpPadding, y,
			width-2*shortcutHelpPadding-70, 17)
		setFrame(p.shortcutKeys[index], width-shortcutHelpPadding-70, y, 70, 17)
		p.shortcutLabels[index].Send(sel("setStringValue:"), nsString(item.Label))
		p.shortcutKeys[index].Send(sel("setStringValue:"), nsString(item.Keys))
	}
	p.shortcutPopover.Send(sel("setContentSize:"), CGSize{Width: width, Height: height})
}

// toggleShortcutHelp shows or hides the cheat sheet next to the footer
// keyboard button.
func (p *Panel) toggleShortcutHelp() {
	if p.shortcutPopover != 0 && objc.Send[bool](p.shortcutPopover, sel("isShown")) {
		p.shortcutPopover.Send(sel("close"))
		return
	}
	p.refreshShortcutHelp()
	anchor := bounds(p.shortcutButton)
	rect := CGRect{
		Origin: CGPoint{X: anchor.Size.Width - 1, Y: anchor.Size.Height / 2},
		Size:   CGSize{Width: 1, Height: 1},
	}
	p.shortcutPopover.Send(sel("showRelativeToRect:ofView:preferredEdge:"),
		rect, p.shortcutButton, uint64(3))
}
