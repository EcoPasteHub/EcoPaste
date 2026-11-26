//go:build darwin

package macos

import (
	"math"
	"testing"

	"github.com/ebitengine/purego/objc"
)

func TestKeyHintKeysAreSingleUppercase(t *testing.T) {
	keys := []string{keyHintPinKey, keyHintSettingsKey, keyHintShortcutKey, keyHintNewGroupKey}
	seen := map[string]bool{}
	for _, key := range keys {
		if len([]rune(key)) != 1 {
			t.Errorf("key hint %q is not a single character", key)
		}
		if got := shortcutCharacter(key); got != key {
			t.Errorf("key hint %q normalizes to %q", key, got)
		}
		if seen[key] {
			t.Errorf("duplicate key hint %q", key)
		}
		seen[key] = true
	}
}

func TestCommandHeldTogglesKeyHints(t *testing.T) {
	onMain(func() {
		panel := NewPanel(nil).(*Panel)
		defer panel.Close()

		if len(panel.keyHints) != 3 {
			t.Fatalf("panel has %d key hints, want pin, settings and shortcuts", len(panel.keyHints))
		}
		hidden := func(ref *keyHintRef) bool {
			return objc.Send[bool](ref.badge, sel("isHidden"))
		}
		for _, ref := range panel.keyHints {
			if !hidden(ref) {
				t.Fatalf("hint visible before Command is held")
			}
			// The badge floats centered over the icon without resizing the button.
			buttonBounds := bounds(ref.button)
			badgeFrame := objc.Send[CGRect](ref.badge, sel("frame"))
			if buttonBounds.Size.Width >= keyHintBadgeSize {
				if want := (buttonBounds.Size.Width - keyHintBadgeSize) / 2; math.Abs(badgeFrame.Origin.X-want) > 0.5 {
					t.Errorf("badge x = %v, want %v (centered)", badgeFrame.Origin.X, want)
				}
			}
		}

		panel.setCommandHeld(true)
		clear := objc.ID(classNSColor).Send(sel("clearColor"))
		for _, ref := range panel.keyHints {
			if hidden(ref) {
				t.Errorf("hint still hidden while Command is held")
			}
			tint := objc.Send[objc.ID](ref.button, sel("contentTintColor"))
			if tint == 0 || !objc.Send[bool](tint, sel("isEqual:"), clear) {
				t.Errorf("icon not hidden behind the badge while Command is held")
			}
		}

		panel.setCommandHeld(false)
		for _, ref := range panel.keyHints {
			if !hidden(ref) {
				t.Errorf("hint visible after Command is released")
			}
		}
	})
}

func TestKeyHintDigitForRow(t *testing.T) {
	cases := map[int]string{
		-1: "", 0: "1", 8: "9", 9: "0", 10: "", 42: "",
	}
	for index, want := range cases {
		if got := keyHintDigitForRow(index); got != want {
			t.Errorf("keyHintDigitForRow(%d) = %q, want %q", index, got, want)
		}
	}
}

func TestVisibleDigitIndexFollowsFirstVisibleRow(t *testing.T) {
	if index, ok := visibleDigitIndex(key1, 5, 20); !ok || index != 5 {
		t.Errorf("Command-1 at first visible 5 = (%d, %v), want (5, true)", index, ok)
	}
	if index, ok := visibleDigitIndex(key0, 5, 20); !ok || index != 14 {
		t.Errorf("Command-0 at first visible 5 = (%d, %v), want (14, true)", index, ok)
	}
	if _, ok := visibleDigitIndex(key0, 15, 20); ok {
		t.Errorf("Command-0 past the last row should have no target")
	}
	if index, ok := visibleDigitIndex(key3, 0, 20); !ok || index != 2 {
		t.Errorf("Command-3 at the top = (%d, %v), want (2, true)", index, ok)
	}
}

func TestRowIconHintShowsDigitOverIcon(t *testing.T) {
	onMain(func() {
		panel := NewPanel(nil).(*Panel)
		defer panel.Close()

		cell := newHistoryRow()
		cell.Send(sel("retain"))
		defer cell.Send(sel("release"))
		badge := rowHintBadge(cell)
		icon := viewWithTag(cell, rowIconTag)
		hidden := func() bool { return objc.Send[bool](badge, sel("isHidden")) }
		alpha := func() float64 { return objc.Send[float64](icon, sel("alphaValue")) }

		setRowIconHintDigit(cell, keyHintDigitForRow(2))
		applyRowIconHint(cell, true)
		if hidden() {
			t.Errorf("digit badge hidden while Command is held")
		}
		if alpha() != 0 {
			t.Errorf("app icon alpha = %v while its digit is shown, want 0", alpha())
		}

		applyRowIconHint(cell, false)
		if !hidden() {
			t.Errorf("digit badge visible after Command is released")
		}
		if alpha() != 1 {
			t.Errorf("app icon alpha = %v after release, want 1", alpha())
		}

		// Rows past the first ten have no Command digit and keep their icon.
		setRowIconHintDigit(cell, keyHintDigitForRow(10))
		applyRowIconHint(cell, true)
		if !hidden() {
			t.Errorf("digit badge visible for a row without a shortcut")
		}
		if alpha() != 1 {
			t.Errorf("app icon hidden for a row without a shortcut")
		}
	})
}
