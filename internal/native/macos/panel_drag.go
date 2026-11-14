//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

const (
	panelDragThreshold = 3.0

	eventLeftMouseDown    = 1
	eventLeftMouseUp      = 2
	eventLeftMouseDragged = 6
)

// panelDragWindow moves window from a press in one of the panel's blank
// areas: the window follows the pointer once the press passes the drag
// threshold, and stays a click below it. click, when not nil, handles
// that click. It returns the monitor that follows the drag, 0 when no
// drag could start.
//
// The panel cannot use AppKit's background dragging: its full-size
// content view turns every blank area into a title bar, and NSGo keeps
// panels unmovable so a press never moves the window on its own. Moving
// the frame by the pointer's delta keeps both windows together, since
// AppKit carries the capsule bar along as the panel's child.
func panelDragWindow(p *Panel, window objc.ID, click func()) objc.ID {
	if window == 0 || p.dragMonitor != 0 {
		return 0
	}
	window.Send(sel("makeKeyWindow"))
	start := mouseLocation()
	origin := objc.Send[CGRect](window, sel("frame")).Origin
	moved := false
	block := objc.NewBlock(func(_ objc.Block, event objc.ID) objc.ID {
		switch objc.Send[int64](event, sel("type")) {
		case eventLeftMouseDown:
			// A press without its mouse-up leaves a stale monitor behind;
			// the next press ends it instead of dragging the panel.
			endPanelDrag(p)
		case eventLeftMouseDragged:
			if !objc.Send[bool](window, sel("isVisible")) {
				endPanelDrag(p)
				return event
			}
			point := mouseLocation()
			dx, dy := point.X-start.X, point.Y-start.Y
			if !moved && dx*dx+dy*dy < panelDragThreshold*panelDragThreshold {
				return event
			}
			moved = true
			p.dragging = true
			window.Send(sel("setFrameOrigin:"), CGPoint{X: origin.X + dx, Y: origin.Y + dy})
			return 0
		case eventLeftMouseUp:
			endPanelDrag(p)
			if !moved && click != nil {
				click()
			}
			return 0
		}
		return event
	})
	p.dragMonitor = objc.ID(class("NSEvent")).Send(
		sel("addLocalMonitorForEventsMatchingMask:handler:"),
		uint64(1<<eventLeftMouseDown|1<<eventLeftMouseDragged|1<<eventLeftMouseUp), objc.ID(block))
	block.Release()
	return p.dragMonitor
}

// endPanelDrag stops following a press, whether it moved the window or
// not.
func endPanelDrag(p *Panel) {
	if p.dragMonitor != 0 {
		objc.ID(class("NSEvent")).Send(sel("removeMonitor:"), p.dragMonitor)
		p.dragMonitor = 0
	}
	p.dragging = false
}

// beginNavDrag moves the panel from the capsule bar. The bar is a child
// window, so the drag moves the parent instead of sliding the bar out of
// alignment.
func (n *panelNav) beginNavDrag() {
	panelDragWindow(n.panel, n.panel.view.Send(sel("window")), nil)
}
