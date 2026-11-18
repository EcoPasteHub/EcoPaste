//go:build darwin

package macos

import (
	"sync"

	"github.com/ebitengine/purego/objc"
)

// inputChrome owns the outer border state of an input: its hover and focus
// transitions and the view that draws them. Input components wrap an inner
// editable control (NSTextField / NSTextView) in an outer view, so focus
// must be tracked from the inner control and mirrored onto that view.
//
// Focus is driven by the AppKit text-editing notifications (NSControl for
// NSTextField, NSText for NSTextView), which are posted with the inner view
// as the notification object. That keeps every input's border in step with
// its field instead of each component wiring its own delegate callback.
type inputChrome struct {
	view  objc.ID
	field objc.ID

	focus  transition
	hover  transition
	radius float64
}

var (
	registerInputChromeOnce sync.Once

	chromeByView  = map[objc.ID]*inputChrome{}
	chromeByField = map[objc.ID]*inputChrome{}
)

// newInputChrome binds a field to the outer view that draws its chrome. The
// view redraws whenever the hover or focus transition moves.
func newInputChrome(view, field objc.ID) *inputChrome {
	registerInputChromeOnce.Do(registerInputChromeObserver)
	c := &inputChrome{view: view, field: field}
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

// syncFocus sets the focus transition and repaints immediately so the border
// follows the field even before the shared transition timer ticks.
func (c *inputChrome) syncFocus(focused bool) {
	target := 0.0
	if focused {
		target = 1
	}
	c.focus.setTarget(target)
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
	focus := c.focus.value
	// Reflect the real editor state so the border tracks the inner field even
	// if a transition frame has not landed yet. This keeps focus visible on
	// every input, independently of the shared transition timer.
	if inputFieldFocused(c.field) {
		focus = 1
	}
	drawSettingsInputChrome(view, c.hover.value, focus, c.radius)
}

// inputFieldFocused reports whether a text field (via its field editor) or a
// text view is the window's current first responder.
func inputFieldFocused(field objc.ID) bool {
	if field == 0 {
		return false
	}
	if objc.Send[bool](field, sel("respondsToSelector:"), sel("currentEditor")) {
		return objc.Send[objc.ID](field, sel("currentEditor")) != 0
	}
	window := objc.Send[objc.ID](field, sel("window"))
	if window == 0 {
		return false
	}
	return objc.Send[objc.ID](window, sel("firstResponder")) == field
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
}
