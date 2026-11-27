//go:build darwin

package macos

import "testing"

func rowsWithIDs(ids ...string) []HistoryRow {
	rows := make([]HistoryRow, len(ids))
	for i, id := range ids {
		rows[i] = HistoryRow{ID: id}
	}
	return rows
}

func TestSelectionIndexAlwaysKeepsOneRow(t *testing.T) {
	cases := []struct {
		name          string
		rows          []HistoryRow
		currentID     string
		previousIndex int
		intent        selectionIntent
		want          int
	}{
		{"empty list has nothing to select", nil, "", -1, selectionKeep, -1},
		{"first load selects first", rowsWithIDs("a", "b", "c"), "", -1, selectionKeep, 0},
		{"preserves selection after insertion", rowsWithIDs("new", "a", "b", "c"), "c", 1, selectionKeep, 3},
		{"force first ignores the old selection", rowsWithIDs("new", "a", "b", "c"), "c", 3, selectionFirst, 0},
		{"group switch selects first even with old row", rowsWithIDs("a", "b", "c"), "b", 1, selectionFirst, 0},
		{"search selects first even with old row", rowsWithIDs("a", "b", "c"), "c", 2, selectionFirst, 0},
		{"delete middle selects next", rowsWithIDs("a", "b", "d", "e"), "gone", 2, selectionKeep, 2},
		{"delete last selects previous", rowsWithIDs("a", "b", "c", "d"), "gone", 4, selectionKeep, 3},
		{"delete only row clears selection", nil, "gone", 0, selectionKeep, -1},
		{"removed item clamps to last", rowsWithIDs("a", "b", "c"), "gone", 4, selectionKeep, 2},
		{"nothing selected selects first", rowsWithIDs("a", "b"), "", -1, selectionKeep, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := selectionIndex(tc.rows, tc.currentID, tc.previousIndex, tc.intent); got != tc.want {
				t.Errorf("selectionIndex = %d; want %d", got, tc.want)
			}
		})
	}
}

func TestNextRowIndexClampsAndStartsAtFirst(t *testing.T) {
	cases := []struct {
		name           string
		current, delta int
		count, want    int
	}{
		{"empty list", 0, 1, 0, -1},
		{"nothing selected moves down to first", -1, 1, 5, 0},
		{"nothing selected moves up to first", -1, -1, 5, 0},
		{"down one", 2, 1, 5, 3},
		{"up one", 2, -1, 5, 1},
		{"down stops at last", 4, 1, 5, 4},
		{"up stops at first", 0, -1, 5, 0},
		{"page down stays in bounds", 3, 10, 5, 4},
		{"page up stays in bounds", 1, -10, 5, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := nextRowIndex(tc.current, tc.delta, tc.count); got != tc.want {
				t.Errorf("nextRowIndex(%d, %d, %d) = %d; want %d",
					tc.current, tc.delta, tc.count, got, tc.want)
			}
		})
	}
}
