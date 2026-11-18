//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

// drawSettingsInputChrome draws the shared input treatment (default, hover
// and focus) used by every in-app input: the shortcut recorder, numeric
// fields and the shared textInput widget. It is the single source of truth
// for input borders, radius and background.
func drawSettingsInputChrome(self objc.ID, hover, focus float64) {
	rect := bounds(self)
	rect.Origin.X += 0.5
	rect.Origin.Y += 0.5
	rect.Size.Width -= 1
	rect.Size.Height -= 1
	radius := rect.Size.Height / 2

	background := objc.ID(classNSColor).Send(sel("whiteColor"))
	if appearanceIsDark(self) {
		background = background.Send(sel("colorWithAlphaComponent:"), 0.075)
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
