//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

// searchModeKeyAction maps a key press to a list action while the search
// field has focus. It deliberately claims only keys that have no meaning for
// editing the current query: Up/Down move the selection and Return activates
// it. Delete removes the selected row only when the query is empty, so it can
// never steal a backspace the user meant for the search text. Left/Right,
// Home/End, typing and delete-with-text all stay with the field editor.
func searchModeKeyAction(key uint16, flags uint64, searchEmpty bool) (string, bool) {
	unmodified := userModifiers(flags) == 0
	switch {
	case key == keyUpArrow && unmodified:
		return "previousRow", true
	case key == keyDownArrow && unmodified:
		return "nextRow", true
	case (key == keyReturn || key == keyEnter) && unmodified:
		return "activate", true
	case searchEmpty && (key == keyDelete || key == keyForwardDelete):
		return "delete", true
	}
	return "", false
}

// handleSearchKey keeps the list keyboard contract while the search field
// has focus. The panel's key monitor runs before AppKit dispatches the event,
// so claiming a key here stops the field editor from also acting on it.
func (p *Panel) handleSearchKey(key uint16, flags uint64, repeat bool) bool {
	if p.previewing || !p.searchFieldFocused() || p.searchHasMarkedText() {
		return false
	}
	empty := p.searchText() == ""
	// Space only previews from an empty search so it can never be swallowed
	// while the user is typing a query that contains spaces.
	if key == keySpace && userModifiers(flags) == 0 {
		if !p.behavior.SpacePreview || !empty {
			return false
		}
		if !repeat && p.selection != "" {
			p.cx.Emit("preview", p.selection)
		}
		return true
	}
	action, handled := searchModeKeyAction(key, flags, empty)
	if !handled {
		return false
	}
	switch action {
	case "previousRow", "nextRow":
		// Repeats are welcome here: holding an arrow keeps moving the
		// selection and scrolling the list, matching native list behavior.
		p.applyNavigation(action)
	case "activate":
		if !repeat {
			p.activateSelection()
		}
	case "delete":
		if !repeat {
			p.deleteCurrentRow()
		}
	}
	return true
}

// searchFieldFocused reports whether this panel window is key and its search
// field owns the current editor. When a text field is focused AppKit keeps the
// field editor (an NSTextView) as first responder, so checking the editor
// rather than the control is required.
func (p *Panel) searchFieldFocused() bool {
	if p.search == 0 {
		return false
	}
	window := p.view.Send(sel("window"))
	if window == 0 || !objc.Send[bool](window, sel("isKeyWindow")) {
		return false
	}
	responder := objc.Send[objc.ID](window, sel("firstResponder"))
	if responder == p.search {
		return true
	}
	editor := objc.Send[objc.ID](p.search, sel("currentEditor"))
	return editor != 0 && responder == editor
}

// searchHasMarkedText reports whether an input method is composing text in
// the search field. Claiming keys during composition would break candidate
// selection and IME confirmation, so the list handlers yield to the editor.
func (p *Panel) searchHasMarkedText() bool {
	editor := objc.Send[objc.ID](p.search, sel("currentEditor"))
	if editor == 0 {
		return false
	}
	return objc.Send[bool](editor, sel("hasMarkedText"))
}

func (p *Panel) searchText() string {
	if p.search == 0 {
		return ""
	}
	return goString(p.search.Send(sel("stringValue")))
}

// activateSelection runs the selected row's default action.
func (p *Panel) activateSelection() {
	if p.selection != "" {
		p.cx.Emit("activate", p.selection)
	}
}

// deleteCurrentRow removes the selected row, honoring the row's own delete
// permission and confirmation. It shares the delete path used by the table
// and preview keyboard handlers.
func (p *Panel) deleteCurrentRow() {
	index := p.selectedIndex()
	if index < 0 || index >= len(p.rows) {
		return
	}
	p.deleteRow(p.rows[index])
}
