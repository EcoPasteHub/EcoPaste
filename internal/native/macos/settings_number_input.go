//go:build darwin

package macos

import (
	"math"
	"strconv"
	"strings"
	"sync"
	"unicode/utf16"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

const (
	numberInputWidth       = shortcutRecorderWidth
	numberInputHeight      = settingsControlHeight
	numberInputButtonSize  = 18.0
	numberInputButtonInset = 3.0
	numberInputTextInset   = numberInputButtonInset + numberInputButtonSize + 3
	numberInputTextHeight  = 18.0
	numberInputTextSize    = 13.0

	numberInputCenterAlignment = 1
)

type numberInput struct {
	owner    *SettingsView
	path     string
	tab      string
	title    i18n.Key
	control  objc.ID
	field    objc.ID
	decrease objc.ID
	increase objc.ID

	minValue  int64
	maxValue  int64
	stepValue int64
	unit      any
	get       func(SettingsValues) uint32
	value     uint32
	editing   bool
	monitor   objc.ID

	chrome *inputChrome
}

var (
	registerNumberInputOnce      sync.Once
	numberInputClass             objc.Class
	numberTextFieldClass         objc.Class
	numberTextFieldDelegateClass objc.Class
	numberTextObserverClass      objc.Class
	numberStepButtonClass        objc.Class
	numberTextFieldDelegate      objc.ID
	numberTextObserver           objc.ID

	numberInputs     = map[objc.ID]*numberInput{}
	numberTextFields = map[objc.ID]*numberInput{}
	numberEditors    = map[objc.ID]*numberInput{}
	numberStepHovers = map[objc.ID]*transition{}
)

func (v *SettingsView) addNumberInput(content objc.ID, y float64, title i18n.Key, path string, minValue, maxValue, step float64, unit any, get func(SettingsValues) uint32) {
	registerNumberInputOnce.Do(registerNumberInputTargets)
	v.addLabelText(content, y, settingsContentWidth-2*settingsRowInset-numberInputWidth-24,
		func() string { return numberInputTitle(title, unit) })

	control := objc.ID(numberInputClass).Send(sel("alloc")).Send(sel("initWithFrame:"), CGRect{
		Origin: CGPoint{
			X: settingsContentX + settingsContentWidth - settingsRowInset - numberInputWidth,
			Y: y + (settingsRowHeight-numberInputHeight)/2,
		},
		Size: CGSize{Width: numberInputWidth, Height: numberInputHeight},
	})
	control.Send(sel("setAccessibilityElement:"), true)
	control.Send(sel("setAccessibilityRole:"), nsString("AXGroup"))

	n := &numberInput{
		owner:     v,
		path:      path,
		tab:       v.currentTab,
		title:     title,
		control:   control,
		minValue:  int64(math.Round(minValue)),
		maxValue:  int64(math.Round(maxValue)),
		stepValue: max(int64(math.Round(step)), 1),
		unit:      unit,
		get:       get,
	}

	n.decrease = newNumberStepButton("minus", control, "stepDown:", i18n.Tf(i18n.SettingsNumberDecrease, numberInputTitle(title, unit)))
	control.Send(sel("addSubview:"), n.decrease)
	n.decrease.Send(sel("release"))
	n.increase = newNumberStepButton("plus", control, "stepUp:", i18n.Tf(i18n.SettingsNumberIncrease, numberInputTitle(title, unit)))
	control.Send(sel("addSubview:"), n.increase)
	n.increase.Send(sel("release"))

	n.field = objc.ID(numberTextFieldClass).Send(sel("alloc")).Send(sel("initWithFrame:"), CGRect{})
	n.field.Send(sel("setEditable:"), true)
	n.field.Send(sel("setSelectable:"), true)
	n.field.Send(sel("setBezeled:"), false)
	n.field.Send(sel("setBordered:"), false)
	n.field.Send(sel("setDrawsBackground:"), false)
	n.field.Send(sel("setFocusRingType:"), int64(1))
	n.field.Send(sel("setAlignment:"), int64(numberInputCenterAlignment))
	n.field.Send(sel("setFont:"), numberInputValueFont())
	n.field.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("labelColor")))
	n.field.Send(sel("setUsesSingleLineMode:"), true)
	n.field.Send(sel("setLineBreakMode:"), int64(4))
	n.field.Send(sel("setAccessibilityHelp:"), nsString(i18n.T(i18n.SettingsNumberHelp)))
	n.field.Send(sel("setDelegate:"), numberTextFieldDelegate)
	control.Send(sel("addSubview:"), n.field)
	n.field.Send(sel("release"))
	n.chrome = newInputChrome(control, n.field)

	content.Send(sel("addSubview:"), control)
	control.Send(sel("release"))
	numberInputs[control] = n
	numberTextFields[n.field] = n
	v.numberInputs = append(v.numberInputs, n)
	n.refresh()
}

func newNumberStepButton(symbol string, target objc.ID, action, accessibility string) objc.ID {
	button := objc.ID(numberStepButtonClass).Send(sel("alloc")).Send(sel("initWithFrame:"), CGRect{})
	image := systemSymbol(symbol)
	image.Send(sel("setSize:"), CGSize{Width: 12, Height: 12})
	button.Send(sel("setImage:"), image)
	button.Send(sel("setImagePosition:"), int64(1))
	button.Send(sel("setImageScaling:"), int64(2))
	button.Send(sel("setBordered:"), false)
	button.Send(sel("setFocusRingType:"), int64(1))
	button.Send(sel("setRefusesFirstResponder:"), true)
	button.Send(sel("setContentTintColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	button.Send(sel("setTarget:"), target)
	button.Send(sel("setAction:"), sel(action))
	button.Send(sel("setToolTip:"), nsString(accessibility))
	button.Send(sel("setAccessibilityLabel:"), nsString(accessibility))
	numberStepHovers[button] = newViewTransition(button)
	return button
}

func (v *SettingsView) applyNumberInputs(values SettingsValues, languageChanged bool) {
	for _, n := range v.numberInputs {
		if n.editing {
			continue
		}
		value := n.get(values)
		if !languageChanged && n.value == value {
			continue
		}
		n.value = value
		n.refresh()
	}
}

func (v *SettingsView) closeNumberInputs() {
	for _, n := range v.numberInputs {
		n.removeBlurMonitor()
		if n.chrome != nil {
			n.chrome.close()
		}
		delete(numberInputs, n.control)
		delete(numberTextFields, n.field)
		for editor, owner := range numberEditors {
			if owner == n {
				delete(numberEditors, editor)
			}
		}
		discardNumberStepButton(n.decrease)
		discardNumberStepButton(n.increase)
	}
	v.numberInputs = nil
}

func discardNumberStepButton(button objc.ID) {
	if hover := numberStepHovers[button]; hover != nil {
		hover.stop()
		delete(numberStepHovers, button)
	}
}

func numberInputTitle(title i18n.Key, unit any) string {
	text := strings.TrimSpace(settingsTextValue(unit))
	if text == "" {
		return i18n.T(title)
	}
	return i18n.Tf(i18n.SettingsNumberTitleWithUnit, i18n.T(title), text)
}

func (n *numberInput) displayText() string {
	if n.value == 0 && n.minValue == 0 && !n.editing {
		return i18n.T(i18n.SettingsUnlimited)
	}
	if n.editing {
		return n.fieldText()
	}
	return strconv.FormatUint(uint64(n.value), 10)
}

func (n *numberInput) fieldText() string {
	return goString(objc.Send[objc.ID](n.field, sel("stringValue")))
}

func (n *numberInput) refresh() {
	if n.control == 0 {
		return
	}
	n.field.Send(sel("setStringValue:"), nsString(n.displayText()))
	n.refreshAccessibility()
	n.layout()
	n.refreshButtons()
	n.control.Send(sel("setNeedsDisplay:"), true)
}

func (n *numberInput) refreshAccessibility() {
	title := numberInputTitle(n.title, n.unit)
	label := i18n.Tf(i18n.SettingsNumberFieldA11y, title)
	help := i18n.T(i18n.SettingsNumberHelp)
	n.field.Send(sel("setAccessibilityLabel:"), nsString(label))
	n.field.Send(sel("setAccessibilityHelp:"), nsString(help))
	n.field.Send(sel("setToolTip:"), nsString(help))
	n.decrease.Send(sel("setAccessibilityLabel:"), nsString(i18n.Tf(i18n.SettingsNumberDecrease, title)))
	n.increase.Send(sel("setAccessibilityLabel:"), nsString(i18n.Tf(i18n.SettingsNumberIncrease, title)))
	n.field.Send(sel("setAccessibilityValue:"), nsString(n.displayText()))
}

func (n *numberInput) layout() {
	rect := bounds(n.control)
	buttonY := (rect.Size.Height - numberInputButtonSize) / 2
	n.decrease.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: numberInputButtonInset, Y: buttonY},
		Size:   CGSize{Width: numberInputButtonSize, Height: numberInputButtonSize},
	})
	n.increase.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: rect.Size.Width - numberInputButtonInset - numberInputButtonSize, Y: buttonY},
		Size:   CGSize{Width: numberInputButtonSize, Height: numberInputButtonSize},
	})

	contentX := numberInputTextInset
	contentWidth := rect.Size.Width - 2*numberInputTextInset
	y := (rect.Size.Height - numberInputTextHeight) / 2
	n.field.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: contentX, Y: y},
		Size:   CGSize{Width: contentWidth, Height: numberInputTextHeight},
	})
}

func numberInputValueFont() objc.ID {
	return objc.ID(classNSFont).Send(sel("systemFontOfSize:weight:"), numberInputTextSize, 0.23)
}

func (n *numberInput) refreshButtons() {
	value := n.currentValue()
	n.decrease.Send(sel("setEnabled:"), int64(value) > n.minValue)
	n.increase.Send(sel("setEnabled:"), int64(value) < n.maxValue)
}

func (n *numberInput) currentValue() uint32 {
	if n.editing {
		if value, ok := n.parseField(); ok {
			return value
		}
	}
	return n.value
}

func (n *numberInput) parseField() (uint32, bool) {
	text := n.fieldText()
	if text == "" {
		return uint32(max(n.minValue, 0)), false
	}
	value, err := strconv.ParseUint(text, 10, 32)
	if err != nil {
		return uint32(max(n.minValue, 0)), false
	}
	return uint32(max(min(int64(value), n.maxValue), n.minValue)), true
}

func (n *numberInput) beginEditing() {
	if n.editing {
		return
	}
	n.editing = true
	n.installBlurMonitor()
	if n.value == 0 && n.minValue == 0 {
		n.field.Send(sel("setStringValue:"), nsString("0"))
	}
	n.refreshAccessibility()
	n.layout()
	n.refreshButtons()
	n.setFocusTarget(1)
}

func (n *numberInput) endEditing() {
	if !n.editing {
		return
	}
	n.editing = false
	n.removeBlurMonitor()
	value, _ := n.parseField()
	changed := value != n.value
	n.value = value
	n.refresh()
	n.setFocusTarget(0)
	if changed {
		n.emit()
	}
}

// installBlurMonitor ends editing when the click lands outside the input,
// matching the native text field behavior the panel normally provides.
func (n *numberInput) installBlurMonitor() {
	if n.monitor != 0 {
		return
	}
	block := objc.NewBlock(func(_ objc.Block, event objc.ID) objc.ID {
		if n.editing && !n.pointInside(event) {
			if window := objc.Send[objc.ID](n.control, sel("window")); window != 0 {
				window.Send(sel("makeFirstResponder:"), objc.ID(0))
			} else {
				n.endEditing()
			}
		}
		return event
	})
	n.monitor = objc.ID(class("NSEvent")).Send(
		sel("addLocalMonitorForEventsMatchingMask:handler:"),
		uint64(1<<eventLeftMouseDown), objc.ID(block))
	block.Release()
}

func (n *numberInput) pointInside(event objc.ID) bool {
	window := objc.Send[objc.ID](n.control, sel("window"))
	if window == 0 || objc.Send[objc.ID](event, sel("window")) != window {
		return false
	}
	point := objc.Send[CGPoint](event, sel("locationInWindow"))
	local := objc.Send[CGPoint](n.control, sel("convertPoint:fromView:"), point, objc.ID(0))
	rect := bounds(n.control)
	return local.X >= 0 && local.Y >= 0 && local.X <= rect.Size.Width && local.Y <= rect.Size.Height
}

func (n *numberInput) removeBlurMonitor() {
	if n.monitor == 0 {
		return
	}
	objc.ID(class("NSEvent")).Send(sel("removeMonitor:"), n.monitor)
	n.monitor = 0
}

func (n *numberInput) sanitizeText() {
	text := n.fieldText()
	editor := objc.Send[objc.ID](n.field, sel("currentEditor"))
	selection := NSRange{}
	if editor != 0 {
		selection = objc.Send[NSRange](editor, sel("selectedRange"))
		text = goString(objc.Send[objc.ID](editor, sel("string")))
	}
	clean, caret := numberInputDigits(text, selection.Location)
	if clean != text {
		nsClean := nsString(clean)
		if editor != 0 {
			editor.Send(sel("setString:"), nsClean)
			editor.Send(sel("setSelectedRange:"), NSRange{Location: uint64(caret)})
		} else {
			n.field.Send(sel("setStringValue:"), nsClean)
		}
	}
	n.refreshAccessibility()
	n.layout()
	n.refreshButtons()
}

func numberInputDigits(text string, location uint64) (string, int) {
	units := utf16.Encode([]rune(text))
	digits := make([]uint16, 0, len(units))
	caret := 0
	for index, unit := range units {
		if unit < '0' || unit > '9' {
			continue
		}
		if uint64(index) < location {
			caret++
		}
		digits = append(digits, unit)
	}
	return string(utf16.Decode(digits)), caret
}

func (n *numberInput) step(delta int64) {
	base := int64(n.currentValue())
	next := max(min(base+delta, n.maxValue), n.minValue)
	if next < 0 || next > math.MaxUint32 {
		return
	}
	changed := uint32(next) != n.value
	n.value = uint32(next)
	if n.editing {
		n.field.Send(sel("setStringValue:"), nsString(strconv.FormatInt(next, 10)))
		n.refreshAccessibility()
		n.layout()
		n.refreshButtons()
	} else {
		n.refresh()
	}
	if changed {
		n.emit()
	}
}

func (n *numberInput) emit() {
	n.owner.cx.Emit("change", SettingsChange{Path: n.path, Value: n.value})
}

func (n *numberInput) setFocusTarget(target float64) {
	if n.chrome != nil {
		n.chrome.syncFocus(target > 0)
	}
}

func (n *numberInput) setHoverTarget(target float64) {
	if n.chrome != nil {
		n.chrome.setHoverTarget(target)
	}
}

func drawNumberInput(self objc.ID, _ objc.SEL, _ CGRect) {
	drawInputChrome(self)
}

func drawNumberStepButton(self objc.ID, _ objc.SEL, _ CGRect) {
	enabled := objc.Send[bool](self, sel("isEnabled"))
	pressed := objc.Send[bool](self, sel("isHighlighted"))
	hover := 0.0
	if state := numberStepHovers[self]; state != nil {
		hover = state.value
	}
	if enabled && (hover > 0.001 || pressed) {
		rect := bounds(self)
		radius := rect.Size.Width / 2
		fill := objc.ID(classNSColor).Send(sel("quaternaryLabelColor"))
		if pressed {
			fill = objc.ID(classNSColor).Send(sel("tertiaryLabelColor"))
		}
		fraction := hover
		if pressed {
			fraction = 1
		}
		fillRoundedRect(rect, radius, fadeColor(fill, fraction))
	}
	sendSuper[objc.ID](self, numberStepButtonClass, sel("drawRect:"))
}
