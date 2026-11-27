//go:build darwin

package macos

import "testing"

// TestContextMenuAccelerators pins the shortcut hints each context-menu action
// advertises. Every action that the panel handles in table_keys.go must show a
// hint here, and actions without a shortcut must stay empty so the native menu
// keeps its right side blank.
func TestContextMenuAccelerators(t *testing.T) {
	cases := []struct {
		action    string
		key       string
		modifiers uint64
	}{
		{"menuPaste:", keyEquivalentReturn, 0},
		{"menuCopy:", "c", commandModifier},
		{"menuCopyPlain:", "c", commandModifier | alternateModifier},
		{"menuOpen:", "o", commandModifier},
		{"menuReveal:", "o", commandModifier},
		{"menuNote:", "m", commandModifier},
		{"menuFavorite:", "d", commandModifier},
		{"menuPin:", "t", commandModifier},
		{"menuDelete:", keyEquivalentBackspace, 0},
		{"menuSaveImage:", "", 0},
		{"menuMoveGroup:", "", 0},
	}
	for _, c := range cases {
		key, modifiers := contextMenuAccelerator(c.action)
		if key != c.key || modifiers != c.modifiers {
			t.Errorf("contextMenuAccelerator(%q) = (%q, %d), want (%q, %d)",
				c.action, key, modifiers, c.key, c.modifiers)
		}
	}
}
