//go:build darwin

package macos

import (
	"math"
	"sync"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

const (
	shortcutRecorderWidth    = 120.0
	shortcutRecorderHeight   = settingsControlHeight
	shortcutDeleteButtonSize = 18.0
	shortcutRecorderInset    = 14.0
	shortcutDeleteRightInset = 7.0
)

type shortcutRecorder struct {
	owner   *SettingsView
	path    string
	title   i18n.Key
	control objc.ID
	delete  objc.ID
	get     func(SettingsValues) string
	value   string

	recording  bool
	draft      shortcutRecording
	submitting bool
	attempted  bool
	monitor    objc.ID

	focus       float64
	focusTarget float64
	hover       float64
	hoverTarget float64
	timer       objc.ID
}

var (
	registerShortcutRecorderOnce sync.Once
	shortcutRecorderClass        objc.Class
	shortcutRecorders            = map[objc.ID]*shortcutRecorder{}
	shortcutDeleteButtons        = map[objc.ID]*shortcutRecorder{}
	shortcutRecorderTimers       = map[objc.ID]*shortcutRecorder{}
)

func (v *SettingsView) addShortcutField(content objc.ID, y float64, title i18n.Key, path string, get func(SettingsValues) string) {
	registerShortcutRecorderOnce.Do(registerShortcutRecorderTargets)
	v.addLabel(content, y, title)
	control := objc.ID(shortcutRecorderClass).Send(sel("alloc")).Send(sel("initWithFrame:"), CGRect{
		Origin: CGPoint{
			X: settingsContentX + settingsContentWidth - settingsRowInset - shortcutRecorderWidth,
			Y: y + (settingsRowHeight-shortcutRecorderHeight)/2,
		},
		Size: CGSize{Width: shortcutRecorderWidth, Height: shortcutRecorderHeight},
	})
	control.Send(sel("setAccessibilityElement:"), true)
	control.Send(sel("setAccessibilityRole:"), nsString("AXTextField"))
	control.Send(sel("setAccessibilityLabel:"), nsString(i18n.Tf(i18n.SettingsShortcutFieldA11y, i18n.T(title))))
	control.Send(sel("setAccessibilityHelp:"), nsString(i18n.T(i18n.SettingsShortcutHelp)))
	content.Send(sel("addSubview:"), control)

	recorder := &shortcutRecorder{
		owner:   v,
		path:    path,
		title:   title,
		control: control,
		get:     get,
	}
	shortcutRecorders[control] = recorder
	recorder.delete = newShortcutDeleteButton(recorder)
	control.Send(sel("addSubview:"), recorder.delete)
	v.shortcutRecorders = append(v.shortcutRecorders, recorder)
	recorder.refresh()
}

func newShortcutDeleteButton(recorder *shortcutRecorder) objc.ID {
	button := objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(""), settingsTarget, sel("shortcutDeletePressed:"))
	button.Send(sel("setBordered:"), false)
	button.Send(sel("setImage:"), shortcutDeleteSymbol())
	button.Send(sel("setImagePosition:"), int64(1))
	button.Send(sel("setImageScaling:"), int64(2))
	button.Send(sel("setContentTintColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	button.Send(sel("setFocusRingType:"), int64(1))
	button.Send(sel("setRefusesFirstResponder:"), true)
	button.Send(sel("setToolTip:"), nsString(i18n.T(i18n.SettingsShortcutClear)))
	button.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.SettingsShortcutClear)))
	button.Send(sel("setAccessibilityHelp:"), nsString(i18n.T(i18n.SettingsShortcutClearHelp)))
	shortcutDeleteButtons[button] = recorder
	return button
}

func shortcutDeleteSymbol() objc.ID {
	image := systemSymbol("xmark.circle.fill")
	image.Send(sel("setSize:"), CGSize{Width: 12, Height: 12})
	return image
}

func (v *SettingsView) applyShortcuts(values SettingsValues) {
	for _, recorder := range v.shortcutRecorders {
		if recorder.recording {
			continue
		}
		recorder.value = recorder.get(values)
		recorder.refresh()
	}
}

func (r *shortcutRecorder) refresh() {
	r.layoutDeleteButton()
	r.control.Send(sel("setAccessibilityLabel:"),
		nsString(i18n.Tf(i18n.SettingsShortcutFieldA11y, i18n.T(r.title))))
	r.delete.Send(sel("setToolTip:"), nsString(i18n.T(i18n.SettingsShortcutClear)))
	r.delete.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.SettingsShortcutClear)))
	r.delete.Send(sel("setAccessibilityHelp:"), nsString(i18n.T(i18n.SettingsShortcutClearHelp)))
	r.delete.Send(sel("setHidden:"), r.recording || r.value == "")
	text, _, _ := r.presentation()
	help := i18n.T(i18n.SettingsShortcutHelp)
	r.control.Send(sel("setAccessibilityValue:"), nsString(text))
	r.control.Send(sel("setAccessibilityHelp:"), nsString(help))
	r.control.Send(sel("setToolTip:"), nsString(help))
	r.control.Send(sel("setNeedsDisplay:"), true)
}

func (r *shortcutRecorder) layoutDeleteButton() {
	rect := bounds(r.control)
	size := shortcutDeleteButtonSize
	r.delete.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{
			X: rect.Size.Width - shortcutDeleteRightInset - size,
			Y: (rect.Size.Height - size) / 2,
		},
		Size: CGSize{Width: size, Height: size},
	})
}

func (r *shortcutRecorder) presentation() (string, objc.ID, float64) {
	secondary := objc.ID(classNSColor).Send(sel("secondaryLabelColor"))
	if r.recording {
		if text := r.draft.display(); text != "" {
			return text, objc.ID(classNSColor).Send(sel("controlAccentColor")), 13
		}
		return i18n.T(i18n.SettingsRecording), secondary, 11.5
	}
	if r.value != "" {
		return formatShortcutGlyphs(r.value), objc.ID(classNSColor).Send(sel("controlAccentColor")), 13
	}
	return i18n.T(i18n.SettingsShortcutIdle), secondary, 11.5
}

func (r *shortcutRecorder) beginRecording() {
	if r.recording {
		return
	}
	r.recording = true
	r.draft = shortcutRecording{}
	r.submitting = false
	r.attempted = false
	r.delete.Send(sel("setHidden:"), true)
	r.installBlurMonitor()
	r.owner.cx.Emit("suspendShortcuts", true)
	r.setFocusTarget(1)
	r.refresh()
}

func (r *shortcutRecorder) cancelRecording() {
	if !r.recording {
		return
	}
	r.recording = false
	r.draft = shortcutRecording{}
	r.submitting = false
	r.attempted = false
	r.removeBlurMonitor()
	r.refresh()
	r.owner.cx.Emit("suspendShortcuts", false)
}

func (r *shortcutRecorder) submitRecording(value string) {
	if !r.recording || r.submitting || r.attempted {
		return
	}
	r.submitting = true
	r.attempted = true
	r.refresh()
	r.owner.cx.Emit("change", SettingsChange{Path: r.path, Value: value})
}

func (r *shortcutRecorder) acceptRecording(value string) {
	r.submitting = false
	r.attempted = false
	r.value = value
	if !r.recording {
		r.refresh()
		return
	}
	r.recording = false
	r.draft = shortcutRecording{}
	r.removeBlurMonitor()
	r.refresh()
	r.setFocusTarget(0)
	if window := objc.Send[objc.ID](r.control, sel("window")); window != 0 {
		window.Send(sel("makeFirstResponder:"), objc.ID(0))
	}
}

func (r *shortcutRecorder) rejectRecording() {
	if !r.recording {
		return
	}
	r.submitting = false
	r.refresh()
}

func (r *shortcutRecorder) clear() {
	if r.recording {
		return
	}
	// The empty value is applied through the settings store so a failed
	// update cannot leave the field out of sync with the registration.
	r.owner.cx.Emit("change", SettingsChange{Path: r.path, Value: ""})
}

func (r *shortcutRecorder) capture(event objc.ID) {
	if !r.recording {
		return
	}
	keyCode := objc.Send[uint16](event, sel("keyCode"))
	flags := uint64(objc.Send[uint](event, sel("modifierFlags")))
	r.draft.updateModifiers(shortcutModifiersFromFlags(flags))
	if keyCode == keyEscape {
		r.cancelRecording()
		if window := objc.Send[objc.ID](r.control, sel("window")); window != 0 {
			window.Send(sel("makeFirstResponder:"), objc.ID(0))
		}
		return
	}
	if (keyCode == keyDelete || keyCode == keyForwardDelete) && flags&shortcutModifierMask() == 0 {
		r.submitRecording("")
		return
	}
	name := shortcutKeyName(keyCode)
	if name == "" {
		name = shortcutCharacter(goString(objc.Send[objc.ID](event, sel("charactersIgnoringModifiers"))))
	}
	if name == "" {
		return
	}
	if !r.draft.holdsKey(name) {
		r.attempted = false
	}
	r.draft.press(name)
	r.refresh()
}

func (r *shortcutRecorder) installBlurMonitor() {
	if r.monitor != 0 {
		return
	}
	block := objc.NewBlock(func(_ objc.Block, event objc.ID) objc.ID {
		if !r.recording || r.pointInside(event) {
			return event
		}
		if window := objc.Send[objc.ID](r.control, sel("window")); window != 0 {
			window.Send(sel("makeFirstResponder:"), objc.ID(0))
		}
		return event
	})
	r.monitor = objc.ID(class("NSEvent")).Send(
		sel("addLocalMonitorForEventsMatchingMask:handler:"),
		uint64(1<<eventLeftMouseDown), objc.ID(block))
	block.Release()
}

func (r *shortcutRecorder) pointInside(event objc.ID) bool {
	window := objc.Send[objc.ID](r.control, sel("window"))
	eventWindow := objc.Send[objc.ID](event, sel("window"))
	if window == 0 || eventWindow != window {
		return false
	}
	point := objc.Send[CGPoint](event, sel("locationInWindow"))
	local := objc.Send[CGPoint](r.control, sel("convertPoint:fromView:"), point, objc.ID(0))
	rect := bounds(r.control)
	return local.X >= 0 && local.Y >= 0 && local.X <= rect.Size.Width && local.Y <= rect.Size.Height
}

func (r *shortcutRecorder) removeBlurMonitor() {
	if r.monitor == 0 {
		return
	}
	objc.ID(class("NSEvent")).Send(sel("removeMonitor:"), r.monitor)
	r.monitor = 0
}

func shortcutModifiersFromFlags(flags uint64) []string {
	modifiers := make([]string, 0, 4)
	if flags&commandModifier != 0 {
		modifiers = append(modifiers, "command")
	}
	if flags&controlModifier != 0 {
		modifiers = append(modifiers, "control")
	}
	if flags&alternateModifier != 0 {
		modifiers = append(modifiers, "alt")
	}
	if flags&shiftModifier != 0 {
		modifiers = append(modifiers, "shift")
	}
	return modifiers
}

// shortcutModifiersReduced reports whether before contained modifiers that
// are no longer down in after. Releasing a modifier finalizes the currently
// held combination instead of dropping the non-modifier key.
func shortcutModifiersReduced(before, after []string) bool {
	if len(after) >= len(before) {
		return false
	}
	set := map[string]bool{}
	for _, name := range after {
		set[name] = true
	}
	for _, name := range before {
		if !set[name] {
			return true
		}
	}
	return false
}

func shortcutModifierMask() uint64 {
	return commandModifier | controlModifier | alternateModifier | shiftModifier
}

func (r *shortcutRecorder) setFocusTarget(target float64) {
	if r.focusTarget == target {
		return
	}
	r.focusTarget = target
	r.startAnimation()
}

func (r *shortcutRecorder) setHoverTarget(target float64) {
	if r.hoverTarget == target {
		return
	}
	r.hoverTarget = target
	r.startAnimation()
}

func (r *shortcutRecorder) startAnimation() {
	if r.timer != 0 {
		return
	}
	r.timer = objc.ID(class("NSTimer")).Send(sel("scheduledTimerWithTimeInterval:target:selector:userInfo:repeats:"),
		1.0/60.0, settingsTarget, sel("shortcutRecorderTick:"), objc.ID(0), true)
	shortcutRecorderTimers[r.timer] = r
}

func (r *shortcutRecorder) advanceAnimation() {
	r.focus = shortcutApproach(r.focus, r.focusTarget)
	r.hover = shortcutApproach(r.hover, r.hoverTarget)
	r.control.Send(sel("setNeedsDisplay:"), true)
	if math.Abs(r.focus-r.focusTarget) > 0.001 || math.Abs(r.hover-r.hoverTarget) > 0.001 {
		return
	}
	r.focus = r.focusTarget
	r.hover = r.hoverTarget
	r.control.Send(sel("setNeedsDisplay:"), true)
	r.stopAnimation()
}

func shortcutApproach(current, target float64) float64 {
	delta := target - current
	if math.Abs(delta) < 0.02 {
		return target
	}
	return current + delta*0.28
}

func (r *shortcutRecorder) stopAnimation() {
	if r.timer == 0 {
		return
	}
	r.timer.Send(sel("invalidate"))
	delete(shortcutRecorderTimers, r.timer)
	r.timer = 0
}

func (r *shortcutRecorder) close() {
	if r.recording {
		r.recording = false
		r.removeBlurMonitor()
		r.owner.cx.Emit("suspendShortcuts", false)
	}
	r.stopAnimation()
}

func (v *SettingsView) closeShortcutRecorders() {
	for _, recorder := range v.shortcutRecorders {
		recorder.close()
		delete(shortcutRecorders, recorder.control)
		if recorder.delete != 0 {
			delete(shortcutDeleteButtons, recorder.delete)
		}
	}
	v.shortcutRecorders = nil
}

func registerShortcutRecorderTargets() {
	var err error
	shortcutRecorderClass, err = objc.RegisterClass("EcoPasteShortcutRecorder", class("NSView"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("drawRect:"), Fn: drawShortcutRecorder},
			{Cmd: sel("acceptsFirstResponder"), Fn: func(_ objc.ID, _ objc.SEL) bool { return true }},
			{Cmd: sel("acceptsFirstMouse:"), Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) bool { return true }},
			{Cmd: sel("becomeFirstResponder"), Fn: func(self objc.ID, _ objc.SEL) bool {
				if recorder := shortcutRecorders[self]; recorder != nil {
					recorder.beginRecording()
				}
				return true
			}},
			{Cmd: sel("resignFirstResponder"), Fn: func(self objc.ID, _ objc.SEL) bool {
				if recorder := shortcutRecorders[self]; recorder != nil {
					recorder.cancelRecording()
					recorder.setFocusTarget(0)
				}
				return true
			}},
			{Cmd: sel("mouseDown:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if window := objc.Send[objc.ID](self, sel("window")); window != 0 {
					window.Send(sel("makeFirstResponder:"), self)
				}
			}},
			{Cmd: sel("keyDown:"), Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
				if recorder := shortcutRecorders[self]; recorder != nil {
					recorder.capture(event)
				}
			}},
			{Cmd: sel("keyUp:"), Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
				recorder := shortcutRecorders[self]
				if recorder == nil || !recorder.recording {
					return
				}
				keyCode := objc.Send[uint16](event, sel("keyCode"))
				name := shortcutKeyName(keyCode)
				if name == "" {
					name = shortcutCharacter(goString(objc.Send[objc.ID](event, sel("charactersIgnoringModifiers"))))
				}
				if recorder.draft.holdsKey(name) {
					if value := recorder.draft.complete(); value != "" {
						recorder.submitRecording(value)
					}
				}
				recorder.draft.release(name)
				flags := uint64(objc.Send[uint](event, sel("modifierFlags")))
				recorder.draft.updateModifiers(shortcutModifiersFromFlags(flags))
				recorder.refresh()
			}},
			{Cmd: sel("flagsChanged:"), Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
				recorder := shortcutRecorders[self]
				if recorder == nil || !recorder.recording {
					return
				}
				flags := uint64(objc.Send[uint](event, sel("modifierFlags")))
				next := shortcutModifiersFromFlags(flags)
				if shortcutModifiersReduced(recorder.draft.modifiers, next) {
					if value := recorder.draft.complete(); value != "" {
						recorder.submitRecording(value)
					}
				} else if len(next) > len(recorder.draft.modifiers) {
					recorder.attempted = false
				}
				recorder.draft.updateModifiers(next)
				recorder.refresh()
			}},
			{Cmd: sel("performKeyEquivalent:"), Fn: func(self objc.ID, _ objc.SEL, event objc.ID) bool {
				recorder := shortcutRecorders[self]
				if recorder == nil || !recorder.recording {
					return false
				}
				recorder.capture(event)
				return true
			}},
			{Cmd: sel("cancelOperation:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				recorder := shortcutRecorders[self]
				if recorder == nil || !recorder.recording {
					return
				}
				recorder.cancelRecording()
				if window := objc.Send[objc.ID](self, sel("window")); window != 0 {
					window.Send(sel("makeFirstResponder:"), objc.ID(0))
				}
			}},
			{Cmd: sel("updateTrackingAreas"), Fn: func(self objc.ID, _ objc.SEL) {
				sendSuper[objc.ID](self, shortcutRecorderClass, sel("updateTrackingAreas"))
				area := objc.ID(class("NSTrackingArea")).Send(sel("alloc")).
					Send(sel("initWithRect:options:owner:userInfo:"), CGRect{}, uint64(1|128|512), self, objc.ID(0))
				self.Send(sel("addTrackingArea:"), area)
				area.Send(sel("release"))
			}},
			{Cmd: sel("mouseEntered:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if recorder := shortcutRecorders[self]; recorder != nil {
					recorder.setHoverTarget(1)
				}
			}},
			{Cmd: sel("mouseExited:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if recorder := shortcutRecorders[self]; recorder != nil {
					recorder.setHoverTarget(0)
				}
			}},
			{Cmd: sel("viewDidChangeEffectiveAppearance"), Fn: func(self objc.ID, _ objc.SEL) {
				sendSuper[objc.ID](self, shortcutRecorderClass, sel("viewDidChangeEffectiveAppearance"))
				self.Send(sel("setNeedsDisplay:"), true)
			}},
		})
	if err != nil {
		panic(err)
	}
}

func drawShortcutRecorder(self objc.ID, _ objc.SEL, _ CGRect) {
	recorder := shortcutRecorders[self]
	if recorder == nil {
		return
	}
	rect := bounds(self)
	rect.Origin.X += 0.5
	rect.Origin.Y += 0.5
	rect.Size.Width -= 1
	rect.Size.Height -= 1
	radius := rect.Size.Height / 2

	dark := appearanceIsDark(self)
	background := objc.ID(classNSColor).Send(sel("whiteColor"))
	if dark {
		background = background.Send(sel("colorWithAlphaComponent:"), 0.075)
	}
	fillRoundedRect(rect, radius, background)

	border := objc.ID(classNSColor).Send(sel("separatorColor")).
		Send(sel("blendedColorWithFraction:ofColor:"), recorder.hover,
			objc.ID(classNSColor).Send(sel("controlAccentColor"))).
		Send(sel("blendedColorWithFraction:ofColor:"), recorder.focus,
			objc.ID(classNSColor).Send(sel("controlAccentColor")))
	path := objc.ID(class("NSBezierPath")).Send(sel("bezierPathWithRoundedRect:xRadius:yRadius:"),
		rect, radius, radius)
	path.Send(sel("setLineWidth:"), 1.0+0.6*recorder.focus)
	border.Send(sel("setStroke"))
	path.Send(sel("stroke"))

	text, color, size := recorder.presentation()
	if text == "" {
		return
	}
	left := shortcutRecorderInset
	right := rect.Size.Width - shortcutRecorderInset
	if !objc.Send[bool](recorder.delete, sel("isHidden")) {
		frame := objc.Send[CGRect](recorder.delete, sel("frame"))
		right = frame.Origin.X - 4
	}
	available := right - left
	if available <= 0 {
		return
	}
	font := objc.ID(classNSFont).Send(sel("systemFontOfSize:weight:"), size, shortcutFontWeight(recorder))
	attributes := objc.ID(class("NSMutableDictionary")).Send(sel("dictionary"))
	attributes.Send(sel("setObject:forKey:"), font, nsString("NSFont"))
	attributes.Send(sel("setObject:forKey:"), color, nsString("NSColor"))
	measured := objc.Send[CGSize](nsString(text), sel("sizeWithAttributes:"), attributes)
	point := CGPoint{
		X: left + (available-measured.Width)/2,
		Y: (rect.Size.Height - measured.Height) / 2,
	}
	nsString(text).Send(sel("drawAtPoint:withAttributes:"), point, attributes)
}

func shortcutFontWeight(recorder *shortcutRecorder) float64 {
	if recorder.recording && recorder.draft.display() != "" {
		return 0.23
	}
	if !recorder.recording && recorder.value != "" {
		return 0.23
	}
	return 0.0
}
