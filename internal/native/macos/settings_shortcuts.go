//go:build darwin

package macos

import (
	"strings"
	"sync"

	"github.com/ebitengine/purego/objc"
)

type shortcutControl struct {
	path    string
	control objc.ID
	get     func(SettingsValues) string
}

type shortcutBinding struct {
	view  *SettingsView
	path  string
	value string
}

var (
	registerSettingsShortcutsOnce sync.Once
	shortcutFieldClass            objc.Class
	settingsShortcutBindings      = map[objc.ID]*shortcutBinding{}
	settingsShortcutRecording     = map[objc.ID]bool{}
)

func (v *SettingsView) buildShortcuts() {
	registerSettingsShortcutsOnce.Do(registerSettingsShortcutTargets)
	content := v.newTab("shortcuts", "快捷键")

	intro := objc.ID(classNSTextField).Send(sel("labelWithString:"),
		nsString("点击输入框后按下组合键；Delete 清空，Esc 取消。"))
	intro.Send(sel("setFont:"), systemFont(11))
	intro.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	intro.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: 438}, Size: CGSize{Width: tabContentW - 48, Height: 18}})
	content.Send(sel("addSubview:"), intro)

	y := 388.0
	v.addShortcutField(content, y, "打开剪贴板", "shortcuts.openClipboard",
		func(s SettingsValues) string { return s.OpenClipboardShortcut })
	v.addShortcutField(content, y-74, "打开设置", "shortcuts.openPreference",
		func(s SettingsValues) string { return s.OpenPreferenceShortcut })

	note := objc.ID(classNSTextField).Send(sel("labelWithString:"),
		nsString("快捷键至少包含 Command、Option 或 Control；F1–F12 可单独使用。"))
	note.Send(sel("setFont:"), systemFont(11))
	note.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	note.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: y - 150}, Size: CGSize{Width: tabContentW - 48, Height: 18}})
	content.Send(sel("addSubview:"), note)
}

func (v *SettingsView) addShortcutField(content objc.ID, y float64, title, path string, get func(SettingsValues) string) {
	v.addLabel(content, y, title)
	field := objc.ID(shortcutFieldClass).Send(sel("new"))
	field.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: tabContentW - 280, Y: y - 6}, Size: CGSize{Width: 260, Height: 30}})
	field.Send(sel("setFont:"), systemFont(12))
	field.Send(sel("setAlignment:"), int64(1))
	field.Send(sel("setEditable:"), false)
	field.Send(sel("setSelectable:"), false)
	field.Send(sel("setBezeled:"), true)
	field.Send(sel("setDrawsBackground:"), true)
	field.Send(sel("setFocusRingType:"), int64(2))
	field.Send(sel("setPlaceholderString:"), nsString("未设置"))
	field.Send(sel("setAccessibilityElement:"), true)
	field.Send(sel("setAccessibilityRole:"), nsString("AXTextField"))
	field.Send(sel("setAccessibilityLabel:"), nsString(title+"快捷键"))
	field.Send(sel("setAccessibilityHelp:"), nsString("按下组合键录制，Delete 清空，Esc 取消"))
	content.Send(sel("addSubview:"), field)
	v.shortcutFields = append(v.shortcutFields, &shortcutControl{path: path, control: field, get: get})
	settingsShortcutBindings[field] = &shortcutBinding{view: v, path: path}
}

func (v *SettingsView) applyShortcuts(values SettingsValues) {
	for _, field := range v.shortcutFields {
		value := field.get(values)
		binding := settingsShortcutBindings[field.control]
		if binding != nil {
			binding.value = value
		}
		if settingsShortcutRecording[field.control] {
			continue
		}
		field.control.Send(sel("setStringValue:"), nsString(value))
	}
}

func (v *SettingsView) beginShortcutRecording(field objc.ID) {
	for other := range settingsShortcutRecording {
		if other != field {
			v.endShortcutRecording(other, true)
		}
	}
	settingsShortcutRecording[field] = true
	field.Send(sel("setStringValue:"), nsString("请按下组合键…"))
	if window := objc.Send[objc.ID](field, sel("window")); window != 0 {
		window.Send(sel("makeFirstResponder:"), field)
	}
	v.cx.Emit("suspendShortcuts", true)
}

func (v *SettingsView) recordShortcut(field, event objc.ID) {
	keyCode := objc.Send[uint16](event, sel("keyCode"))
	flags := uint64(objc.Send[uint](event, sel("modifierFlags")))
	if keyCode == keyEscape {
		v.endShortcutRecording(field, true)
		return
	}
	clear := keyCode == keyDelete || keyCode == keyForwardDelete
	if clear && flags&(commandModifier|controlModifier|alternateModifier|shiftModifier) == 0 {
		v.finishShortcut(field, "")
		return
	}
	name := shortcutKeyName(keyCode)
	if name == "" {
		name = shortcutCharacter(goString(objc.Send[objc.ID](event, sel("charactersIgnoringModifiers"))))
	}
	if name == "" {
		return
	}
	var modifiers []string
	if flags&commandModifier != 0 {
		modifiers = append(modifiers, "Command")
	}
	if flags&controlModifier != 0 {
		modifiers = append(modifiers, "Control")
	}
	if flags&alternateModifier != 0 {
		modifiers = append(modifiers, "Alt")
	}
	if flags&shiftModifier != 0 {
		modifiers = append(modifiers, "Shift")
	}
	hasPrimary := flags&(commandModifier|controlModifier|alternateModifier) != 0
	if !hasPrimary && !recorderFunctionKey(name) {
		field.Send(sel("setStringValue:"), nsString("需包含 ⌘ / ⌥ / ⌃"))
		return
	}
	modifiers = append(modifiers, name)
	v.finishShortcut(field, strings.Join(modifiers, "+"))
}

func (v *SettingsView) finishShortcut(field objc.ID, value string) {
	binding := settingsShortcutBindings[field]
	if binding == nil {
		return
	}
	binding.value = value
	field.Send(sel("setStringValue:"), nsString(value))
	delete(settingsShortcutRecording, field)
	if window := objc.Send[objc.ID](field, sel("window")); window != 0 {
		window.Send(sel("makeFirstResponder:"), objc.ID(0))
	}
	v.cx.Emit("change", SettingsChange{Path: binding.path, Value: value})
	v.cx.Emit("suspendShortcuts", false)
}

func (v *SettingsView) endShortcutRecording(field objc.ID, restore bool) {
	if !settingsShortcutRecording[field] {
		return
	}
	delete(settingsShortcutRecording, field)
	if restore {
		if binding := settingsShortcutBindings[field]; binding != nil {
			field.Send(sel("setStringValue:"), nsString(binding.value))
		}
	}
	if window := objc.Send[objc.ID](field, sel("window")); window != 0 {
		window.Send(sel("makeFirstResponder:"), objc.ID(0))
	}
	v.cx.Emit("suspendShortcuts", false)
}

func registerSettingsShortcutTargets() {
	var err error
	shortcutFieldClass, err = objc.RegisterClass("EcoPasteShortcutField", class("NSTextField"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("acceptsFirstResponder"), Fn: func(_ objc.ID, _ objc.SEL) bool { return true }},
			{Cmd: sel("mouseDown:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if binding := settingsShortcutBindings[self]; binding != nil {
					binding.view.beginShortcutRecording(self)
				}
			}},
			{Cmd: sel("keyDown:"), Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
				if binding := settingsShortcutBindings[self]; binding != nil {
					binding.view.recordShortcut(self, event)
				}
			}},
			{Cmd: sel("performKeyEquivalent:"), Fn: func(self objc.ID, _ objc.SEL, event objc.ID) bool {
				if !settingsShortcutRecording[self] {
					return false
				}
				if binding := settingsShortcutBindings[self]; binding != nil {
					binding.view.recordShortcut(self, event)
				}
				return true
			}},
		})
	if err != nil {
		panic(err)
	}
}

func shortcutCharacter(value string) string {
	if value == "" {
		return ""
	}
	runeValue := []rune(value)[0]
	text := strings.ToUpper(string(runeValue))
	if len([]rune(text)) != 1 {
		return ""
	}
	return text
}

func recorderFunctionKey(name string) bool {
	switch name {
	case "F1", "F2", "F3", "F4", "F5", "F6", "F7", "F8", "F9", "F10", "F11", "F12":
		return true
	}
	return false
}

// shortcutKeyName maps AppKit key codes to the stored shortcut names.
func shortcutKeyName(code uint16) string {
	if name, ok := recorderKeys[code]; ok {
		return name
	}
	return ""
}

var recorderKeys = map[uint16]string{
	0: "A", 1: "S", 2: "D", 3: "F", 4: "H", 5: "G", 6: "Z", 7: "X", 8: "C", 9: "V",
	11: "B", 12: "Q", 13: "W", 14: "E", 15: "R", 16: "Y", 17: "T", 18: "1", 19: "2",
	20: "3", 21: "4", 22: "6", 23: "5", 24: "=", 25: "9", 26: "7", 27: "-", 28: "8",
	29: "0", 30: "]", 31: "O", 32: "U", 33: "[", 34: "I", 35: "P", 36: "Enter",
	37: "L", 38: "J", 39: "'", 40: "K", 41: ";", 42: "\\", 43: ",", 44: "/", 45: "N",
	46: "M", 47: ".", 48: "Tab", 49: "Space", 50: "`", 51: "Backspace", 53: "Esc",
	64: "F17", 79: "F18", 80: "F19", 90: "F20", 96: "F5", 97: "F6", 98: "F7", 99: "F3",
	100: "F8", 101: "F9", 103: "F11", 105: "F13", 106: "F16", 107: "F14", 109: "F10",
	111: "F12", 113: "F15", 115: "Home", 116: "PageUp", 117: "Delete", 118: "F4",
	119: "End", 120: "F2", 121: "PageDown", 122: "F1", 123: "ArrowLeft", 124: "ArrowRight",
	125: "ArrowDown", 126: "ArrowUp",
}
