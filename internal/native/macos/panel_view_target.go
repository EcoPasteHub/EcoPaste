//go:build darwin

package macos

import (
	"github.com/ebitengine/purego/objc"
)

func registerPanelViewClass() error {
	var err error
	panelViewClass, err = objc.RegisterClass("EcoPastePanelView", class("NSView"), nil, nil, []objc.MethodDef{
		{Cmd: sel("viewDidMoveToWindow"), Fn: panelViewDidMoveToWindow},
		{Cmd: sel("setFrameSize:"), Fn: panelViewSetFrameSize},
		{Cmd: sel("mouseDown:"), Fn: panelViewMouseDown},
		{Cmd: sel("mouseEntered:"), Fn: panelViewMouseEntered},
		{Cmd: sel("mouseExited:"), Fn: panelViewMouseExited},
		{Cmd: sel("acceptsFirstMouse:"), Fn: panelViewAcceptsFirstMouse},
		{Cmd: sel("keyDown:"), Fn: panelViewKeyDown},
		{Cmd: sel("cancelOperation:"), Fn: panelViewCancelOperation},
	})
	return err
}

func panelViewDidMoveToWindow(self objc.ID, _ objc.SEL) {
	sendSuper[objc.ID](self, panelViewClass, sel("viewDidMoveToWindow"))
	if p := panelsByView[self]; p != nil {
		installPanelHoverTracking(p)
		applyPlacement(p)
		attachNavWindow(p)
	}
}

// installPanelHoverTracking lets the panel hide the action bar when the pointer
// leaves it. Mouse movement itself is read from a local event monitor, which
// also sees moves that land on the bar floating over the list.
func installPanelHoverTracking(p *Panel) {
	window := p.view.Send(sel("window"))
	if window == 0 {
		return
	}
	window.Send(sel("setAcceptsMouseMovedEvents:"), true)
	if p.viewTracking != 0 {
		return
	}
	area := objc.ID(class("NSTrackingArea")).Send(sel("alloc")).
		Send(sel("initWithRect:options:owner:userInfo:"), CGRect{}, uint64(1|128|512), p.view, objc.ID(0))
	p.view.Send(sel("addTrackingArea:"), area)
	area.Send(sel("release"))
	p.viewTracking = area
}

func panelViewMouseEntered(self objc.ID, _ objc.SEL, event objc.ID) {
	if p := panelsByView[self]; p != nil {
		p.updateHoverAtWindowPoint(objc.Send[CGPoint](event, sel("locationInWindow")))
	}
}

func panelViewMouseExited(self objc.ID, _ objc.SEL, _ objc.ID) {
	if p := panelsByView[self]; p != nil {
		p.hideHover()
	}
}

func panelViewSetFrameSize(self objc.ID, _ objc.SEL, size CGSize) {
	sendSuper[objc.ID](self, panelViewClass, sel("setFrameSize:"), size)
	if p := panelsByView[self]; p != nil {
		layoutPanel(p, size.Width, size.Height)
	}
}

func panelViewMouseDown(self objc.ID, _ objc.SEL, event objc.ID) {
	p := panelsByView[self]
	if p == nil || objc.Send[int64](event, sel("buttonNumber")) != 0 {
		sendSuper[objc.ID](self, panelViewClass, sel("mouseDown:"), event)
		return
	}
	panelDragWindow(p, self.Send(sel("window")), nil)
}

func panelViewAcceptsFirstMouse(_ objc.ID, _ objc.SEL, _ objc.ID) bool {
	return true
}

func panelViewKeyDown(self objc.ID, _ objc.SEL, event objc.ID) {
	p := panelsByView[self]
	if p != nil && p.previewing && objc.Send[uint16](event, sel("keyCode")) == 53 {
		p.cx.Emit("hidePreview", nil)
		return
	}
	sendSuper[objc.ID](self, panelViewClass, sel("keyDown:"), event)
}

func panelViewCancelOperation(self objc.ID, _ objc.SEL, _ objc.ID) {
	p := panelsByView[self]
	if p != nil && p.previewing {
		p.cx.Emit("hidePreview", nil)
		return
	}
	if p != nil {
		p.cx.Emit("hide", nil)
	}
}
