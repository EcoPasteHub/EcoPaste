//go:build darwin

package macos

import "testing"

func TestHistoryKeyAction(t *testing.T) {
	text := HistoryRow{Kind: "text"}
	files := HistoryRow{Kind: "files"}
	image := HistoryRow{Kind: "image"}
	cases := []struct {
		name    string
		key     uint16
		flags   uint64
		row     HistoryRow
		action  string
		handled bool
	}{
		{name: "return activates", key: keyReturn, row: text, action: "activate", handled: true},
		{name: "enter activates", key: keyEnter, row: files, action: "activate", handled: true},
		{name: "plain P pins", key: keyP, row: text, action: "togglePinned", handled: true},
		{name: "command P falls through", key: keyP, flags: commandModifier, row: text, handled: false},
		{name: "plain space previews", key: keySpace, row: image, action: "preview", handled: true},
		{name: "option space falls through", key: keySpace, flags: alternateModifier, row: image, handled: false},
		{name: "escape hides", key: keyEscape, row: text, action: "hide", handled: true},
		{name: "delete removes", key: keyDelete, row: text, action: "delete", handled: true},
		{name: "home selects first", key: keyHome, row: text, action: "firstRow", handled: true},
		{name: "end selects last", key: keyEnd, row: text, action: "lastRow", handled: true},
		{name: "page up moves up", key: keyPageUp, row: text, action: "pageUp", handled: true},
		{name: "page down moves down", key: keyPageDown, row: text, action: "pageDown", handled: true},
		{name: "fn up pages up", key: keyUpArrow, flags: functionModifier, row: text, action: "pageUp", handled: true},
		{name: "fn down pages down", key: keyDownArrow, flags: functionModifier, row: text, action: "pageDown", handled: true},
		{name: "plain up falls through", key: keyUpArrow, row: text, handled: false},
		{name: "command home falls through", key: keyHome, flags: commandModifier, row: text, handled: false},
		{name: "command C copies", key: keyC, flags: commandModifier, row: text, action: "copy", handled: true},
		{name: "option command C copies plain text", key: keyC, flags: commandModifier | alternateModifier, row: text, action: "copyPlain", handled: true},
		{name: "option command C falls back for images", key: keyC, flags: commandModifier | alternateModifier, row: image, action: "copy", handled: true},
		{name: "command V activates", key: keyV, flags: commandModifier, row: files, action: "activate", handled: true},
		{name: "option command V pastes plain text", key: keyV, flags: commandModifier | alternateModifier, row: text, action: "pastePlain", handled: true},
		{name: "option command V pastes file paths", key: keyV, flags: commandModifier | alternateModifier, row: files, action: "pastePath", handled: true},
		{name: "option command V activates images", key: keyV, flags: commandModifier | alternateModifier, row: image, action: "activate", handled: true},
		{name: "command F focuses search", key: keyF, flags: commandModifier, row: text, action: "focusSearch", handled: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			action, handled := historyKeyAction(tc.key, tc.flags, tc.row)
			if action != tc.action || handled != tc.handled {
				t.Errorf("historyKeyAction() = %q, %v; want %q, %v", action, handled, tc.action, tc.handled)
			}
		})
	}
}
