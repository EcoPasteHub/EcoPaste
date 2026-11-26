//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

// keyHintDigitForRow maps a row's offset from the first visible row to the digit
// shown over its app icon while Command is held. Command-1 through Command-0
// paste the first ten visible rows, so only those offsets carry a hint.
func keyHintDigitForRow(offset int) string {
	if offset < 0 || offset > 9 {
		return ""
	}
	if offset == 9 {
		return "0"
	}
	return string(rune('1' + offset))
}

// firstVisibleRow returns the first row intersecting the viewport. Quick-paste
// digits are numbered from here, so they follow the scroll position.
func firstVisibleRow(table objc.ID) int {
	if table == 0 {
		return 0
	}
	visible := objc.Send[CGRect](table, sel("visibleRect"))
	if visible.Size.Height <= 0 {
		return 0
	}
	rows := objc.Send[NSRange](table, sel("rowsInRect:"), visible)
	if rows.Length == 0 {
		return 0
	}
	return int(rows.Location)
}

// setRowIconHintDigit records the digit a cell's icon badge should show. Row
// cells are reused, so the digit lives on the badge view itself.
func setRowIconHintDigit(cell objc.ID, digit string) {
	badge := rowHintBadge(cell)
	if badge == 0 {
		return
	}
	badge.Send(sel("setIdentifier:"), nsString(shortcutCharacter(digit)))
	badge.Send(sel("setNeedsDisplay:"), true)
}

// applyRowIconHint shows or hides one cell's digit badge. While shown the app
// icon is hidden so the badge takes its place, matching the legacy KeyHint
// overlay.
func applyRowIconHint(cell objc.ID, active bool) {
	badge := rowHintBadge(cell)
	icon := viewWithTag(cell, rowIconTag)
	if badge == 0 || icon == 0 {
		return
	}
	show := active && goString(badge.Send(sel("identifier"))) != ""
	badge.Send(sel("setHidden:"), !show)
	alpha := 1.0
	if show {
		alpha = 0
	}
	icon.Send(sel("setAlphaValue:"), alpha)
}

// refreshRowHints renumbers the visible cards from the current top row and
// applies the current Command state. It runs when Command toggles and while the
// list scrolls, so "在哪一行就显示到哪一行".
func (p *Panel) refreshRowHints() {
	if p.table == 0 {
		return
	}
	first := firstVisibleRow(p.table)
	start, end := visibleRowRange(p.table)
	if end > len(p.rows) {
		end = len(p.rows)
	}
	for index := start; index < end; index++ {
		cell := p.table.Send(sel("viewAtColumn:row:makeIfNecessary:"), int64(0), int64(index), false)
		if cell == 0 {
			continue
		}
		setRowIconHintDigit(cell, keyHintDigitForRow(index-first))
		applyRowIconHint(cell, p.commandHeld)
	}
}
