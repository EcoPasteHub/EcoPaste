//go:build darwin

package macos

import (
	"testing"

	"github.com/ebitengine/purego/objc"
)

// TestNSRangeStructReturn guards the ABI the visible-row refresh relies on:
// rowsInRect: returns an NSRange, so objc.Send[NSRange] must decode a
// two-word struct return correctly on this architecture. rangeOfString: is a
// Foundation method with the same return shape and needs no AppKit window.
func TestNSRangeStructReturn(t *testing.T) {
	text := nsString("abcd")
	got := objc.Send[NSRange](text, sel("rangeOfString:"), nsString("bc"))
	if got.Location != 1 || got.Length != 2 {
		t.Fatalf("rangeOfString: = {Location:%d Length:%d}; want {1 2}", got.Location, got.Length)
	}
}

func TestHistoryTableSupportsVisibleRowRange(t *testing.T) {
	tableClass := objc.ID(objc.GetClass("NSTableView"))
	if tableClass == 0 {
		t.Fatal("NSTableView class not found")
	}
	if !objc.Send[bool](tableClass, sel("instancesRespondToSelector:"), sel("rowsInRect:")) {
		t.Fatal("NSTableView does not respond to rowsInRect:")
	}
}
