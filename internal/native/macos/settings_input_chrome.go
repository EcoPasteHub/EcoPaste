//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

// drawSettingsInputChrome is the single source of truth for the input
// border, background and default/hover/focus treatment. A zero radius keeps
// the single-line pill shape; multi-line areas pass a smaller corner radius.
func drawSettingsInputChrome(self objc.ID, hover, focus, radius float64) {
	rect := bounds(self)
	rect.Origin.X += 0.5
	rect.Origin.Y += 0.5
	rect.Size.Width -= 1
	rect.Size.Height -= 1
	if radius <= 0 {
		radius = rect.Size.Height / 2
	}

	background := objc.ID(classNSColor).Send(sel("whiteColor"))
	if appearanceIsDark(self) {
		// Solid dark, not a translucent white wash: the input may sit on a
		// light popover material and its text must stay readable there.
		background = objc.ID(classNSColor).Send(sel("colorWithWhite:alpha:"), 0.12, 1.0)
	}
	fillRoundedRect(rect, radius, background)

	border := objc.ID(classNSColor).Send(sel("separatorColor")).
		Send(sel("blendedColorWithFraction:ofColor:"), hover,
			objc.ID(classNSColor).Send(sel("controlAccentColor"))).
		Send(sel("blendedColorWithFraction:ofColor:"), focus,
			objc.ID(classNSColor).Send(sel("controlAccentColor")))
	path := objc.ID(class("NSBezierPath")).Send(sel("bezierPathWithRoundedRect:xRadius:yRadius:"),
		rect, radius, radius)
	path.Send(sel("setLineWidth:"), 1.0+0.6*focus)
	border.Send(sel("setStroke"))
	path.Send(sel("stroke"))
}
