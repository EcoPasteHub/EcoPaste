//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

const iconButtonRadius = 6.0

var (
	iconButtonClass  objc.Class
	iconButtonHovers = map[objc.ID]*transition{}
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
				ensureHoverTracking(self)
			}},
			{Cmd: sel("mouseEntered:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if hover := iconButtonHovers[self]; hover != nil {
					hover.setTarget(1)
				}
			}},
			{Cmd: sel("mouseExited:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if hover := iconButtonHovers[self]; hover != nil {
					hover.setTarget(0)
				}
			}},
			{Cmd: sel("viewDidChangeEffectiveAppearance"), Fn: func(self objc.ID, _ objc.SEL) {
				sendSuper[objc.ID](self, iconButtonClass, sel("viewDidChangeEffectiveAppearance"))
				self.Send(sel("setNeedsDisplay:"), true)
			}},
		})
	return err
}

func newIconButton() objc.ID {
	button := objc.ID(iconButtonClass).Send(sel("alloc")).Send(sel("initWithFrame:"), CGRect{})
	iconButtonHovers[button] = newViewTransition(button)
	return button
}

// discardIconButton releases the hover transition when the button is torn
// down, so the shared driver keeps no reference to a dead view.
func discardIconButton(button objc.ID) {
	if hover := iconButtonHovers[button]; hover != nil {
		hover.stop()
		delete(iconButtonHovers, button)
	}
}

func drawIconButton(self objc.ID, _ objc.SEL, _ CGRect) {
	pressed := objc.Send[bool](self, sel("isHighlighted"))
	hover := 0.0
	if state := iconButtonHovers[self]; state != nil {
		hover = state.value
	}
	if hover > 0.001 || pressed {
		fraction := hover
		if pressed {
			fraction = 1
		}
		fillRoundedRect(iconButtonBackgroundRect(self), iconButtonRadius,
			fadeColor(iconButtonFill(pressed), fraction))
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

func iconButtonFill(pressed bool) objc.ID {
	color := objc.ID(classNSColor).Send(sel("quaternaryLabelColor"))
	if pressed {
		color = objc.ID(classNSColor).Send(sel("tertiaryLabelColor"))
	}
	return color
}
