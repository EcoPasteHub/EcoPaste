//go:build darwin

package macos

import "strings"

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

// shortcutKeyName maps AppKit key codes to the names stored in shortcuts.
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
