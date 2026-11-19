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
	responder := p.table
	if p.behavior.DefaultFocus {
		responder = p.search
	}
	window.Send(sel("makeFirstResponder:"), responder)
}

func panelWindowResignedKey(self objc.ID, _ objc.SEL, note objc.ID) {
	p := panelsByObserver[self]
	if p == nil || suppressPanelBlur.Load() || p.pinned || p.dragging || p.previewing ||
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
	p := panelsByObserver[self]
	if p != nil {
		p.hideHover()
	}
	if p != nil && p.shortcutPopover != 0 &&
		objc.Send[bool](p.shortcutPopover, sel("isShown")) {
		p.shortcutPopover.Send(sel("close"))
	}
	if p != nil && p.groupPopoverActive() {
		if _, window := panelForWindow(self, note); window != 0 {
			p.groupClosing = true
			p.groupPopover.Send(sel("close"))
		}
	}
	if p != nil && p.notePopoverActive() {
		if _, window := panelForWindow(self, note); window != 0 {
			p.noteClosing = true
			p.notePopover.Send(sel("close"))
		}
	}
	if p == nil || !p.behavior.ClearOnHide {
		return
	}
	p, _ = panelForWindow(self, note)
	if p == nil {
		return
	}
	p.clearSearch()
	syncNavVisibility(p)
	p.cx.Emit("searchCleared", nil)
}

func panelWindowDidOrderOnScreen(self objc.ID, _ objc.SEL, note objc.ID) {
	p, _ := panelForWindow(self, note)
	if p == nil {
		return
	}
	applyPlacement(p)
	syncNavVisibility(p)
	animateIn(p.view, panelFadeDuration)
	p.cx.Emit("opened", nil)
}

func panelSearchTextChanged(_ objc.ID, _ objc.SEL, note objc.ID) {
	panelApplySearch(objc.Send[objc.ID](note, sel("object")))
}

// panelScrollBoundsChanged reacts to the clip view scrolling: it keeps the
// hover bar on the row under the pointer and asks for the next page when the
// bottom nears the loaded rows.
func panelScrollBoundsChanged(self objc.ID, _ objc.SEL, _ objc.ID) {
	p := panelsByObserver[self]
	if p == nil {
		return
	}
	p.refreshHoverFromMouse()
	p.checkLoadMore()
}
