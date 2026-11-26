//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

func registerPanelObserverClass() error {
	var err error
	observerClass, err = objc.RegisterClass("EcoPastePanelObserver", class("NSObject"), nil, nil, panelObserverMethods())
	return err
}

func panelObserverMethods() []objc.MethodDef {
	return []objc.MethodDef{
		{Cmd: sel("windowBecameKey:"), Fn: panelWindowBecameKey},
		{Cmd: sel("windowResignedKey:"), Fn: panelWindowResignedKey},
		{Cmd: sel("windowDidOrderOffScreen:"), Fn: panelWindowDidOrderOffScreen},
		{Cmd: sel("windowDidOrderOnScreen:"), Fn: panelWindowDidOrderOnScreen},
		{Cmd: sel("searchTextChanged:"), Fn: panelSearchTextChanged},
		{Cmd: sel("scrollBoundsChanged:"), Fn: panelScrollBoundsChanged},
	}
}

// panelForWindow returns the panel observing the notification when its
// window is the one named by the notification.
func panelForWindow(self, note objc.ID) (*Panel, objc.ID) {
	p := panelsByObserver[self]
	if p == nil {
		return nil, 0
	}
	window := p.view.Send(sel("window"))
	if window == 0 || window != objc.Send[objc.ID](note, sel("object")) {
		return nil, 0
	}
	return p, window
}

func panelWindowBecameKey(self objc.ID, _ objc.SEL, note objc.ID) {
	p, window := panelForWindow(self, note)
	if p == nil {
		return
	}
	// A popover closing can hand key focus back while Command is still down, so
	// re-sync the hints with the live modifier state instead of waiting for the
	// next flag change (which would not fire until Command is pressed again).
	flags := uint64(objc.Send[uint](objc.ID(class("NSEvent")), sel("modifierFlags")))
	p.setCommandHeld(flags&commandModifier != 0)
	// An open popover owns keyboard focus. On the first summon the app becomes
	// active and this panel-key notification can arrive after the popover opened;
	// moving first responder back to the list would steal the caret from the
	// popover. Keep the popover focused instead: the guarded focus helper re-keys
	// the popover window and focuses its input if that is still pending.
	if p.notePopoverActive() {
		p.focusNoteArea()
		return
	}
	if p.groupPopoverActive() {
		p.focusGroupInput()
		return
	}
	if p.previewing || (p.shortcutPopover != 0 && objc.Send[bool](p.shortcutPopover, sel("isShown"))) {
		return
	}
	// The search field always takes focus when the panel opens, so typing
	// filters immediately; arrow keys then move into the list.
	window.Send(sel("makeFirstResponder:"), p.search)
}

func panelWindowResignedKey(self objc.ID, _ objc.SEL, note objc.ID) {
	p := panelsByObserver[self]
	if p == nil {
		return
	}
	// Command-hold hints belong to the focused panel; drop them when key focus
	// leaves so they cannot stay stuck on screen.
	p.setCommandHeld(false)
	if suppressPanelBlur.Load() || p.pinned || p.dragging || p.previewing ||
		p.groupPopoverActive() || p.notePopoverActive() {
		return
	}
	p, window := panelForWindow(self, note)
	if p == nil || !objc.Send[bool](window, sel("isVisible")) {
		return
	}
	syncNavVisibility(p)
	p.cx.Emit("blurHide", nil)
}

func panelWindowDidOrderOffScreen(self objc.ID, _ objc.SEL, note objc.ID) {
	p, _ := panelForWindow(self, note)
	if p == nil {
		return
	}
	p.setCommandHeld(false)
	p.cancelInteractions()
	if p.shortcutPopover != 0 &&
		objc.Send[bool](p.shortcutPopover, sel("isShown")) {
		p.shortcutPopover.Send(sel("close"))
	}
	if p.groupPopoverActive() {
		p.groupClosing = true
		p.groupPopover.Send(sel("close"))
	}
	if p.notePopoverActive() {
		p.noteClosing = true
		p.notePopover.Send(sel("close"))
	}
	if !p.behavior.ClearOnHide {
		return
	}
	p.clearSearch()
	syncNavVisibility(p)
	p.cx.Emit("searchCleared", nil)
}

func (p *Panel) cancelInteractions() {
	cancelPendingClick(p.table)
	p.pendingClick = ""
	endPanelDrag(p)
	p.hideHover()
}

func panelWindowDidOrderOnScreen(self objc.ID, _ objc.SEL, note objc.ID) {
	p, _ := panelForWindow(self, note)
	if p == nil {
		return
	}
	applyPlacement(p)
	syncNavVisibility(p)
	animateIn(p.view, panelFadeDuration)
}

func panelSearchTextChanged(_ objc.ID, _ objc.SEL, note objc.ID) {
	panelApplySearch(objc.Send[objc.ID](note, sel("object")))
}

// panelScrollBoundsChanged reacts to the clip view scrolling: it keeps the
// hovered-row highlight on the row under the pointer and asks for the next page
// when the bottom nears the loaded rows.
func panelScrollBoundsChanged(self objc.ID, _ objc.SEL, _ objc.ID) {
	p := panelsByObserver[self]
	if p == nil {
		return
	}
	p.refreshHoverFromMouse()
	p.checkLoadMore()
	if p.commandHeld {
		// The Command-digit hints are numbered from the first visible row, so a
		// scroll while Command is held has to renumber the cards on screen.
		p.refreshRowHints()
	}
}
