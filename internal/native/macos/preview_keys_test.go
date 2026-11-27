//go:build darwin

package macos

import "testing"

// TestPreviewKeyActionKeepsNavigation covers the keyboard contract while the
// preview popover is open: navigation actions must stay navigation (so the
// popover follows the selection) rather than becoming a hide request.
func TestPreviewKeyActionKeepsNavigation(t *testing.T) {
	row := HistoryRow{ID: "a", Kind: "text"}
	nav := functionModifier | numericPadModifier
	cases := []struct {
		name   string
		key    uint16
		flags  uint64
		want   string
		handle bool
	}{
		{"space closes", keySpace, 0, "hide", true},
		{"escape closes", keyEscape, 0, "hide", true},
		{"return activates", keyReturn, 0, "activate", true},
		{"enter activates", keyEnter, 0, "activate", true},
		{"up navigates", keyUpArrow, nav, "previousRow", true},
		{"down navigates", keyDownArrow, nav, "nextRow", true},
		{"page up navigates", keyPageUp, nav, "pageUp", true},
		{"page down navigates", keyPageDown, nav, "pageDown", true},
		{"home navigates", keyHome, nav, "firstRow", true},
		{"end navigates", keyEnd, nav, "lastRow", true},
		{"command C copies", keyC, commandModifier, "copy", true},
		{"delete removes", keyDelete, 0, "delete", true},
		{"letters are left alone", keyF, 0, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			action, handled := previewKeyAction(tc.key, tc.flags, row, true)
			if action != tc.want || handled != tc.handle {
				t.Errorf("previewKeyAction() = %q, %v; want %q, %v", action, handled, tc.want, tc.handle)
			}
		})
	}
}

// TestPreviewNavigationIsNotHide guards the specific regression: switching
// rows with the arrow keys must never dismiss the popover.
func TestPreviewNavigationIsNotHide(t *testing.T) {
	nav := functionModifier | numericPadModifier
	for _, key := range []uint16{keyUpArrow, keyDownArrow, keyPageUp, keyPageDown, keyHome, keyEnd} {
		action, handled := previewKeyAction(key, nav, HistoryRow{Kind: "text"}, true)
		if !handled {
			t.Fatalf("key %d should be handled while previewing", key)
		}
		if action == "hide" {
			t.Fatalf("key %d closed the preview; it must navigate", key)
		}
	}
}
