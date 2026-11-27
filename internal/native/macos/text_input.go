//go:build darwin

package macos

import (
	"sync"

	"github.com/ebitengine/purego/objc"
)

// textInput is the shared single-line input for in-app forms. Its border,
// corner radius, background and default/hover/focus states all come from
// drawSettingsInputChrome (through inputChrome) -- the same chrome the
// settings numeric field and shortcut recorder draw -- so every input looks
// identical. New Input components must reuse this widget instead of drawing
// their own chrome.
const (
	textInputTextSize   = 13.0
	textInputTextInset  = 10.0
	textInputTextHeight = 18.0
)

type textInput struct {
	control objc.ID
	field   objc.ID
	chrome  *inputChrome

	onReturn func()
	onCancel func()
}

var (
	registerTextInputOnce sync.Once
	textInputClass        objc.Class
	textInputDelegate     objc.ID

	textInputs      = map[objc.ID]*textInput{} // control view -> input
	textInputFields = map[objc.ID]*textInput{} // field -> input
)

// newTextInput builds a single-line input of the given height. The height
// should follow settingsControlHeight so it matches the other form controls.
func newTextInput(height float64) *textInput {
	registerTextInputOnce.Do(registerTextInputTargets)

	control := objc.ID(textInputClass).Send(sel("alloc")).
		Send(sel("initWithFrame:"), CGRect{Size: CGSize{Width: 0, Height: height}})
	control.Send(sel("setAccessibilityElement:"), true)
	control.Send(sel("setAccessibilityRole:"), nsString("AXTextField"))

	field := objc.ID(classNSTextField).Send(sel("alloc")).Send(sel("initWithFrame:"), CGRect{})
	field.Send(sel("setEditable:"), true)
	field.Send(sel("setSelectable:"), true)
	field.Send(sel("setBezeled:"), false)
	field.Send(sel("setBordered:"), false)
	field.Send(sel("setDrawsBackground:"), false)
	field.Send(sel("setFocusRingType:"), int64(1))
	field.Send(sel("setFont:"), systemFont(textInputTextSize))
	field.Send(sel("setUsesSingleLineMode:"), true)
	field.Send(sel("setLineBreakMode:"), int64(4))
	field.Send(sel("setDelegate:"), textInputDelegate)
	control.Send(sel("addSubview:"), field)
	field.Send(sel("release"))

	t := &textInput{control: control, field: field}
	t.chrome = newInputChrome(control, field)
	t.applyTextColor()
	textInputs[control] = t
	textInputFields[field] = t
	t.layout()
	return t
}

func (t *textInput) setFrame(x, y, width, height float64) {
	setFrame(t.control, x, y, width, height)
	t.layout()
}

func (t *textInput) layout() {
	rect := bounds(t.control)
	width := rect.Size.Width - 2*textInputTextInset
	if width < 0 {
		width = 0
	}
	setFrame(t.field, textInputTextInset, (rect.Size.Height-textInputTextHeight)/2,
		width, textInputTextHeight)
}

func (t *textInput) setPlaceholder(text string) {
	t.field.Send(sel("setPlaceholderString:"), nsString(text))
}

func (t *textInput) setStringValue(value string) {
	t.field.Send(sel("setStringValue:"), nsString(value))
}

func (t *textInput) applyTextColor() {
	applyFieldTextColor(t.field, t.control)
}

func (t *textInput) stringValue() string {
	return goString(t.field.Send(sel("stringValue")))
}

func (t *textInput) setAccessibilityLabel(label string) {
	t.control.Send(sel("setAccessibilityLabel:"), nsString(label))
}

func (t *textInput) setReturnHandler(fn func()) { t.onReturn = fn }

func (t *textInput) setCancelHandler(fn func()) { t.onCancel = fn }

// focus moves key focus to the input and marks its border focused.
func (t *textInput) focus() {
	window := objc.Send[objc.ID](t.control, sel("window"))
	if window == 0 {
		return
	}
	window.Send(sel("makeFirstResponder:"), t.field)
	t.field.Send(sel("selectText:"), objc.ID(0))
	t.applyTextColor()
	t.chrome.syncFocus(true)
}

// focusOnShow gives a popover input a real editing session once its window is
// key. AppKit can set the field as first responder while the popover window is
// still keyless, which leaves the field without an insertion point; resigning
// first and refocusing now (while the window really is key) makes the caret
// appear. It reports false while the window is not key so the caller retries on
// the next key transition instead of guessing a delay.
func (t *textInput) focusOnShow() bool {
	window := objc.Send[objc.ID](t.control, sel("window"))
	if window == 0 {
		return false
	}
	window.Send(sel("makeKeyWindow"))
	if !objc.Send[bool](window, sel("isKeyWindow")) {
		return false
	}
	first := objc.Send[objc.ID](window, sel("firstResponder"))
	if first == t.field || objc.Send[objc.ID](t.field, sel("currentEditor")) != 0 {
		window.Send(sel("makeFirstResponder:"), objc.ID(0))
	}
	window.Send(sel("makeFirstResponder:"), t.field)
	t.field.Send(sel("selectText:"), objc.ID(0))
	t.applyTextColor()
	t.chrome.syncFocus(true)
	return true
}

// resetFocus clears any cached border focus when the input is about to be
// shown again in a reused popover.
func (t *textInput) resetFocus() {
	t.chrome.resetFocus()
}

func (t *textInput) close() {
	delete(textInputs, t.control)
	delete(textInputFields, t.field)
	t.chrome.close()
	t.control.Send(sel("release"))
	t.control = 0
	t.field = 0
}

func registerTextInputTargets() {
	var err error
	textInputClass, err = objc.RegisterClass("EcoPasteTextInput", class("NSView"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("drawRect:"), Fn: func(self objc.ID, _ objc.SEL, _ CGRect) {
				drawInputChrome(self)
			}},
			{Cmd: sel("acceptsFirstMouse:"), Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) bool { return true }},
			{Cmd: sel("mouseDown:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if t := textInputs[self]; t != nil {
					t.focus()
				}
			}},
			{Cmd: sel("updateTrackingAreas"), Fn: func(self objc.ID, _ objc.SEL) {
				sendSuper[objc.ID](self, textInputClass, sel("updateTrackingAreas"))
				ensureHoverTracking(self)
			}},
			{Cmd: sel("mouseEntered:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if t := textInputs[self]; t != nil {
					t.chrome.setHoverTarget(1)
				}
			}},
			{Cmd: sel("mouseExited:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if t := textInputs[self]; t != nil {
					t.chrome.setHoverTarget(0)
				}
			}},
			{Cmd: sel("viewDidChangeEffectiveAppearance"), Fn: func(self objc.ID, _ objc.SEL) {
				sendSuper[objc.ID](self, textInputClass, sel("viewDidChangeEffectiveAppearance"))
				if t := textInputs[self]; t != nil {
					t.applyTextColor()
				}
				self.Send(sel("setNeedsDisplay:"), true)
			}},
		})
	if err != nil {
		panic(err)
	}

	delegateClass, err := objc.RegisterClass("EcoPasteTextInputDelegate", class("NSObject"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("control:textView:doCommandBySelector:"), Fn: func(_ objc.ID, _ objc.SEL, control objc.ID, _ objc.ID, commandSelector objc.SEL) bool {
				t := textInputFields[control]
				if t == nil {
					return false
				}
				switch commandSelector {
				case sel("insertNewline:"):
					if t.onReturn != nil {
						t.onReturn()
					}
					return true
				case sel("cancelOperation:"):
					if t.onCancel != nil {
						t.onCancel()
					}
					return true
				}
				return false
			}},
		})
	if err != nil {
		panic(err)
	}
	textInputDelegate = objc.ID(delegateClass).Send(sel("new"))
}

// textArea is the shared multi-line input. It reuses inputChrome for the
// outer border and focus/hover states, while the inner NSTextView uses
// AppKit's own text and background colors: that pair is resolved together by
// the system, so typed text is always readable in light and dark appearances.
const (
	textAreaTextSize    = 13.0
	textAreaInset       = 7.0
	textAreaTextInset   = 5.0
	textAreaMinTextSize = 16.0
)

type textArea struct {
	control objc.ID
	scroll  objc.ID
	text    objc.ID
	chrome  *inputChrome

	onCancel func()
}

var (
	registerTextAreaOnce sync.Once
	textAreaClass        objc.Class
	textAreaViewClass    objc.Class

	textAreas     = map[objc.ID]*textArea{} // control -> area
	textAreaViews = map[objc.ID]*textArea{} // text view -> area
)

func newTextArea(height float64) *textArea {
	registerTextAreaOnce.Do(registerTextAreaTargets)

	control := objc.ID(textAreaClass).Send(sel("alloc")).
		Send(sel("initWithFrame:"), CGRect{Size: CGSize{Width: 0, Height: height}})
	control.Send(sel("setAccessibilityElement:"), true)
	control.Send(sel("setAccessibilityRole:"), nsString("AXTextArea"))

	width := textAreaTextSize * 20
	text := objc.ID(textAreaViewClass).Send(sel("alloc")).
		Send(sel("initWithFrame:"), CGRect{Size: CGSize{Width: width, Height: height}})
	text.Send(sel("setRichText:"), false)
	text.Send(sel("setAllowsUndo:"), true)
	text.Send(sel("setFont:"), systemFont(textAreaTextSize))
	text.Send(sel("setDrawsBackground:"), false)
	text.Send(sel("setHorizontallyResizable:"), false)
	text.Send(sel("setVerticallyResizable:"), true)
	text.Send(sel("setTextContainerInset:"), CGSize{Width: textAreaTextInset, Height: textAreaTextInset})
	if container := text.Send(sel("textContainer")); container != 0 {
		container.Send(sel("setWidthTracksTextView:"), true)
		container.Send(sel("setContainerSize:"), CGSize{Width: width, Height: 10000})
	}

	scroll := objc.ID(class("NSScrollView")).Send(sel("new"))
	scroll.Send(sel("setHasVerticalScroller:"), true)
	scroll.Send(sel("setAutohidesScrollers:"), true)
	scroll.Send(sel("setBorderType:"), int64(0))
	scroll.Send(sel("setDrawsBackground:"), true)
	scroll.Send(sel("setBackgroundColor:"), objc.ID(classNSColor).Send(sel("textBackgroundColor")))
	scroll.Send(sel("setDocumentView:"), text)
	control.Send(sel("addSubview:"), scroll)
	scroll.Send(sel("release"))
	text.Send(sel("release"))

	area := &textArea{control: control, scroll: scroll, text: text}
	area.chrome = newInputChrome(control, text)
	area.chrome.radius = 10
	textAreas[control] = area
	textAreaViews[text] = area
	area.layout()
	return area
}

func (a *textArea) setFrame(x, y, width, height float64) {
	setFrame(a.control, x, y, width, height)
	a.layout()
}

func (a *textArea) layout() {
	rect := bounds(a.control)
	inner := textAreaInset
	width := rect.Size.Width - 2*inner
	height := rect.Size.Height - 2*inner
	if width < 0 {
		width = 0
	}
	if height < textAreaMinTextSize {
		height = textAreaMinTextSize
	}
	setFrame(a.scroll, inner, inner, width, height)
	clipBounds := bounds(a.scroll.Send(sel("contentView")))
	textWidth, textHeight := clipBounds.Size.Width, clipBounds.Size.Height
	if textWidth <= 0 {
		textWidth = width
	}
	if textHeight < textAreaMinTextSize {
		textHeight = height
	}
	a.text.Send(sel("setFrame:"), CGRect{Size: CGSize{Width: textWidth, Height: textHeight}})
}

func (a *textArea) setString(value string) {
	a.text.Send(sel("setString:"), nsString(value))
}

func (a *textArea) stringValue() string {
	return goString(a.text.Send(sel("string")))
}

func (a *textArea) setAccessibilityLabel(label string) {
	a.control.Send(sel("setAccessibilityLabel:"), nsString(label))
}

func (a *textArea) setCancelHandler(fn func()) { a.onCancel = fn }

func (a *textArea) focus() {
	window := objc.Send[objc.ID](a.control, sel("window"))
	if window == 0 {
		return
	}
	window.Send(sel("makeFirstResponder:"), a.text)
	a.chrome.syncFocus(true)
}

// focusOnShow starts a real editing session for the note editor once the
// popover window is key. A first responder set while the window was keyless has
// no caret and no begin-editing notification, so resign and refocus here. It
// reports false while the window is not key so the caller can retry on the key
// transition rather than with a timer.
func (a *textArea) focusOnShow() bool {
	window := objc.Send[objc.ID](a.control, sel("window"))
	if window == 0 {
		return false
	}
	window.Send(sel("makeKeyWindow"))
	if !objc.Send[bool](window, sel("isKeyWindow")) {
		return false
	}
	if objc.Send[objc.ID](window, sel("firstResponder")) == a.text {
		window.Send(sel("makeFirstResponder:"), objc.ID(0))
	}
	window.Send(sel("makeFirstResponder:"), a.text)
	a.text.Send(sel("moveToEndOfDocument:"), objc.ID(0))
	a.chrome.syncFocus(true)
	return true
}

// resetFocus clears any cached border focus when the area is about to be shown
// again in a reused popover.
func (a *textArea) resetFocus() {
	a.chrome.resetFocus()
}

func (a *textArea) close() {
	delete(textAreas, a.control)
	delete(textAreaViews, a.text)
	a.chrome.close()
	a.control.Send(sel("release"))
	a.control = 0
	a.scroll = 0
	a.text = 0
}

func registerTextAreaTargets() {
	var err error
	textAreaClass, err = objc.RegisterClass("EcoPasteTextArea", class("NSView"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("drawRect:"), Fn: func(self objc.ID, _ objc.SEL, _ CGRect) {
				drawInputChrome(self)
			}},
			{Cmd: sel("acceptsFirstMouse:"), Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) bool { return true }},
			{Cmd: sel("mouseDown:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if a := textAreas[self]; a != nil {
					a.focus()
				}
			}},
			{Cmd: sel("updateTrackingAreas"), Fn: func(self objc.ID, _ objc.SEL) {
				sendSuper[objc.ID](self, textAreaClass, sel("updateTrackingAreas"))
				ensureHoverTracking(self)
			}},
			{Cmd: sel("mouseEntered:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if c := chromeByView[self]; c != nil {
					c.setHoverTarget(1)
				}
			}},
			{Cmd: sel("mouseExited:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if c := chromeByView[self]; c != nil {
					c.setHoverTarget(0)
				}
			}},
			{Cmd: sel("viewDidChangeEffectiveAppearance"), Fn: func(self objc.ID, _ objc.SEL) {
				sendSuper[objc.ID](self, textAreaClass, sel("viewDidChangeEffectiveAppearance"))
				self.Send(sel("setNeedsDisplay:"), true)
			}},
		})
	if err != nil {
		panic(err)
	}

	textAreaViewClass, err = objc.RegisterClass("EcoPasteTextAreaView", class("NSTextView"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("cancelOperation:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if a := textAreaViews[self]; a != nil && a.onCancel != nil {
					a.onCancel()
				}
			}},
		})
	if err != nil {
		panic(err)
	}
}

// inputTextColor keeps the single-line field ink in step with the chrome
// fill: explicit black on the white light field, white on the dark one.
func inputTextColor(view objc.ID) objc.ID {
	if appearanceIsDark(view) {
		return objc.ID(classNSColor).Send(sel("whiteColor"))
	}
	return objc.ID(classNSColor).Send(sel("blackColor"))
}

func applyFieldTextColor(field, view objc.ID) {
	field.Send(sel("setTextColor:"), inputTextColor(view))
}
