//go:build darwin

package macos

import (
	"testing"

	"github.com/ebitengine/purego/objc"
)

// withFocusProbe replaces the editor-state probe for one test and restores it.
func withFocusProbe(t *testing.T, probe func(objc.ID) bool) {
	t.Helper()
	previous := inputFocusProbe
	inputFocusProbe = probe
	t.Cleanup(func() { inputFocusProbe = previous })
}

// TestInputChromeTracksRealFocusNotNotifications verifies the border follows
// the real editor state even when an editing notification disagrees. This is
// the core invariant: a stale or premature notification must not light the
// border unless the field is actually the key window's editing responder.
func TestInputChromeTracksRealFocusNotNotifications(t *testing.T) {
	view := objc.ID(class("NSView")).Send(sel("new"))
	field := objc.ID(classNSTextField).Send(sel("alloc")).Send(sel("initWithFrame:"), CGRect{})
	chrome := newInputChrome(view, field)
	t.Cleanup(chrome.close)

	var focused bool
	withFocusProbe(t, func(f objc.ID) bool { return f == field && focused })

	center := objc.ID(class("NSNotificationCenter")).Send(sel("defaultCenter"))
	// Begin-editing arrives, but the window is not key: focus must stay off.
	focused = false
	center.Send(sel("postNotificationName:object:"),
		nsString("NSControlTextDidBeginEditingNotification"), field)
	if chrome.focus.target != 0 {
		t.Fatalf("unfocused field lit border: target = %v, want 0", chrome.focus.target)
	}

	// The window becomes key and the field is really editing: focus on.
	focused = true
	center.Send(sel("postNotificationName:object:"),
		nsString("NSControlTextDidBeginEditingNotification"), field)
	if chrome.focus.target != 1 {
		t.Fatalf("focused field did not light border: target = %v, want 1", chrome.focus.target)
	}

	// Losing the editing responder turns the border off again.
	focused = false
	center.Send(sel("postNotificationName:object:"),
		nsString("NSControlTextDidEndEditingNotification"), field)
	if chrome.focus.target != 0 {
		t.Fatalf("after end: target = %v, want 0", chrome.focus.target)
	}
}

// TestInputChromeSyncFocusIgnoresStaleExplicitState proves syncFocus derives the
// target from the field instead of trusting its argument, so a remembered focus
// from a previous popover session cannot keep the border lit.
func TestInputChromeSyncFocusIgnoresStaleExplicitState(t *testing.T) {
	view := objc.ID(class("NSView")).Send(sel("new"))
	field := objc.ID(classNSTextField).Send(sel("alloc")).Send(sel("initWithFrame:"), CGRect{})
	chrome := newInputChrome(view, field)
	t.Cleanup(chrome.close)

	withFocusProbe(t, func(objc.ID) bool { return false })
	chrome.focus.value = 1
	chrome.focus.target = 1

	chrome.syncFocus(true)

	if chrome.focus.target != 0 {
		t.Fatalf("stale focus kept border lit: target = %v, want 0", chrome.focus.target)
	}
}

// TestInputChromeManualFocusStillWorks covers controls with no bound field (the
// shortcut recorder): their explicit focus argument drives the border.
func TestInputChromeManualFocusStillWorks(t *testing.T) {
	view := objc.ID(class("NSView")).Send(sel("new"))
	chrome := newInputChrome(view, 0)
	t.Cleanup(chrome.close)

	chrome.syncFocus(true)
	if chrome.focus.target != 1 {
		t.Fatalf("manual focus: target = %v, want 1", chrome.focus.target)
	}
	chrome.syncFocus(false)
	if chrome.focus.target != 0 {
		t.Fatalf("manual blur: target = %v, want 0", chrome.focus.target)
	}
}

// TestInputChromeResetFocusClearsStaleValue makes sure a reused input starts
// visually idle before it is focused again.
func TestInputChromeResetFocusClearsStaleValue(t *testing.T) {
	view := objc.ID(class("NSView")).Send(sel("new"))
	field := objc.ID(classNSTextField).Send(sel("alloc")).Send(sel("initWithFrame:"), CGRect{})
	chrome := newInputChrome(view, field)
	t.Cleanup(chrome.close)

	chrome.focus.value = 1
	chrome.focus.target = 1
	chrome.resetFocus()

	if chrome.focus.value != 0 || chrome.focus.target != 0 {
		t.Fatalf("resetFocus left value=%v target=%v, want 0/0", chrome.focus.value, chrome.focus.target)
	}
}

// TestInputChromeIgnoresUnknownField makes sure an unrelated field's editing
// notification does not touch another input's border.
func TestInputChromeIgnoresUnknownField(t *testing.T) {
	view := objc.ID(class("NSView")).Send(sel("new"))
	field := objc.ID(classNSTextField).Send(sel("alloc")).Send(sel("initWithFrame:"), CGRect{})
	other := objc.ID(classNSTextField).Send(sel("alloc")).Send(sel("initWithFrame:"), CGRect{})
	chrome := newInputChrome(view, field)
	t.Cleanup(chrome.close)

	withFocusProbe(t, func(objc.ID) bool { return true })
	center := objc.ID(class("NSNotificationCenter")).Send(sel("defaultCenter"))
	center.Send(sel("postNotificationName:object:"),
		nsString("NSControlTextDidBeginEditingNotification"), other)
	if chrome.focus.target != 0 {
		t.Fatalf("unknown field changed focus target to %v", chrome.focus.target)
	}
}
