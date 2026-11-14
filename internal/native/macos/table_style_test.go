//go:build darwin

package macos

import "testing"

func TestHistoryRowSelectionRectInsetsRow(t *testing.T) {
	got := historyRowSelectionRect(CGRect{Size: CGSize{Width: 420, Height: rowHeight}})
	want := CGRect{
		Origin: CGPoint{X: historyRowInsetX, Y: historyRowInsetY},
		Size:   CGSize{Width: 400, Height: 58},
	}
	if got != want {
		t.Fatalf("selection rect = %+v, want %+v", got, want)
	}
}

func TestHistoryTableContentHeightFollowsRows(t *testing.T) {
	cases := []struct {
		rows int
		want float64
	}{
		{rows: 0, want: 0},
		{rows: 1, want: 62},
		{rows: 5, want: 334},
	}
	for _, tc := range cases {
		if got := historyTableContentHeight(tc.rows, 62, 6); got != tc.want {
			t.Errorf("content height for %d rows = %.0f, want %.0f", tc.rows, got, tc.want)
		}
	}
}

func TestHistoryRowHeightTracksDensity(t *testing.T) {
	cases := []struct {
		name   string
		height float64
		want   float64
	}{
		{name: "single text line", height: historyTextRowHeight(1), want: 62},
		{name: "three text lines", height: historyTextRowHeight(3), want: 89},
		{name: "five text lines", height: historyTextRowHeight(5), want: 121},
		{name: "small image", height: historyImageRowHeight(PanelDisplay{ImageMaxHeight: 20}), want: 62},
		{name: "default image", height: historyImageRowHeight(PanelDisplay{ImageMaxHeight: 64}), want: 88},
		{name: "large image", height: historyImageRowHeight(PanelDisplay{ImageMaxHeight: 100}), want: 124},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.height != tc.want {
				t.Fatalf("row height = %.0f, want %.0f", tc.height, tc.want)
			}
		})
	}
}
