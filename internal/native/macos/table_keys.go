//go:build darwin

package macos

// historyKeyAction maps a key press to a panel action.
func historyKeyAction(key uint16, flags uint64, row HistoryRow, spacePreview bool) (string, bool) {
	if action, handled := activationKeyAction(key); handled {
		return action, true
	}
	if action, handled := itemCommandKeyAction(key, flags, row); handled {
		return action, true
	}
	if action, handled := editKeyAction(key, flags, spacePreview); handled {
		return action, true
	}
	if action, handled := navigationKeyAction(key, flags); handled {
		return action, true
	}
	return clipboardKeyAction(key, flags, row)
}

func activationKeyAction(key uint16) (string, bool) {
	if key != keyReturn && key != keyEnter {
		return "", false
	}
	return "activate", true
}

func itemCommandKeyAction(key uint16, flags uint64, row HistoryRow) (string, bool) {
	if !commandShortcut(flags) {
		return "", false
	}
	switch key {
	case keyP:
		return "toggleWindowPinned", true
	case keyT:
		return "togglePinned", true
	case keyD:
		return "toggleFavorite", true
	case keyM:
		return "editNote", true
	case keyN:
		return "newGroup", true
	case keyO:
		if !rowOpenable(row) {
			return "", false
		}
		if row.Kind == "files" || row.SubKind == "path" {
			return "reveal", true
		}
		return "open", true
	}
	return "", false
}

func editKeyAction(key uint16, flags uint64, spacePreview bool) (string, bool) {
	unmodified := userModifiers(flags) == 0
	switch {
	case key == keySpace && unmodified && spacePreview:
		return "preview", true
	case key == keyEscape:
		return "hide", true
	case key == keyDelete || key == keyForwardDelete:
		return "delete", true
	}
	return "", false
}

// navigationKeyAction maps the navigation keys while ignoring the Function
// and NumericPad flags AppKit attaches to every arrow and navigation key.
// Treating those synthetic flags as real modifiers made plain Up/Down look
// like page keys and plain Left/Right jump to the ends of the list.
func navigationKeyAction(key uint16, flags uint64) (string, bool) {
	commandOnly := commandShortcut(flags)
	unmodified := userModifiers(flags) == 0
	switch {
	case key == keyHome && unmodified ||
		key == keyUpArrow && commandOnly:
		return "firstRow", true
	case key == keyEnd && unmodified ||
		key == keyDownArrow && commandOnly:
		return "lastRow", true
	case key == keyPageUp && unmodified:
		return "pageUp", true
	case key == keyPageDown && unmodified:
		return "pageDown", true
	case key == keyUpArrow && unmodified:
		return "previousRow", true
	case key == keyDownArrow && unmodified:
		return "nextRow", true
	}
	return "", false
}

func clipboardKeyAction(key uint16, flags uint64, row HistoryRow) (string, bool) {
	command := flags&commandModifier != 0
	option := flags&alternateModifier != 0
	switch {
	case command && option && key == keyC && row.Kind == "text":
		return "copyPlain", true
	case command && key == keyC:
		return "copy", true
	case command && key == keyV:
		return "activate", true
	case command && key == keyF:
		return "focusSearch", true
	}
	return "", false
}

// commandDigitIndex maps Command-1 through Command-0 to an offset from the
// first visible row.
func commandDigitIndex(key uint16) (int, bool) {
	switch key {
	case key1:
		return 0, true
	case key2:
		return 1, true
	case key3:
		return 2, true
	case key4:
		return 3, true
	case key5:
		return 4, true
	case key6:
		return 5, true
	case key7:
		return 6, true
	case key8:
		return 7, true
	case key9:
		return 8, true
	case key0:
		return 9, true
	}
	return 0, false
}

// visibleDigitIndex resolves a Command-digit to an absolute row relative to the
// first visible row, so quick paste follows the scroll position: whichever row
// is at the top of the viewport becomes Command-1.
func visibleDigitIndex(key uint16, firstVisible, rowCount int) (int, bool) {
	offset, ok := commandDigitIndex(key)
	if !ok || firstVisible < 0 {
		return 0, false
	}
	index := firstVisible + offset
	if index >= rowCount {
		return 0, false
	}
	return index, true
}

// userModifiers returns the modifiers a person is actually holding. AppKit
// reports the Function and NumericPad bits for arrow and navigation keys,
// so they must be masked out before a key is matched against a shortcut.
func userModifiers(flags uint64) uint64 {
	return flags & (shiftModifier | controlModifier | alternateModifier | commandModifier)
}

// commandShortcut reports whether Command is the only modifier pressed.
func commandShortcut(flags uint64) bool {
	return userModifiers(flags) == commandModifier
}

// rowOpenable reports whether Command-O has a target: URLs, email
// addresses, local paths, and file lists.
func rowOpenable(row HistoryRow) bool {
	switch row.Kind {
	case "files":
		return true
	case "text":
		return row.SubKind == "url" || row.SubKind == "email" || row.SubKind == "path"
	}
	return false
}
