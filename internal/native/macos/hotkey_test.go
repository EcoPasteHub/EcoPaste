//go:build darwin

package macos

import "testing"

func TestParseHotKey(t *testing.T) {
	cases := []struct {
		value     string
		code      uint32
		modifiers uint32
		ok        bool
	}{
		{value: "Alt+C", code: 8, modifiers: carbonOption, ok: true},
		{value: "Command+Shift+K", code: 40, modifiers: carbonCmd | carbonShift, ok: true},
		{value: "Ctrl+Alt+Delete", code: 117, modifiers: carbonControl | carbonOption, ok: true},
		{value: "F5", code: 96, ok: true},
		{value: "Alt+ArrowUp", code: 126, modifiers: carbonOption, ok: true},
		{value: "Control+Esc", code: 53, modifiers: carbonControl, ok: true},
		{value: "Alt+Nope"},
	}
	for _, tc := range cases {
		code, modifiers, ok := parseHotKey(tc.value)
		if code != tc.code || modifiers != tc.modifiers || ok != tc.ok {
			t.Errorf("parseHotKey(%q) = %d, %d, %t", tc.value, code, modifiers, ok)
		}
	}
}
