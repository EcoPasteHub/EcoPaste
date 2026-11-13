//go:build darwin

package macos

import "testing"

func TestHistoryRowSelectionRectInsetsRow(t *testing.T) {
	got := historyRowSelectionRect(CGRect{Size: CGSize{Width: 420, Height: rowHeight}})
	want := CGRect{
		Origin: CGPoint{X: historyRowInsetX, Y: historyRowInsetY},
		Size:   CGSize{Width: 404, Height: 52},
	}
	if got != want {
		t.Fatalf("selection rect = %+v, want %+v", got, want)
	}
}
