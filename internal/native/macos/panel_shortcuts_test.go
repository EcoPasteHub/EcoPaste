//go:build darwin

package macos

import "testing"

func TestShortcutHelpItemsAreComplete(t *testing.T) {
	items := shortcutHelpItems()
	if len(items) < 10 {
		t.Fatalf("shortcut help lists %d items, want the full panel reference", len(items))
	}
	seen := map[string]bool{}
	for _, item := range items {
		if item.Label == "" || item.Keys == "" {
			t.Errorf("empty shortcut row: %+v", item)
		}
		if seen[item.Keys] {
			t.Errorf("duplicate shortcut keys %q", item.Keys)
		}
		seen[item.Keys] = true
	}
	if height := shortcutHelpHeight(); height < shortcutHelpHeader {
		t.Errorf("shortcut help height %v is too small", height)
	}
}
