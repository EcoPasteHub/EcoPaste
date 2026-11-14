//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

const (
	panelFadeDuration = 0.14
)

func animateIn(view objc.ID, duration float64) {
	if view == 0 {
		return
	}
	view.Send(sel("setAlphaValue:"), 1.0)
	if reduceMotionEnabled() {
		return
	}
	view.Send(sel("setWantsLayer:"), true)
	layer := view.Send(sel("layer"))
	if layer == 0 {
		return
	}
	animation := objc.ID(class("CABasicAnimation")).Send(sel("animationWithKeyPath:"), nsString("opacity"))
	if animation == 0 {
		return
	}
	animation.Send(sel("setFromValue:"),
		objc.ID(class("NSNumber")).Send(sel("numberWithFloat:"), float32(0)))
	animation.Send(sel("setToValue:"),
		objc.ID(class("NSNumber")).Send(sel("numberWithFloat:"), float32(1)))
	animation.Send(sel("setDuration:"), duration)
	timing := objc.ID(class("CAMediaTimingFunction")).Send(sel("functionWithName:"), nsString("easeOut"))
	if timing != 0 {
		animation.Send(sel("setTimingFunction:"), timing)
	}
	layer.Send(sel("addAnimation:forKey:"), animation, nsString("fadeIn"))
}

func reduceMotionEnabled() bool {
	workspace := objc.ID(class("NSWorkspace")).Send(sel("sharedWorkspace"))
	return objc.Send[bool](workspace, sel("accessibilityDisplayShouldReduceMotion"))
}
