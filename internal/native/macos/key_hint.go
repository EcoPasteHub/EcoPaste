//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

// Command-hold shortcut hints mirror the legacy KeyHint component: while the
// user holds Command, toolbar icon buttons and the first ten history cards show
// their Command key on a small dark badge. Each badge is an absolutely placed
// subview, so it floats over the icon without taking layout space; releasing
// Command hides it again.
//
// The toolbar letters match the panel's own Command shortcuts and the settings
// menu item (Command-,); history cards show the digits of Command-1..Command-0.
const (
	keyHintPinKey      = "P"
	keyHintSettingsKey = ","
	keyHintShortcutKey = "K"
	keyHintNewGroupKey = "N"

	keyHintBadgeSize   = 16.0
	keyHintBadgeRadius = 4.0
	keyHintFontSize    = 11.0
	keyHintFontWeight  = 0.4

	// keyHintRowBadgeSize is the quick-paste cap over a card's app icon. The
	// icon view is 30pt but its glyph is smaller, so a 20pt cap reads as sitting
	// on the icon rather than swallowing the whole column.
	keyHintRowBadgeSize = 20.0

	// eventMaskFlagsChanged is NSEventMaskFlagsChanged, the event AppKit sends
	// when a modifier key is pressed or released.
	eventMaskFlagsChanged = 1 << 12

	// keyHintAutoresizesCentered keeps a fixed-size badge centered when its
	// button is laid out after creation.
	keyHintAutoresizesCentered = 1 | 4 | 8 | 32
)

var (
	keyHintClass        objc.Class
	keyHintTexts        = map[objc.ID]string{}
	keyHintToolbarStyle = keyHintStyle{radius: keyHintBadgeRadius, fontSize: keyHintFontSize, fontWeight: keyHintFontWeight}
	keyHintRowStyle     = keyHintStyle{radius: 5, fontSize: 12, fontWeight: 0.4}
)

// keyHintStyle tunes the badge per placement: a compact toolbar badge versus a
// slightly larger cap that sits on a card's app icon.
type keyHintStyle struct {
	radius     float64
	fontSize   float64
	fontWeight float64
}

// keyHintRef ties a badge to the button whose icon it replaces while shown.
// tint remembers the button's content tint so hiding the badge can restore it.
type keyHintRef struct {
	button objc.ID
	badge  objc.ID
	tint   objc.ID
	shown  bool
}

func registerKeyHintClass() error {
	var err error
	keyHintClass, err = objc.RegisterClass("EcoPasteKeyHintBadge", class("NSView"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("drawRect:"), Fn: drawKeyHintBadge},
			// The badge only paints; clicks must fall through to the button.
			{Cmd: sel("hitTest:"), Fn: func(_ objc.ID, _ objc.SEL, _ CGPoint) objc.ID { return 0 }},
			{Cmd: sel("viewDidChangeEffectiveAppearance"), Fn: func(self objc.ID, _ objc.SEL) {
				sendSuper[objc.ID](self, keyHintClass, sel("viewDidChangeEffectiveAppearance"))
				self.Send(sel("setNeedsDisplay:"), true)
			}},
		})
	return err
}

func newKeyHintBadge(text string) objc.ID {
	badge := objc.ID(keyHintClass).Send(sel("alloc")).Send(sel("initWithFrame:"),
		CGRect{Size: CGSize{Width: keyHintBadgeSize, Height: keyHintBadgeSize}})
	badge.Send(sel("setHidden:"), true)
	if text != "" {
		keyHintTexts[badge] = shortcutCharacter(text)
	}
	return badge
}

func drawKeyHintBadge(self objc.ID, _ objc.SEL, _ CGRect) {
	text := keyHintTextFor(self)
	if text == "" {
		return
	}
	style := keyHintToolbarStyle
	if keyHintTexts[self] == "" {
		style = keyHintRowStyle
	}
	rect := bounds(self)
	fillRoundedRect(rect, style.radius, keyHintBadgeFill(self))

	font := objc.ID(classNSFont).Send(sel("monospacedSystemFontOfSize:weight:"), style.fontSize, style.fontWeight)
	attributes := objc.ID(class("NSMutableDictionary")).Send(sel("dictionary"))
	attributes.Send(sel("setObject:forKey:"), font, nsString("NSFont"))
	attributes.Send(sel("setObject:forKey:"), objc.ID(classNSColor).Send(sel("whiteColor")), nsString("NSColor"))

	measured := objc.Send[CGSize](nsString(text), sel("sizeWithAttributes:"), attributes)
	textRect := CGRect{
		Origin: CGPoint{X: (rect.Size.Width - measured.Width) / 2, Y: (rect.Size.Height - measured.Height) / 2},
		Size:   measured,
	}
	nsString(text).Send(sel("drawInRect:withAttributes:"), textRect, attributes)
}

// keyHintTextFor reads a badge's letter. Toolbar badges keep it in the global
// map; row badges are reused with the table cells, so they store it in the
// view identifier and need no bookkeeping.
func keyHintTextFor(view objc.ID) string {
	if text := keyHintTexts[view]; text != "" {
		return text
	}
	return goString(view.Send(sel("identifier")))
}

func keyHintBadgeFill(view objc.ID) objc.ID {
	if appearanceIsDark(view) {
		return objc.ID(classNSColor).Send(sel("whiteColor")).Send(sel("colorWithAlphaComponent:"), 0.22)
	}
	return objc.ID(classNSColor).Send(sel("blackColor")).Send(sel("colorWithAlphaComponent:"), 0.82)
}

func (r *keyHintRef) show() {
	if r.shown {
		return
	}
	r.tint = objc.Send[objc.ID](r.button, sel("contentTintColor"))
	r.button.Send(sel("setContentTintColor:"), objc.ID(classNSColor).Send(sel("clearColor")))
	r.badge.Send(sel("setHidden:"), false)
	r.shown = true
}

func (r *keyHintRef) hide() {
	if !r.shown {
		return
	}
	r.badge.Send(sel("setHidden:"), true)
	current := objc.Send[objc.ID](r.button, sel("contentTintColor"))
	if current == 0 || objc.Send[bool](current, sel("isEqual:"), objc.ID(classNSColor).Send(sel("clearColor"))) {
		r.button.Send(sel("setContentTintColor:"), r.tint)
	}
	r.tint = 0
	r.shown = false
}

// addKeyHint attaches a shortcut badge to an icon button. The badge sits at the
// button's center and resizes with it, so it never occupies layout space.
func (p *Panel) addKeyHint(button objc.ID, text string) {
	if button == 0 {
		return
	}
	button.Send(sel("setAutoresizesSubviews:"), true)
	badge := newKeyHintBadge(text)
	badge.Send(sel("setAutoresizingMask:"), uint64(keyHintAutoresizesCentered))
	buttonBounds := bounds(button)
	setFrame(badge,
		(buttonBounds.Size.Width-keyHintBadgeSize)/2,
		(buttonBounds.Size.Height-keyHintBadgeSize)/2,
		keyHintBadgeSize, keyHintBadgeSize)
	button.Send(sel("addSubview:"), badge)
	badge.Send(sel("release"))

	ref := &keyHintRef{button: button, badge: badge}
	p.keyHints = append(p.keyHints, ref)
	if p.commandHeld {
		ref.show()
	}
}

// setCommandHeld shows or hides every hint as the Command key changes.
func (p *Panel) setCommandHeld(held bool) {
	if p.commandHeld == held {
		return
	}
	p.commandHeld = held
	for _, ref := range p.keyHints {
		if held {
			ref.show()
		} else {
			ref.hide()
		}
	}
	p.refreshRowHints()
}

// removeKeyHint drops the badge attached to button, used when a lazily built
// button (the nav plus) is torn down.
func (p *Panel) removeKeyHint(button objc.ID) {
	kept := p.keyHints[:0]
	for _, ref := range p.keyHints {
		if ref.button == button {
			ref.hide()
			delete(keyHintTexts, ref.badge)
			continue
		}
		kept = append(kept, ref)
	}
	p.keyHints = kept
}

// clearKeyHints hides and forgets every badge before the panel is torn down.
func (p *Panel) clearKeyHints() {
	p.setCommandHeld(false)
	for _, ref := range p.keyHints {
		delete(keyHintTexts, ref.badge)
	}
	p.keyHints = nil
}
