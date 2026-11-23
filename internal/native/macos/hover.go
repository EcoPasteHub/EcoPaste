//go:build darwin

package macos

import (
	"strings"

	"github.com/ebitengine/purego/objc"
)

// Hover tracking keeps the row under the pointer highlighted and, for note
// entries, swaps the card back to the original content while hovered. There is
// no floating action bar: actions are provided by the right-click menu and
// keyboard shortcuts.

// updateHoverAtWindowPoint updates the hovered row for a point in window
// coordinates. A mouse-moved event calls it; hover needs no polling.
func (p *Panel) updateHoverAtWindowPoint(windowPoint CGPoint) {
	row, ok := p.hoverRowAtWindowPoint(windowPoint)
	if !ok {
		p.hideHover()
		return
	}
	p.updateHover(row)
}

// refreshHoverFromMouse re-evaluates the hovered row after the list scrolls,
// when the cursor has not moved and so no mouse-moved event fires.
func (p *Panel) refreshHoverFromMouse() {
	window := p.view.Send(sel("window"))
	if window == 0 {
		return
	}
	location := objc.Send[CGPoint](objc.ID(class("NSEvent")), sel("mouseLocation"))
	windowPoint := objc.Send[CGPoint](window, sel("convertPointFromScreen:"), location)
	row, ok := p.hoverRowAtWindowPoint(windowPoint)
	if !ok {
		p.hideHover()
		return
	}
	p.updateHover(row)
}

// hoverRowAtWindowPoint resolves the row under a window point, or false when
// the point is outside the list or the panel is not showing.
func (p *Panel) hoverRowAtWindowPoint(windowPoint CGPoint) (int, bool) {
	window := p.view.Send(sel("window"))
	if window == 0 || !objc.Send[bool](window, sel("isVisible")) || p.previewing {
		return 0, false
	}
	point := objc.Send[CGPoint](p.view, sel("convertPoint:fromView:"), windowPoint, objc.ID(0))
	frame := objc.Send[CGRect](p.scroll, sel("frame"))
	if !pointInRect(point, frame) {
		return 0, false
	}
	local := objc.Send[CGPoint](p.table, sel("convertPoint:fromView:"), windowPoint, objc.ID(0))
	return int(objc.Send[int64](p.table, sel("rowAtPoint:"), local)), true
}

func (p *Panel) updateHover(row int) {
	if row < 0 || row >= len(p.rows) {
		p.hideHover()
		return
	}
	// Moving within the same row needs no further work.
	if row == p.hoverRow {
		return
	}
	p.hoverRow = row
	p.setHoveredRow(row)
	p.setOriginalRow(row)
}

func (p *Panel) hideHover() {
	p.hoverRow = -1
	p.setHoveredRow(-1)
	p.setOriginalRow(-1)
}

func (p *Panel) setOriginalRow(row int) {
	if row < 0 || row >= len(p.rows) || strings.TrimSpace(p.rows[row].Note) == "" || !p.display.ShowOriginal {
		row = -1
	}
	if p.originalRow == row {
		return
	}
	previous := p.originalRow
	p.originalRow = row
	p.refreshHistoryRow(previous)
	p.refreshHistoryRow(row)
}

func (p *Panel) refreshHistoryRow(row int) {
	if row < 0 || row >= len(p.rows) {
		return
	}
	rows := objc.ID(class("NSIndexSet")).Send(sel("indexSetWithIndex:"), int64(row))
	columns := objc.ID(class("NSIndexSet")).Send(sel("indexSetWithIndex:"), int64(0))
	p.table.Send(sel("reloadDataForRowIndexes:columnIndexes:"), rows, columns)
	p.applyHoveredRow(row, row == p.hoveredRow)
}

func (p *Panel) setHoveredRow(row int) {
	if row < 0 || row >= len(p.rows) {
		row = -1
	}
	if p.hoveredRow == row {
		return
	}
	previous := p.hoveredRow
	p.hoveredRow = row
	p.applyHoveredRow(previous, false)
	p.applyHoveredRow(row, true)
}

func (p *Panel) applyHoveredRow(row int, hovered bool) {
	if row < 0 || row >= len(p.rows) {
		return
	}
	cell := p.table.Send(sel("viewAtColumn:row:makeIfNecessary:"), int64(0), int64(row), false)
	if cell != 0 {
		setHistoryRowHovered(cell, hovered)
	}
}

func pointInRect(point CGPoint, rect CGRect) bool {
	return point.X >= rect.Origin.X && point.X <= rect.Origin.X+rect.Size.Width &&
		point.Y >= rect.Origin.Y && point.Y <= rect.Origin.Y+rect.Size.Height
}
