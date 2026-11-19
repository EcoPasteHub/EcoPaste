//go:build darwin

package macos

import (
	"slices"

	"github.com/ebitengine/purego/objc"
)

// stringSlicesEqual reports whether two string slices hold the same values.
func stringSlicesEqual(a, b []string) bool { return slices.Equal(a, b) }

// stringIndex returns the position of value in list, or -1.
func stringIndex(list []string, value string) int {
	return slices.Index(list, value)
}

// selectSettingsRowNoScroll selects one row without moving the scroll offset.
// Apply-time refresh uses it so updating a table never shifts the user's view.
func selectSettingsRowNoScroll(table objc.ID, row int) {
	if table == 0 || row < 0 {
		return
	}
	indexes := objc.ID(class("NSIndexSet")).Send(sel("indexSetWithIndex:"), int64(row))
	table.Send(sel("selectRowIndexes:byExtendingSelection:"), indexes, false)
}

// reloadSettingsTableRows redraws only the given rows. It leaves the scroll
// offset and selection untouched, so toggling one row's state does not reset
// the list's position.
func reloadSettingsTableRows(table objc.ID, rows []int) {
	if table == 0 || len(rows) == 0 {
		return
	}
	indexes := objc.ID(class("NSMutableIndexSet")).Send(sel("alloc")).Send(sel("init"))
	for _, row := range rows {
		indexes.Send(sel("addIndex:"), int64(row))
	}
	columns := objc.ID(class("NSIndexSet")).Send(sel("indexSetWithIndex:"), int64(0))
	table.Send(sel("reloadDataForRowIndexes:columnIndexes:"), indexes, columns)
	indexes.Send(sel("release"))
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
