//go:build darwin

package macos

import (
	"strings"
	"testing"
)

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
		name    string
		heights []float64
		want    float64
	}{
		{name: "empty", heights: nil, want: 0},
		// Every row owns a slot of its height plus the intercell spacing,
		// so the last row's trailing slot belongs to the document too.
		{name: "one row", heights: []float64{62}, want: 62 + 6},
		{name: "three rows", heights: []float64{62, 52, 84}, want: 62 + 6 + 52 + 6 + 84 + 6},
	}
	for _, tc := range cases {
		if got := historyTableContentHeight(tc.heights, 6); got != tc.want {
			t.Errorf("%s: content height = %.0f, want %.0f", tc.name, got, tc.want)
		}
	}
}

// TestHistoryTableContentHeightCoversLastCard guards the bottom edge of
// the list. NSTableView centers the cell in a slot that carries the
// intercell spacing, so the last card ends inside the trailing slot: a
// document that stops at the last row's own bottom clips its lower edge
// when the list is scrolled to the end.
func TestHistoryTableContentHeightCoversLastCard(t *testing.T) {
	const spacing = 6.0
	heights := []float64{62, 52, 84}

	content := historyTableContentHeight(heights, spacing)
	previous := historyTableContentHeight(heights[:len(heights)-1], spacing)
	last := heights[len(heights)-1]
	cardBottom := previous + spacing/2 + last - historyRowInsetY

	if content <= cardBottom {
		t.Fatalf("content height %.0f does not cover the last card's bottom %.0f", content, cardBottom)
	}
	if gap := content - cardBottom; gap < historyRowInsetY {
		t.Fatalf("gap below the last card = %.1f, want at least %.1f", gap, historyRowInsetY)
	}
}

func TestHistoryRowHeightAdaptsToTextLines(t *testing.T) {
	display := PanelDisplay{TextMaxLines: 3, ImageMaxHeight: 64, FileMaxCount: 3}
	single := historyRowHeight(HistoryRow{Kind: "text", Title: "one short line"}, display, 420)
	two := historyRowHeight(HistoryRow{Kind: "text", Title: "headline", Preview: "body line"}, display, 420)
	capped := historyRowHeight(HistoryRow{Kind: "text", Title: strings.Repeat("word ", 400)}, display, 420)

	if single != 52 {
		t.Errorf("single line height = %.0f, want 52", single)
	}
	if two != 69 {
		t.Errorf("two line height = %.0f, want 69", two)
	}
	if capped != 84 {
		t.Errorf("capped height = %.0f, want 84 (3 lines)", capped)
	}
	if !(single < two && two < capped) {
		t.Errorf("heights should grow with content: %.0f %.0f %.0f", single, two, capped)
	}
}

func TestHistoryRowHeightFollowsImageAndFiles(t *testing.T) {
	display := PanelDisplay{TextMaxLines: 3, ImageMaxHeight: 64, FileMaxCount: 3}
	image := historyRowHeight(HistoryRow{Kind: "image", ImagePath: "/tmp/a.png", ImageWidth: 480, ImageHeight: 310}, display, 420)
	if image != 100 {
		t.Errorf("image height = %.0f, want 100", image)
	}
	oneFile := historyRowHeight(HistoryRow{Kind: "files", Files: []HistoryFile{{Name: "a.txt"}}}, display, 420)
	twoFiles := historyRowHeight(HistoryRow{Kind: "files", Files: []HistoryFile{{Name: "a.txt"}, {Name: "b.md"}}}, display, 420)
	if oneFile != 56 {
		t.Errorf("one file height = %.0f, want 56", oneFile)
	}
	if twoFiles != 78 {
		t.Errorf("two file height = %.0f, want 78", twoFiles)
	}
}

func TestFitImageSizeKeepsAspectRatioAndMaxHeight(t *testing.T) {
	got := fitImageSize(CGSize{Width: 480, Height: 310}, 338, 64)
	if got.Height != 64 {
		t.Errorf("height = %.1f, want 64", got.Height)
	}
	if got.Width <= 0 || got.Width > 338 {
		t.Errorf("width = %.1f, out of range", got.Width)
	}
	tiny := fitImageSize(CGSize{Width: 16, Height: 16}, 338, 64)
	if tiny.Width != 16 || tiny.Height != 16 {
		t.Errorf("tiny image should not upscale: %+v", tiny)
	}
}
