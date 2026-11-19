//go:build darwin

package macos

import "testing"

func TestReusedRowHeightsOnlyExtendsUnchangedPrefix(t *testing.T) {
	previous := rowsWithIDs("a", "b", "c")
	heights := []float64{10, 20, 30}
	cases := []struct {
		name     string
		rows     []HistoryRow
		heights  []float64
		oldWidth float64
		width    float64
		want     int
	}{
		{"append reuses every previous height", rowsWithIDs("a", "b", "c", "d"), heights, 420, 420, 3},
		{"identical list reuses everything", rowsWithIDs("a", "b", "c"), heights, 420, 420, 3},
		{"changed prefix row is not reusable", rowsWithIDs("a", "x", "c", "d"), heights, 420, 420, 0},
		{"shorter list is not an append", rowsWithIDs("a", "b"), heights, 420, 420, 0},
		{"width change invalidates heights", rowsWithIDs("a", "b", "c", "d"), heights, 420, 500, 0},
		{"height count mismatch invalidates", rowsWithIDs("a", "b", "c", "d"), []float64{10, 20}, 420, 420, 0},
		{"empty previous list reuses nothing", nil, nil, 0, 420, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := reusedRowHeights(previous, tc.heights, tc.rows, tc.oldWidth, tc.width); got != tc.want {
				t.Errorf("reusedRowHeights = %d; want %d", got, tc.want)
			}
		})
	}
}

func TestRowsExtendRequiresUnchangedPrefix(t *testing.T) {
	cases := []struct {
		name     string
		previous []HistoryRow
		rows     []HistoryRow
		want     bool
	}{
		{"extends the list", rowsWithIDs("a", "b"), rowsWithIDs("a", "b", "c"), true},
		{"identical list is still a prefix", rowsWithIDs("a", "b"), rowsWithIDs("a", "b"), true},
		{"changed prefix row", rowsWithIDs("a", "b"), rowsWithIDs("a", "x", "c"), false},
		{"shorter list is not an extend", rowsWithIDs("a", "b", "c"), rowsWithIDs("a", "b"), false},
		{"empty previous is a prefix of anything", nil, rowsWithIDs("a"), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := rowsExtend(tc.previous, tc.rows); got != tc.want {
				t.Errorf("rowsExtend = %v; want %v", got, tc.want)
			}
		})
	}
}

func TestCanAppendRows(t *testing.T) {
	width := 420.0
	panel := func(rows []HistoryRow) *Panel {
		return &Panel{
			rows:            rows,
			rowHeights:      make([]float64, len(rows)),
			rowHeightsWidth: width,
		}
	}
	cases := []struct {
		name string
		p    *Panel
		rows []HistoryRow
		want bool
	}{
		{"appending a page", panel(rowsWithIDs("a", "b")), rowsWithIDs("a", "b", "c", "d"), true},
		{"empty table never appends", panel(nil), rowsWithIDs("a", "b"), false},
		{"same length is a replace", panel(rowsWithIDs("a", "b")), rowsWithIDs("a", "b"), false},
		{"changed prefix is a replace", panel(rowsWithIDs("a", "b")), rowsWithIDs("a", "x", "c"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.p.canAppendRows(tc.rows); got != tc.want {
				t.Errorf("canAppendRows = %v; want %v", got, tc.want)
			}
		})
	}
}
