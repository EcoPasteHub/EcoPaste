//go:build darwin

package macos

import "testing"

func TestSearchModeKeyAction(t *testing.T) {
	nav := functionModifier | numericPadModifier
	cases := []struct {
		name        string
		key         uint16
		flags       uint64
		searchEmpty bool
		action      string
		handled     bool
	}{
		{name: "plain up moves the selection", key: keyUpArrow, flags: nav, action: "previousRow", handled: true},
		{name: "plain down moves the selection", key: keyDownArrow, flags: nav, action: "nextRow", handled: true},
		{name: "shift up stays in the field", key: keyUpArrow, flags: shiftModifier | nav, handled: false},
		{name: "command up stays in the field", key: keyUpArrow, flags: commandModifier | nav, handled: false},
		{name: "left edits the query", key: keyLeftArrow, flags: nav, handled: false},
		{name: "right edits the query", key: keyRightArrow, flags: nav, handled: false},
		{name: "return activates", key: keyReturn, action: "activate", handled: true},
		{name: "enter activates", key: keyEnter, action: "activate", handled: true},
		{name: "command return falls through", key: keyReturn, flags: commandModifier, handled: false},
		{name: "backspace deletes when empty", key: keyDelete, searchEmpty: true, action: "delete", handled: true},
		{name: "forward delete deletes when empty", key: keyForwardDelete, searchEmpty: true, action: "delete", handled: true},
		{name: "backspace edits while searching", key: keyDelete, searchEmpty: false, handled: false},
		{name: "plain space is not claimed here", key: keySpace, handled: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			action, handled := searchModeKeyAction(tc.key, tc.flags, tc.searchEmpty)
			if action != tc.action || handled != tc.handled {
				t.Errorf("searchModeKeyAction() = %q, %v; want %q, %v", action, handled, tc.action, tc.handled)
			}
		})
	}
}
