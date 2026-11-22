//go:build darwin

package macos

import (
	"testing"

	"github.com/ebitengine/purego/objc"
)

// TestSetPopoverContentSizePreservesOrigin locks in the popover contract:
// resizing must keep the origin AppKit assigns inside the popover frame.
// Writing origin (0,0) is what made preview content hug the popover edge.
func TestSetPopoverContentSizePreservesOrigin(t *testing.T) {
	onMain(func() {
		pool := objc.ID(class("NSAutoreleasePool")).Send(sel("new"))
		defer pool.Send(sel("drain"))

		view := newPopoverContentView(200, 100)
		// Simulate the frame AppKit assigns when the popover is shown.
		setFrame(view, 13, 13, 200, 100)
		controller := newPopoverController(view)
		popover := objc.ID(class("NSPopover")).Send(sel("new"))
		popover.Send(sel("setContentViewController:"), controller)

		setPopoverContentSize(popover, view, 320, 240)
		assertPreviewFrame(t, view, 13, 13, 320, 240)

		setPopoverContentSize(popover, view, 400, 300)
		assertPreviewFrame(t, view, 13, 13, 400, 300)

		popover.Send(sel("release"))
		controller.Send(sel("release"))
		view.Send(sel("release"))
	})
}

func assertPreviewFrame(t *testing.T, view objc.ID, x, y, width, height float64) {
	t.Helper()
	frame := objc.Send[CGRect](view, sel("frame"))
	if frame.Origin.X != x || frame.Origin.Y != y ||
		frame.Size.Width != width || frame.Size.Height != height {
		t.Fatalf("content view frame = (%.1f,%.1f,%.1f,%.1f), want (%.1f,%.1f,%.1f,%.1f)",
			frame.Origin.X, frame.Origin.Y, frame.Size.Width, frame.Size.Height, x, y, width, height)
	}
}

func TestNewPopoverContentViewSizesItself(t *testing.T) {
	onMain(func() {
		pool := objc.ID(class("NSAutoreleasePool")).Send(sel("new"))
		defer pool.Send(sel("drain"))
		view := newPopoverContentView(260, 120)
		assertPreviewFrame(t, view, 0, 0, 260, 120)
		if objc.Send[bool](view, sel("autoresizesSubviews")) {
			t.Fatal("popover content view must not autoresize its subviews")
		}
		view.Send(sel("release"))
	})
}

func TestPopoverContentBoundsAppliesPadding(t *testing.T) {
	rect := popoverContentBounds(300, 200, popoverContentPadding)
	if rect.Origin.X != popoverContentPadding || rect.Origin.Y != popoverContentPadding {
		t.Fatalf("origin = (%.1f,%.1f)", rect.Origin.X, rect.Origin.Y)
	}
	if rect.Size.Width != 300-2*popoverContentPadding || rect.Size.Height != 200-2*popoverContentPadding {
		t.Fatalf("size = (%.1f,%.1f)", rect.Size.Width, rect.Size.Height)
	}
}
