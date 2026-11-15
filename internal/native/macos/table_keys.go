//go:build darwin

package macos

// historyKeyAction maps a key press to a panel action. Categories are
// checked in the order the legacy list handled them: activation, item
// commands, single-key edit actions, navigation, then clipboard.
func historyKeyAction(key uint16, flags uint64, row HistoryRow, spacePreview bool) (string, bool) {
	if action, handled := activationKeyAction(key, flags, row); handled {
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

func activationKeyAction(key uint16, flags uint64, row HistoryRow) (string, bool) {
	if key != keyReturn && key != keyEnter {
		return "", false
	}
	if commandShortcut(flags) && row.Kind == "text" {
		return "pastePlain", true
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
	unmodified := flags&(shiftModifier|controlModifier|alternateModifier|commandModifier) == 0
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

func navigationKeyAction(key uint16, flags uint64) (string, bool) {
	commandOnly := commandShortcut(flags)
	function := flags&functionModifier != 0
	unmodified := flags&(shiftModifier|controlModifier|alternateModifier|commandModifier) == 0
	switch {
	case key == keyHome && unmodified ||
		key == keyLeftArrow && function && unmodified ||
		key == keyUpArrow && commandOnly:
		return "firstRow", true
	case key == keyEnd && unmodified ||
		key == keyRightArrow && function && unmodified ||
		key == keyDownArrow && commandOnly:
		return "lastRow", true
	case (key == keyPageUp || key == keyUpArrow && function) && unmodified:
		return "pageUp", true
	case (key == keyPageDown || key == keyDownArrow && function) && unmodified:
		return "pageDown", true
	}
	return "", false
}

func clipboardKeyAction(key uint16, flags uint64, row HistoryRow) (string, bool) {
	command := flags&commandModifier != 0
	option := flags&alternateModifier != 0
	switch {
	case command && option && key == keyC && row.Kind == "text":
		return "copyPlain", true
	case command && option && key == keyV && row.Kind == "text":
		return "pastePlain", true
	case command && option && key == keyV && row.Kind == "files":
		return "pastePath", true
	case command && key == keyC:
		return "copy", true
	case command && key == keyV:
		return "activate", true
	case command && key == keyF:
		return "focusSearch", true
	}
	return "", false
}

// commandDigitIndex maps Command-1 through Command-0 to the first ten
// visible rows.
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

// commandShortcut reports whether Command is the only modifier pressed.
func commandShortcut(flags uint64) bool {
	return flags&(shiftModifier|controlModifier|alternateModifier|functionModifier|commandModifier) == commandModifier
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
