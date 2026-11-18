//go:build darwin

package macos

import (
	"testing"

	"github.com/ebitengine/purego/objc"
)

// TestInputChromeSyncsFocusFromControlNotifications verifies the shared
// input chrome turns its border focus state on/off from the NSControl
// text-editing notifications, keyed by the inner field. This is what keeps
// every input's outer border in step with its text field.
func TestInputChromeSyncsFocusFromControlNotifications(t *testing.T) {
	view := objc.ID(class("NSView")).Send(sel("new"))
	field := objc.ID(classNSTextField).Send(sel("alloc")).Send(sel("initWithFrame:"), CGRect{})
	chrome := newInputChrome(view, field)
	t.Cleanup(chrome.close)

	center := objc.ID(class("NSNotificationCenter")).Send(sel("defaultCenter"))
	center.Send(sel("postNotificationName:object:"),
		nsString("NSControlTextDidBeginEditingNotification"), field)
	if chrome.focus.target != 1 {
		t.Fatalf("after begin: focus target = %v, want 1", chrome.focus.target)
	}

	center.Send(sel("postNotificationName:object:"),
		nsString("NSControlTextDidEndEditingNotification"), field)
	if chrome.focus.target != 0 {
		t.Fatalf("after end: focus target = %v, want 0", chrome.focus.target)
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

	center := objc.ID(class("NSNotificationCenter")).Send(sel("defaultCenter"))
	center.Send(sel("postNotificationName:object:"),
		nsString("NSControlTextDidBeginEditingNotification"), other)
	if chrome.focus.target != 0 {
		t.Fatalf("unknown field changed focus target to %v", chrome.focus.target)
	}
}
