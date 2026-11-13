//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

func panelObserverMethods() []objc.MethodDef {
	return []objc.MethodDef{
		{Cmd: sel("windowBecameKey:"), Fn: func(self objc.ID, _ objc.SEL, note objc.ID) {
			p := panelsByObserver[self]
			if p == nil {
				return
			}
			window := p.view.Send(sel("window"))
			if window == 0 || window != objc.Send[objc.ID](note, sel("object")) {
				return
			}
			p.refreshOnboardingPermissions()
			responder := p.table
			if p.behavior.DefaultFocus {
				responder = p.search
			}
			window.Send(sel("makeFirstResponder:"), responder)
		}},
		{Cmd: sel("windowResignedKey:"), Fn: func(self objc.ID, _ objc.SEL, note objc.ID) {
			p := panelsByObserver[self]
			if p == nil || suppressPanelBlur.Load() || p.pinned || p.dragging || p.onboardingActive {
				return
			}
			window := p.view.Send(sel("window"))
			if window != 0 && window == objc.Send[objc.ID](note, sel("object")) &&
				objc.Send[bool](window, sel("isVisible")) {
				p.cx.Emit("blurHide", nil)
			}
		}},
		{Cmd: sel("windowDidOrderOffScreen:"), Fn: func(self objc.ID, _ objc.SEL, note objc.ID) {
			p := panelsByObserver[self]
			if p == nil || !p.behavior.ClearOnHide {
				return
			}
			window := p.view.Send(sel("window"))
			if window != 0 && window == objc.Send[objc.ID](note, sel("object")) {
				p.clearSearch()
				p.cx.Emit("searchCleared", nil)
			}
		}},
		{Cmd: sel("windowDidOrderOnScreen:"), Fn: func(self objc.ID, _ objc.SEL, note objc.ID) {
			p := panelsByObserver[self]
			if p == nil {
				return
			}
			window := p.view.Send(sel("window"))
			if window != 0 && window == objc.Send[objc.ID](note, sel("object")) {
				applyPlacement(p)
				p.cx.Emit("opened", nil)
			}
		}},
		{Cmd: sel("pollSearch:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
			p := panelsByObserver[self]
			if p == nil {
				return
			}
			next := goString(p.search.Send(sel("stringValue")))
			if next != p.query {
				p.query = next
				p.cx.Emit("query", next)
			}
			p.pollHover()
		}},
	}
}
