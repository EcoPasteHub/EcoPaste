//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

const settingsMaximumContentHeight = 680.0

// resizeWindow follows the active page's measured document height, bounded
// by the usable screen area and a comfortable maximum. It keeps the window's
// top edge in place and lets the page's scroll view handle any clipped content.
func (v *SettingsView) resizeWindow(animated bool) {
	window := v.view.Send(sel("window"))
	if window == 0 {
		return
	}
	frame := objc.Send[CGRect](window, sel("frame"))
	contentView := window.Send(sel("contentView"))
	contentHeight := objc.Send[CGRect](contentView, sel("frame")).Size.Height
	chromeHeight := max(0.0, frame.Size.Height-contentHeight)
	screen := window.Send(sel("screen"))
	if screen == 0 {
		screen = objc.ID(class("NSScreen")).Send(sel("mainScreen"))
	}
	visible := CGRect{Size: CGSize{Width: settingsWidth, Height: settingsMaximumContentHeight + chromeHeight + 2*settingsScreenMargin}}
	if screen != 0 {
		visible = objc.Send[CGRect](screen, sel("visibleFrame"))
	}
	maxContentHeight := min(settingsMaximumContentHeight, visible.Size.Height-chromeHeight-2*settingsScreenMargin)
	maxContentHeight = max(1.0, maxContentHeight)
	targetHeight := settingsWindowContentHeight(v.pageHeights[v.currentTab], maxContentHeight)
	if v.targetWindowHeight == targetHeight {
		return
	}
	v.targetWindowHeight = targetHeight

	contentRect := CGRect{Size: CGSize{Width: settingsWidth, Height: targetHeight}}
	windowFrame := objc.Send[CGRect](window, sel("frameRectForContentRect:"), contentRect)
	frame = settingsFrameAtHeight(frame, windowFrame.Size.Height, visible)

	if !animated || !objc.Send[bool](window, sel("isVisible")) || reduceMotionEnabled() {
		window.Send(sel("setFrame:display:"), frame, true)
		return
	}
	animateSettingsWindowFrame(window, frame)
}

func settingsWindowContentHeight(pageHeight, maxContentHeight float64) float64 {
	maxContentHeight = max(1.0, maxContentHeight)
	minimum := min(settingsMinimumWindowHeight, maxContentHeight)
	return min(max(settingsNavigationHeight+pageHeight, minimum), maxContentHeight)
}

func settingsFrameAtHeight(frame CGRect, height float64, visible CGRect) CGRect {
	top := frame.Origin.Y + frame.Size.Height
	frame.Size.Height = height
	minX := visible.Origin.X + settingsScreenMargin
	maxX := visible.Origin.X + visible.Size.Width - frame.Size.Width - settingsScreenMargin
	if maxX < minX {
		maxX = minX
	}
	frame.Origin.X = min(max(frame.Origin.X, minX), maxX)
	minY := visible.Origin.Y + settingsScreenMargin
	maxY := visible.Origin.Y + visible.Size.Height - frame.Size.Height - settingsScreenMargin
	if maxY < minY {
		maxY = minY
	}
	frame.Origin.Y = min(max(top-frame.Size.Height, minY), maxY)
	return frame
}

func animateSettingsWindowFrame(window objc.ID, frame CGRect) {
	contextClass := class("NSAnimationContext")
	objc.ID(contextClass).Send(sel("beginGrouping"))
	context := objc.ID(contextClass).Send(sel("currentContext"))
	context.Send(sel("setDuration:"), settingsPageTransitionDuration)
	timing := objc.ID(class("CAMediaTimingFunction")).Send(sel("functionWithName:"), nsString("easeInEaseOut"))
	if timing != 0 {
		context.Send(sel("setTimingFunction:"), timing)
	}
	window.Send(sel("animator")).Send(sel("setFrame:display:"), frame, true)
	objc.ID(contextClass).Send(sel("endGrouping"))
}
