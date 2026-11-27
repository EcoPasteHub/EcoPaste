//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

// NSPopover content layout contract
//
// An NSPopover owns the frame of its content view. When the popover is shown,
// AppKit places that view at an inner origin (about 13pt) inside the popover
// frame to leave room for the arrow and shadow. Writing the root view's frame
// with origin (0,0) therefore shifts every subview into that chrome margin:
// visible padding collapses and text hugs the popover edge. Static popovers
// avoid this by sizing the view only once; dynamic content (preview, and the
// shortcut sheet historically) breaks when it re-frames the root on update.
//
// Every popover must therefore change its content size through
// setPopoverContentSize, which preserves the origin AppKit assigned, and must
// never move the root content view on show or refresh.
const popoverContentPadding = 14.0

// newPopoverContentView creates the root view of a popover and sizes it to the
// requested content size. Subviews are laid out in this coordinate space using
// popoverContentPadding for the outer breathing room.
func newPopoverContentView(width, height float64) objc.ID {
	view := objc.ID(class("NSView")).Send(sel("new"))
	view.Send(sel("setAutoresizesSubviews:"), false)
	setFrame(view, 0, 0, width, height)
	return view
}

// newPopoverController wraps a content view in the NSViewController an
// NSPopover requires.
func newPopoverController(view objc.ID) objc.ID {
	controller := objc.ID(class("NSViewController")).Send(sel("new"))
	controller.Send(sel("setView:"), view)
	return controller
}

// setPopoverContentSize updates the popover's content size and resizes its
// content view while preserving the origin AppKit assigned inside the popover
// frame. Always resize through this helper instead of calling setFrame on the
// root view, otherwise the content shifts into the popover chrome.
func setPopoverContentSize(popover, view objc.ID, width, height float64) {
	popover.Send(sel("setContentSize:"), CGSize{Width: width, Height: height})
	frame := objc.Send[CGRect](view, sel("frame"))
	setFrame(view, frame.Origin.X, frame.Origin.Y, width, height)
}

// popoverContentBounds returns the shared padded content rect for a popover of
// the given content size. Business views may subdivide it further, but the
// outer padding stays the same everywhere.
func popoverContentBounds(width, height, padding float64) CGRect {
	return CGRect{
		Origin: CGPoint{X: padding, Y: padding},
		Size:   CGSize{Width: width - 2*padding, Height: height - 2*padding},
	}
}

// showPopover is the single entry point for presenting a popover so every
// surface anchors the same way.
func showPopover(popover, ofView objc.ID, rect CGRect, edge uint64) {
	popover.Send(sel("showRelativeToRect:ofView:preferredEdge:"), rect, ofView, edge)
}

// newPopoverContentScrollView builds a borderless, transparent scroll view for
// scrolling popover content. Its frame must always span the full popover width
// (see layoutPopoverContentScroll) so the scroller hugs the right edge; the
// content padding lives inside the document view instead of shrinking the
// scroll view.
func newPopoverContentScrollView() objc.ID {
	scroll := objc.ID(class("NSScrollView")).Send(sel("new"))
	scroll.Send(sel("setHasVerticalScroller:"), true)
	scroll.Send(sel("setAutohidesScrollers:"), true)
	scroll.Send(sel("setDrawsBackground:"), false)
	scroll.Send(sel("setBorderType:"), int64(0))
	return scroll
}

// layoutPopoverContentScroll places a content scroll view across the full
// popover width. Keeping the horizontal origin at 0 and the width at the full
// content width pins the vertical scroller to the popover's right edge; the
// document view's own inset provides the horizontal content padding.
func layoutPopoverContentScroll(scroll objc.ID, y, width, height float64) {
	setFrame(scroll, 0, y, width, height)
}
