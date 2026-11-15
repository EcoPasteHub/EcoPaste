//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

const iconButtonRadius = 6.0

var (
	iconButtonClass    objc.Class
	hoveredIconButtons = map[objc.ID]bool{}
)

// registerIconButtonClass installs the shared hover treatment used by the
// panel's icon-only actions.
func registerIconButtonClass() error {
	var err error
	iconButtonClass, err = objc.RegisterClass("EcoPasteIconButton", class("NSButton"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("drawRect:"), Fn: drawIconButton},
			{Cmd: sel("acceptsFirstMouse:"), Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) bool { return true }},
			{Cmd: sel("updateTrackingAreas"), Fn: func(self objc.ID, _ objc.SEL) {
				sendSuper[objc.ID](self, iconButtonClass, sel("updateTrackingAreas"))
				area := objc.ID(class("NSTrackingArea")).Send(sel("alloc")).
					Send(sel("initWithRect:options:owner:userInfo:"), CGRect{}, uint64(1|128|512), self, objc.ID(0))
				self.Send(sel("addTrackingArea:"), area)
				area.Send(sel("release"))
			}},
			{Cmd: sel("mouseEntered:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				hoveredIconButtons[self] = true
				self.Send(sel("setNeedsDisplay:"), true)
			}},
			{Cmd: sel("mouseExited:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				delete(hoveredIconButtons, self)
				self.Send(sel("setNeedsDisplay:"), true)
			}},
			{Cmd: sel("viewDidChangeEffectiveAppearance"), Fn: func(self objc.ID, _ objc.SEL) {
				sendSuper[objc.ID](self, iconButtonClass, sel("viewDidChangeEffectiveAppearance"))
				self.Send(sel("setNeedsDisplay:"), true)
			}},
		})
	return err
}

func newIconButton() objc.ID {
	return objc.ID(iconButtonClass).Send(sel("alloc")).Send(sel("initWithFrame:"), CGRect{})
}

func drawIconButton(self objc.ID, _ objc.SEL, _ CGRect) {
	pressed := objc.Send[bool](self, sel("isHighlighted"))
	if hoveredIconButtons[self] || pressed {
		fillRoundedRect(iconButtonBackgroundRect(self), iconButtonRadius, iconButtonFill(self, pressed))
	}
	sendSuper[objc.ID](self, iconButtonClass, sel("drawRect:"))
}

func iconButtonBackgroundRect(view objc.ID) CGRect {
	rect := bounds(view)
	inset := 2.0
	if rect.Size.Width < 22 || rect.Size.Height < 22 {
		inset = 1
	}
	rect.Origin.X += inset
	rect.Origin.Y += inset
	rect.Size.Width -= 2 * inset
	rect.Size.Height -= 2 * inset
	return rect
}

func iconButtonFill(view objc.ID, pressed bool) objc.ID {
	color := objc.ID(classNSColor).Send(sel("quaternaryLabelColor"))
	if pressed {
		color = objc.ID(classNSColor).Send(sel("tertiaryLabelColor"))
	}
	return color
}
