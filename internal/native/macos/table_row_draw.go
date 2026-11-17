//go:build darwin

package macos

import (
	"unsafe"

	"github.com/ebitengine/purego/objc"
)

// drawHistoryRowCard paints the single translucent layer each record
// gets. The window material stays visible through it, so the list reads
// as cards on glass instead of another panel inside the window.
func drawHistoryRowCard(self objc.ID, _ objc.SEL, _ CGRect) {
	selected := historyRowSelected(self)
	hovered := historyRowHoveredVal(self)
	rect := visibleRowWidth(self)
	if selected {
		fillHistoryRowCard(rect, historyRowSelectionColor(), historyRowRadius)
		return
	}
	fillHistoryRowCard(rect, historyRowCardColor(self, hovered), historyRowRadius)
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

func historyRowHoveredVal(cell objc.ID) bool {
	value := objc.ID(objcGetAssociatedObject(uintptr(cell), uintptr(unsafe.Pointer(&associatedRowHoveredKey))))
	return value != 0 && objc.Send[bool](value, sel("boolValue"))
}

// historyRowCardColor is a barely-there fill in both appearances: a
// light lift over the light glass, a faint white veil over the dark one.
func historyRowCardColor(rowView objc.ID, hovered bool) objc.ID {
	white := objc.ID(classNSColor).Send(sel("whiteColor"))
	if appearanceIsDark(rowView) {
		if hovered {
			return white.Send(sel("colorWithAlphaComponent:"), 0.12)
		}
		return white.Send(sel("colorWithAlphaComponent:"), 0.06)
	}
	if hovered {
		return white.Send(sel("colorWithAlphaComponent:"), 0.44)
	}
	return white.Send(sel("colorWithAlphaComponent:"), 0.24)
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

func setHistoryRowHovered(rowView objc.ID, hovered bool) {
	value := objc.ID(class("NSNumber")).Send(sel("numberWithBool:"), hovered)
	objcSetAssociatedObject(uintptr(rowView), uintptr(unsafe.Pointer(&associatedRowHoveredKey)),
		uintptr(value), associationRetainNonatomic)
	rowView.Send(sel("setNeedsDisplay:"), true)
}
