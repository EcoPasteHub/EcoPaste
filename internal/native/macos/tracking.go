//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

const hoverTrackingOptions uint64 = 1 | 128 | 512

// ensureHoverTracking reuses the view's visible-rect tracking area. AppKit
// updates its geometry, so repeated layout passes must not add listeners.
func ensureHoverTracking(view objc.ID) {
	areas := view.Send(sel("trackingAreas"))
	count := objc.Send[uint](areas, sel("count"))
	for index := uint(0); index < count; index++ {
		area := areas.Send(sel("objectAtIndex:"), index)
		if objc.Send[objc.ID](area, sel("owner")) == view &&
			objc.Send[uint64](area, sel("options")) == hoverTrackingOptions {
			return
		}
	}
	area := objc.ID(class("NSTrackingArea")).Send(sel("alloc")).
		Send(sel("initWithRect:options:owner:userInfo:"), CGRect{}, hoverTrackingOptions, view, objc.ID(0))
	view.Send(sel("addTrackingArea:"), area)
	area.Send(sel("release"))
}
