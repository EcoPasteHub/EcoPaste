//go:build darwin

package macos

import "testing"

func TestPanelFilterWidthUsesEmptyGroupSpace(t *testing.T) {
	const width = 420.0
	withoutGroups := panelFilterWidth(width, false)
	withGroups := panelFilterWidth(width, true)
	if withoutGroups <= withGroups {
		t.Fatalf("filter width without groups = %v, with groups = %v", withoutGroups, withGroups)
	}
	if want := width - 2*panelSidePadding - panelActionWidth - panelToolbarGap; withoutGroups != want {
		t.Fatalf("filter width without groups = %v, want %v", withoutGroups, want)
	}
	if want := withoutGroups - panelGroupWidth - panelToolbarGap; withGroups != want {
		t.Fatalf("filter width with groups = %v, want %v", withGroups, want)
	}
}
