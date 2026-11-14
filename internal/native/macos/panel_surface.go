//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

// panelBackdropClass draws the milky lift over the window's frosted
// material: light over a light appearance, a faint white veil in dark,
// so the panel keeps its glass instead of turning into a black slab.
var panelBackdropClass objc.Class

func registerPanelBackdrop() error {
	var err error
	panelBackdropClass, err = objc.RegisterClass("EcoPastePanelBackdrop", class("NSView"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("drawRect:"), Fn: func(self objc.ID, _ objc.SEL, _ CGRect) {
				fillRoundedRect(bounds(self), 0, panelBackdropTint(self))
			}},
			{Cmd: sel("viewDidChangeEffectiveAppearance"), Fn: func(self objc.ID, _ objc.SEL) {
				sendSuper[objc.ID](self, panelBackdropClass, sel("viewDidChangeEffectiveAppearance"))
				self.Send(sel("setNeedsDisplay:"), true)
			}},
			{Cmd: sel("hitTest:"), Fn: func(_ objc.ID, _ objc.SEL, _ CGPoint) objc.ID { return 0 }},
		})
	return err
}

func panelBackdropTint(view objc.ID) objc.ID {
	color := objc.ID(classNSColor).Send(sel("whiteColor"))
	if appearanceIsDark(view) {
		return color.Send(sel("colorWithAlphaComponent:"), 0.04)
	}
	return color.Send(sel("colorWithAlphaComponent:"), 0.18)
}

// stylePanelWindow rounds the panel window's own surface so the history
// window reads as its own frosted card, not a plain rectangle.
func stylePanelWindow(p *Panel) {
	window := p.view.Send(sel("window"))
	if window == 0 {
		return
	}
	content := window.Send(sel("contentView"))
	if content != 0 {
		content.Send(sel("setWantsLayer:"), true)
		layer := content.Send(sel("layer"))
		layer.Send(sel("setCornerRadius:"), 16)
		layer.Send(sel("setMasksToBounds:"), true)
	}
	window.Send(sel("setHasShadow:"), true)
	window.Send(sel("invalidateShadow"))
}

// attachNavWindow creates the capsule window once the panel view joins
// its window, and keeps it as the panel's child from then on.
func attachNavWindow(p *Panel) {
	window := p.view.Send(sel("window"))
	if window == 0 {
		return
	}
	stylePanelWindow(p)
	if p.nav == nil {
		p.nav = newPanelNav(p, window)
	}
	p.nav.position()
	if objc.Send[bool](window, sel("isVisible")) && !p.onboardingActive {
		p.nav.show()
	}
}

// syncNavVisibility keeps the capsule window in step with the panel.
func syncNavVisibility(p *Panel) {
	if p.nav == nil {
		return
	}
	window := p.view.Send(sel("window"))
	if window != 0 && objc.Send[bool](window, sel("isVisible")) && !p.onboardingActive {
		p.nav.show()
		return
	}
	p.nav.hide()
}
