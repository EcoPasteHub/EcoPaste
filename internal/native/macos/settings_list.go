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
