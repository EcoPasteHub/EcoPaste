//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

func registerNumberInputTargets() {
	var err error
	numberInputClass, err = objc.RegisterClass("EcoPasteNumberInput", class("NSView"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("drawRect:"), Fn: drawNumberInput},
			{Cmd: sel("acceptsFirstMouse:"), Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) bool { return true }},
			{Cmd: sel("mouseDown:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if n := numberInputs[self]; n != nil {
					n.beginEditing()
					if window := objc.Send[objc.ID](self, sel("window")); window != 0 {
						window.Send(sel("makeFirstResponder:"), n.field)
					}
				}
			}},
			{Cmd: sel("updateTrackingAreas"), Fn: func(self objc.ID, _ objc.SEL) {
				sendSuper[objc.ID](self, numberInputClass, sel("updateTrackingAreas"))
				ensureHoverTracking(self)
			}},
			{Cmd: sel("mouseEntered:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if n := numberInputs[self]; n != nil {
					n.setHoverTarget(1)
				}
			}},
			{Cmd: sel("mouseExited:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if n := numberInputs[self]; n != nil {
					n.setHoverTarget(0)
				}
			}},
			{Cmd: sel("viewDidChangeEffectiveAppearance"), Fn: func(self objc.ID, _ objc.SEL) {
				sendSuper[objc.ID](self, numberInputClass, sel("viewDidChangeEffectiveAppearance"))
				self.Send(sel("setNeedsDisplay:"), true)
			}},
			{Cmd: sel("stepDown:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if n := numberInputs[self]; n != nil {
					n.step(-n.stepValue)
				}
			}},
			{Cmd: sel("stepUp:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if n := numberInputs[self]; n != nil {
					n.step(n.stepValue)
				}
			}},
		})
	if err != nil {
		panic(err)
	}

	numberTextFieldClass, err = objc.RegisterClass("EcoPasteNumberTextField", classNSTextField, nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("acceptsFirstMouse:"), Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) bool { return true }},
			{Cmd: sel("mouseDown:"), Fn: func(self objc.ID, cmd objc.SEL, event objc.ID) {
				if n := numberTextFields[self]; n != nil {
					n.beginEditing()
				}
				sendSuper[objc.ID](self, numberTextFieldClass, cmd, event)
			}},
		})
	if err != nil {
		panic(err)
	}

	numberTextFieldDelegateClass, err = objc.RegisterClass("EcoPasteNumberTextFieldDelegate", class("NSObject"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("controlTextDidBeginEditing:"), Fn: func(_ objc.ID, _ objc.SEL, notification objc.ID) {
				if n := numberTextFields[objc.Send[objc.ID](notification, sel("object"))]; n != nil {
					n.beginEditing()
				}
			}},
			{Cmd: sel("controlTextDidChange:"), Fn: func(_ objc.ID, _ objc.SEL, notification objc.ID) {
				if n := numberTextFields[objc.Send[objc.ID](notification, sel("object"))]; n != nil && n.editing {
					n.sanitizeText()
				}
			}},
			{Cmd: sel("controlTextDidEndEditing:"), Fn: func(_ objc.ID, _ objc.SEL, notification objc.ID) {
				if n := numberTextFields[objc.Send[objc.ID](notification, sel("object"))]; n != nil {
					n.endEditing()
				}
			}},
		})
	if err != nil {
		panic(err)
	}
	numberTextFieldDelegate = objc.ID(numberTextFieldDelegateClass).Send(sel("new"))

	numberTextObserverClass, err = objc.RegisterClass("EcoPasteNumberTextObserver", class("NSObject"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("textDidBeginEditing:"), Fn: func(_ objc.ID, _ objc.SEL, notification objc.ID) {
				editor := objc.Send[objc.ID](notification, sel("object"))
				if n := numberInputForEditor(editor); n != nil {
					numberEditors[editor] = n
					n.beginEditing()
				}
			}},
			{Cmd: sel("textDidEndEditing:"), Fn: func(_ objc.ID, _ objc.SEL, notification objc.ID) {
				editor := objc.Send[objc.ID](notification, sel("object"))
				if n := numberEditors[editor]; n != nil {
					delete(numberEditors, editor)
					n.endEditing()
				}
			}},
			{Cmd: sel("textViewSelectionDidChange:"), Fn: func(_ objc.ID, _ objc.SEL, notification objc.ID) {
				editor := objc.Send[objc.ID](notification, sel("object"))
				if n := numberInputForEditor(editor); n != nil {
					numberEditors[editor] = n
					n.beginEditing()
				}
			}},
		})
	if err != nil {
		panic(err)
	}
	numberTextObserver = objc.ID(numberTextObserverClass).Send(sel("new"))
	center := objc.ID(class("NSNotificationCenter")).Send(sel("defaultCenter"))
	center.Send(sel("addObserver:selector:name:object:"), numberTextObserver, sel("textDidBeginEditing:"),
		nsString("NSTextDidBeginEditingNotification"), objc.ID(0))
	center.Send(sel("addObserver:selector:name:object:"), numberTextObserver, sel("textDidEndEditing:"),
		nsString("NSTextDidEndEditingNotification"), objc.ID(0))
	center.Send(sel("addObserver:selector:name:object:"), numberTextObserver, sel("textViewSelectionDidChange:"),
		nsString("NSTextViewDidChangeSelectionNotification"), objc.ID(0))

	numberStepButtonClass, err = objc.RegisterClass("EcoPasteNumberStepButton", class("NSButton"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("drawRect:"), Fn: drawNumberStepButton},
			{Cmd: sel("acceptsFirstMouse:"), Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) bool { return true }},
			{Cmd: sel("updateTrackingAreas"), Fn: func(self objc.ID, _ objc.SEL) {
				sendSuper[objc.ID](self, numberStepButtonClass, sel("updateTrackingAreas"))
				ensureHoverTracking(self)
			}},
			{Cmd: sel("mouseEntered:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if hover := numberStepHovers[self]; hover != nil {
					hover.setTarget(1)
				}
			}},
			{Cmd: sel("mouseExited:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if hover := numberStepHovers[self]; hover != nil {
					hover.setTarget(0)
				}
			}},
			{Cmd: sel("viewDidChangeEffectiveAppearance"), Fn: func(self objc.ID, _ objc.SEL) {
				sendSuper[objc.ID](self, numberStepButtonClass, sel("viewDidChangeEffectiveAppearance"))
				self.Send(sel("setNeedsDisplay:"), true)
			}},
		})
	if err != nil {
		panic(err)
	}
}

func numberInputForEditor(editor objc.ID) *numberInput {
	for _, n := range numberInputs {
		if objc.Send[objc.ID](n.field, sel("currentEditor")) == editor {
			return n
		}
	}
	return nil
}
