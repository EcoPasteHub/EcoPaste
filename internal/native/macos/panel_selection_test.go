//go:build darwin

package macos

import "testing"

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
