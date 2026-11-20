//go:build darwin

package macos

import (
	"unsafe"

	"github.com/ebitengine/purego/objc"
)

// rowHoverStates keeps one transition per cell view so the hover lift fades
// instead of snapping. Cells are reused by the table, so entries are cheap.
var rowHoverStates = map[objc.ID]*transition{}

const associationRetainNonatomic = 1

// drawHistoryRowCard paints the single translucent layer each record
// gets. The window material stays visible through it, so the list reads
// as cards on glass instead of another panel inside the window.
func drawHistoryRowCard(self objc.ID, _ objc.SEL, _ CGRect) {
	selected := historyRowSelected(self)
	hover := historyRowHoveredVal(self)
	rect := visibleRowWidth(self)
	if selected {
		fillHistoryRowCard(rect, historyRowSelectionColor(), historyRowRadius)
		return
	}
	fillHistoryRowCard(rect, historyRowCardColor(self, hover), historyRowRadius)
}

func setHistoryRowSelected(cell objc.ID, selected bool) {
	value := objc.ID(class("NSNumber")).Send(sel("numberWithBool:"), selected)
	objcSetAssociatedObject(uintptr(cell), uintptr(unsafe.Pointer(&associatedRowSelectedKey)),
		uintptr(value), associationRetainNonatomic)
	cell.Send(sel("setNeedsDisplay:"), true)
}

func historyRowSelected(cell objc.ID) bool {
	value := objc.ID(objcGetAssociatedObject(uintptr(cell), uintptr(unsafe.Pointer(&associatedRowSelectedKey))))
	return value != 0 && objc.Send[bool](value, sel("boolValue"))
}

func historyRowHoveredVal(cell objc.ID) float64 {
	if state := rowHoverStates[cell]; state != nil {
		return state.value
	}
	return 0
}

// historyRowCardColor is a barely-there fill in both appearances: a
// light lift over the light glass, a faint white veil over the dark one.
func historyRowCardColor(rowView objc.ID, hover float64) objc.ID {
	white := objc.ID(classNSColor).Send(sel("whiteColor"))
	idle, active := 0.24, 0.44
	if appearanceIsDark(rowView) {
		idle, active = 0.06, 0.12
	}
	alpha := idle + (active-idle)*hover
	return white.Send(sel("colorWithAlphaComponent:"), alpha)
}

// historyRowSelectionColor uses the system's emphasized selection fill
// so the selected card stays distinct in both appearances.
func historyRowSelectionColor() objc.ID {
	return objc.ID(classNSColor).Send(sel("selectedContentBackgroundColor"))
}

func fillHistoryRowCard(rect CGRect, color objc.ID, radius float64) {
	inset := historyRowSelectionRect(rect)
	path := objc.ID(class("NSBezierPath")).Send(sel("bezierPathWithRoundedRect:xRadius:yRadius:"),
		inset, radius, radius)
	color.Send(sel("setFill"))
	path.Send(sel("fill"))
}

func historyRowSelectionRect(rect CGRect) CGRect {
	return CGRect{
		Origin: CGPoint{X: rect.Origin.X + historyRowInsetX, Y: rect.Origin.Y + historyRowInsetY},
		Size: CGSize{
			Width:  rect.Size.Width - 2*historyRowInsetX,
			Height: rect.Size.Height - 2*historyRowInsetY,
		},
	}
}

// visibleRowWidth trims the cell's frame to the part the table actually
// shows. The cell can be wider than the clip view, and drawing its full
// width would leave no right margin beside the scroll edge.
func visibleRowWidth(view objc.ID) CGRect {
	rect := bounds(view)
	visible := objc.Send[CGRect](view, sel("visibleRect"))
	if visible.Size.Width <= 0 {
		return rect
	}
	left := rect.Origin.X
	right := min(rect.Origin.X+rect.Size.Width, visible.Origin.X+visible.Size.Width)
	if right <= left {
		return rect
	}
	rect.Origin.X = left
	rect.Size.Width = right - left
	return rect
}

func setHistoryRowHovered(view objc.ID, hovered bool) {
	state := rowHoverStates[view]
	if state == nil {
		if !hovered {
			return
		}
		state = newViewTransition(view)
		rowHoverStates[view] = state
	}
	target := 0.0
	if hovered {
		target = 1
	}
	state.setTarget(target)
}

// clearRowHoverStates stops and forgets every row transition, called when the
// owning panel closes.
func clearRowHoverStates() {
	for view, state := range rowHoverStates {
		state.stop()
		delete(rowHoverStates, view)
	}
}
