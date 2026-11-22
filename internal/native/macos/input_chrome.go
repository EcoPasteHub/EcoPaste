//go:build darwin

package macos

import (
	"sync"

	"github.com/ebitengine/purego/objc"
)

// inputChrome owns the outer border state of an input: its hover and focus
// transitions and the view that draws them. Input components wrap an inner
// editable control (NSTextField / NSTextView) in an outer view, so focus is
// mirrored onto that outer view.
//
// Focus is derived from the real editor state (the key window's first responder
// and, for NSTextField, its field editor) instead of from cached notifications.
// Editing notifications and window key changes only trigger a re-sync; the
// target itself always comes from the field. That single source of truth is
// what keeps the border and the caret in step: a programmatically focused
// NSTextView posts no begin-editing notification, and a window can become key a
// frame after its field is made first responder.
type inputChrome struct {
	view  objc.ID
	field objc.ID

	focus  transition
	hover  transition
	radius float64

	// manualFocus holds the focus state for a chrome with no bound field (the
	// shortcut recorder). Such controls drive focus from their own
	// first-responder callbacks, so the state cannot be re-derived from a field.
	manualFocus bool
}

var (
	registerInputChromeOnce sync.Once

	chromeByView  = map[objc.ID]*inputChrome{}
	chromeByField = map[objc.ID]*inputChrome{}

	// inputFocusProbe reads the real editor state. It is a variable so tests can
	// exercise the derivation without a live key window and field editor.
	inputFocusProbe = inputFieldFocused
)

// newInputChrome binds a field to the outer view that draws its chrome. The
// view redraws whenever the hover or focus transition moves.
func newInputChrome(view, field objc.ID) *inputChrome {
	registerInputChromeOnce.Do(registerInputChromeObserver)
	c := &inputChrome{view: view, field: field}
	c.focus.view, c.hover.view = view, view
	c.focus.redraw = func() { view.Send(sel("setNeedsDisplay:"), true) }
	c.hover.redraw = c.focus.redraw
	chromeByView[view] = c
	if field != 0 {
		chromeByField[field] = c
	}
	return c
}

func (c *inputChrome) setHoverTarget(target float64) {
	c.hover.setTarget(target)
}

// syncFocus aligns the border focus with the real editor state and repaints
// immediately so the border follows the field even before the shared
// transition timer ticks. For field-backed inputs the real editing state is the
// single source of truth; the explicit argument only matters for controls with
// no bound field (the shortcut recorder passes field==0 and drives focus from
// its own first-responder callbacks).
//
// Deriving focus here instead of trusting an event or a cached target is what
// keeps the border honest: programmatically focusing an NSTextView posts no
// begin-editing notification, and a popover's window can become key a frame
// after the field is made first responder. Both cases previously left the
// border out of step with the caret.
func (c *inputChrome) syncFocus(focused bool) {
	if c.field != 0 {
		focused = inputFocusProbe(c.field)
	} else {
		c.manualFocus = focused
	}
	c.applyFocus(focused)
	c.view.Send(sel("setNeedsDisplay:"), true)
}

// applyFocus moves the transition toward the given focus state without
// scheduling a repaint. drawInputChrome uses it mid-paint so the target always
// tracks reality while the painted value animates toward it.
func (c *inputChrome) applyFocus(focused bool) {
	target := 0.0
	if focused {
		target = 1
	}
	c.focus.setTarget(target)
}

// resetFocus clears any cached focus without animating. A popover reuses its
// input across openings, so without this a stale focused value from the last
// session would flash before the field is actually focused again.
func (c *inputChrome) resetFocus() {
	c.focus.stop()
	c.focus.value = 0
	c.focus.target = 0
	c.manualFocus = false
	c.view.Send(sel("setNeedsDisplay:"), true)
}

func (c *inputChrome) close() {
	delete(chromeByView, c.view)
	if c.field != 0 {
		delete(chromeByField, c.field)
	}
	c.focus.stop()
	c.hover.stop()
}

// drawInputChrome is the shared drawRect body for input chrome views.
func drawInputChrome(view objc.ID) {
	c := chromeByView[view]
	if c == nil {
		return
	}
	// Keep the transition target equal to the real editor state before
	// painting. This is the authoritative sync: it corrects a stale target from
	// a previous popover session and a begin-editing event that arrived while
	// the popover window was not key yet.
	if c.field != 0 {
		c.applyFocus(inputFocusProbe(c.field))
	}
	drawSettingsInputChrome(view, c.hover.value, c.focus.value, c.radius)
}

// inputFieldFocused reports whether a text field (via its field editor) or a
// text view is the key window's current editing responder. Requiring the window
// to be key is what ties the border to the visible insertion point: a control
// can be first responder while its window is not key, in which case it draws no
// caret, so the chrome must not claim focus either.
func inputFieldFocused(field objc.ID) bool {
	if field == 0 {
		return false
	}
	window := objc.Send[objc.ID](field, sel("window"))
	if window == 0 || !objc.Send[bool](window, sel("isKeyWindow")) {
		return false
	}
	first := objc.Send[objc.ID](window, sel("firstResponder"))
	if first == 0 {
		return false
	}
	if objc.Send[bool](field, sel("respondsToSelector:"), sel("currentEditor")) {
		editor := objc.Send[objc.ID](field, sel("currentEditor"))
		return editor != 0 && editor == first
	}
	return first == field
}

// resyncChromesForWindow re-derives focus for every input living in window. A
// window becoming or resigning key flips the caret on or off without any
// editing notification, so the borders have to follow explicitly. Field-backed
// inputs consult the editor state; fieldless controls (the shortcut recorder)
// keep their manual focus but go dark while their window is not key.
func resyncChromesForWindow(window objc.ID) {
	if window == 0 {
		return
	}
	key := objc.Send[bool](window, sel("isKeyWindow"))
	for _, c := range chromeByView {
		owner := c.field
		if owner == 0 {
			owner = c.view
		}
		if objc.Send[objc.ID](owner, sel("window")) != window {
			continue
		}
		if c.field != 0 {
			c.applyFocus(inputFocusProbe(c.field))
		} else {
			c.applyFocus(c.manualFocus && key)
		}
		c.view.Send(sel("setNeedsDisplay:"), true)
	}
}

// focusedInputChrome resolves a control-editing notification to its chrome.
func focusedInputChrome(note objc.ID) *inputChrome {
	return chromeByField[objc.Send[objc.ID](note, sel("object"))]
}

func registerInputChromeObserver() {
	observerClass, err := objc.RegisterClass("EcoPasteInputChromeObserver", class("NSObject"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("controlTextDidBeginEditing:"), Fn: func(_ objc.ID, _ objc.SEL, note objc.ID) {
				if c := focusedInputChrome(note); c != nil {
					c.syncFocus(true)
				}
			}},
			{Cmd: sel("controlTextDidEndEditing:"), Fn: func(_ objc.ID, _ objc.SEL, note objc.ID) {
				if c := focusedInputChrome(note); c != nil {
					c.syncFocus(false)
				}
			}},
			// A window becoming or resigning key turns the caret on or off with
			// no editing notification, so re-derive every input in that window.
			{Cmd: sel("windowFocusChanged:"), Fn: func(_ objc.ID, _ objc.SEL, note objc.ID) {
				resyncChromesForWindow(objc.Send[objc.ID](note, sel("object")))
			}},
		})
	if err != nil {
		panic(err)
	}
	observer := objc.ID(observerClass).Send(sel("new"))
	center := objc.ID(class("NSNotificationCenter")).Send(sel("defaultCenter"))
	for _, name := range []string{
		"NSControlTextDidBeginEditingNotification",
		"NSTextDidBeginEditingNotification",
	} {
		center.Send(sel("addObserver:selector:name:object:"), observer,
			sel("controlTextDidBeginEditing:"), nsString(name), objc.ID(0))
	}
	for _, name := range []string{
		"NSControlTextDidEndEditingNotification",
		"NSTextDidEndEditingNotification",
	} {
		center.Send(sel("addObserver:selector:name:object:"), observer,
			sel("controlTextDidEndEditing:"), nsString(name), objc.ID(0))
	}
	for _, name := range []string{
		"NSWindowDidBecomeKeyNotification",
		"NSWindowDidResignKeyNotification",
	} {
		center.Send(sel("addObserver:selector:name:object:"), observer,
			sel("windowFocusChanged:"), nsString(name), objc.ID(0))
	}
}
