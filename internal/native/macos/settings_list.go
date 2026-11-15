//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

// selectedRowAfterResize keeps the table's selected row valid once its row
// count changed, preferring the same row and falling back to the edge.
func selectedRowAfterResize(table objc.ID, rows int) int {
	selected := int(objc.Send[int64](table, sel("selectedRow")))
	if selected >= rows {
		selected = rows - 1
	}
	if selected < 0 && rows > 0 {
		selected = 0
	}
	return selected
}

// moveSelectedRow swaps the selected row with a neighbor and reports the
// row's new index.
func moveSelectedRow[T any](table objc.ID, rows []T, delta int) (int, bool) {
	row := int(objc.Send[int64](table, sel("selectedRow")))
	next := row + delta
	if row < 0 || row >= len(rows) || next < 0 || next >= len(rows) {
		return 0, false
	}
	rows[row], rows[next] = rows[next], rows[row]
	return next, true
}

// selectSettingsRow selects and reveals one row in a settings order table.
func selectSettingsRow(table objc.ID, row int) {
	indexes := objc.ID(class("NSIndexSet")).Send(sel("indexSetWithIndex:"), int64(row))
	table.Send(sel("selectRowIndexes:byExtendingSelection:"), indexes, false)
	table.Send(sel("scrollRowToVisible:"), int64(row))
}

// updateSettingsMoveButtons enables the reorder controls for the current
// selection.
func updateSettingsMoveButtons(table, up, down objc.ID, total int) {
	row := -1
	if table != 0 {
		row = int(objc.Send[int64](table, sel("selectedRow")))
	}
	upEnabled, downEnabled := orderMoveEnabled(row, total)
	if up != 0 {
		up.Send(sel("setEnabled:"), upEnabled)
	}
	if down != 0 {
		down.Send(sel("setEnabled:"), downEnabled)
	}
}
