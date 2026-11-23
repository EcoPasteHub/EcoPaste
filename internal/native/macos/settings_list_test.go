//go:build darwin

package macos

import "testing"

func TestOrderMoveEnabled(t *testing.T) {
	cases := []struct {
		selected int
		total    int
		up, down bool
	}{
		{selected: -1, total: 3},
		{selected: 0, total: 0},
		{selected: 0, total: 3, down: true},
		{selected: 1, total: 3, up: true, down: true},
		{selected: 2, total: 3, up: true},
		{selected: 3, total: 3},
	}
	for _, tc := range cases {
		up, down := orderMoveEnabled(tc.selected, tc.total)
		if up != tc.up || down != tc.down {
			t.Errorf("orderMoveEnabled(%d, %d) = %v, %v; want %v, %v",
				tc.selected, tc.total, up, down, tc.up, tc.down)
		}
	}
}
