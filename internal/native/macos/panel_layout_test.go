//go:build darwin

package macos

import "testing"

func TestPanelSearchWidthLeavesRoomForToolbar(t *testing.T) {
	const width = 420.0
	got := panelSearchWidth(width)
	want := width - 2*panelSidePadding - 4*panelIconSize - 3*panelIconGap - 10
	if got != want {
		t.Fatalf("search width = %v, want %v", got, want)
	}
	if got < 200 {
		t.Fatalf("search width = %v, too narrow for the panel", got)
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
