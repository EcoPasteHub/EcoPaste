//go:build darwin

package macos

import "testing"

func TestPanelSearchWidthLeavesRoomForToolbar(t *testing.T) {
	const width = 420.0
	toolbar := headerToolbarX(width)
	got := panelSearchWidth(width)
	want := toolbar.clear - panelHeaderSearchMargin - panelToolbarGap
	if got != want {
		t.Fatalf("search width = %v, want %v", got, want)
	}
	if got < 200 {
		t.Fatalf("search width = %v, too narrow for the panel", got)
	}
	searchRight := panelHeaderSearchMargin + got
	if searchRight > toolbar.clear {
		t.Fatalf("search field ends at %v, toolbar starts at %v", searchRight, toolbar.clear)
	}
}

func TestPanelHeaderKeepsEqualMargins(t *testing.T) {
	const width, height = 420.0, 560.0
	header := layoutHeader(width, height)

	// The glyphs sit inside their frames, so the visible gap is measured
	// from the frames the text and the symbols are drawn in.
	left := header.searchX + panelSearchInset
	right := width - (header.pinX + panelIconSize - panelIconInset)
	top := height - (header.iconY + panelIconSize - panelIconInset)
	bottom := (header.iconY + panelIconInset) - (header.listTop - historyRowInsetY - rowLeadingInset)

	for name, got := range map[string]float64{
		"left":   left,
		"right":  right,
		"top":    top,
		"bottom": bottom,
	} {
		if got != panelHeaderMargin {
			t.Errorf("%s margin = %.1f, want %.1f", name, got, panelHeaderMargin)
		}
	}
}

func TestPanelHeaderCentersSearchOnIconAxis(t *testing.T) {
	const width, height = 420.0, 560.0
	header := layoutHeader(width, height)

	// The search text is drawn just under the field's top edge, the
	// symbols are centered in their button: both must end up on the
	// same vertical axis.
	iconAxis := header.iconY + panelIconSize/2
	textAxis := header.searchY + panelSearchText - panelSearchInset - panelSearchInkHeight/2
	if textAxis != iconAxis {
		t.Fatalf("search text axis = %.1f, icon axis = %.1f", textAxis, iconAxis)
	}
}

func TestPanelNavCapsulesFitPanelWidth(t *testing.T) {
	const width = 420.0
	used := panelNavPadding + panelNavPaddingEnd + panelNavActionSize
	for _, label := range panelNavLabels() {
		used += panelNavCapsuleWidth(label) + panelNavCapsuleGap
	}
	if used > width {
		t.Fatalf("navigation capsules use %.1f points, panel is %.1f wide", used, width)
	}
}

func TestPanelNavTightensForLongerLocale(t *testing.T) {
	const width = 420.0
	english := []string{"All", "Text", "Images", "Files", "Favorites"}
	if panelNavCapsulesFit(english, panelNavCapsulePad, panelNavCapsuleGap, width) {
		t.Fatal("English labels should not fit at the default capsule padding")
	}
	if !panelNavCapsulesFit(english, 6, 2, width) {
		t.Fatal("English labels should fit once the capsules tighten")
	}
}
