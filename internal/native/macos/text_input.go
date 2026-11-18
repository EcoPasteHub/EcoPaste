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
	field.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("labelColor")))
	field.Send(sel("setUsesSingleLineMode:"), true)
	field.Send(sel("setLineBreakMode:"), int64(4))
	field.Send(sel("setDelegate:"), textInputDelegate)
	control.Send(sel("addSubview:"), field)
	field.Send(sel("release"))

	t := &textInput{control: control, field: field}
	t.chrome = newInputChrome(control, field)
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

func (t *textInput) stringValue() string {
	return goString(t.field.Send(sel("stringValue")))
}

func (t *textInput) setAccessibilityLabel(label string) {
	t.control.Send(sel("setAccessibilityLabel:"), nsString(label))
}

func (t *textInput) setAccessibilityHelp(help string) {
	t.control.Send(sel("setAccessibilityHelp:"), nsString(help))
}

func (t *textInput) setReturnHandler(fn func()) { t.onReturn = fn }

func (t *textInput) setCancelHandler(fn func()) { t.onCancel = fn }

// focus moves key focus to the input and selects its contents.
func (t *textInput) focus() {
	window := objc.Send[objc.ID](t.control, sel("window"))
	if window == 0 {
		return
	}
	window.Send(sel("makeKeyWindow"))
	window.Send(sel("makeFirstResponder:"), t.field)
	t.field.Send(sel("selectText:"), objc.ID(0))
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
				area := objc.ID(class("NSTrackingArea")).Send(sel("alloc")).
					Send(sel("initWithRect:options:owner:userInfo:"), CGRect{}, uint64(1|128|512), self, objc.ID(0))
				self.Send(sel("addTrackingArea:"), area)
				area.Send(sel("release"))
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
