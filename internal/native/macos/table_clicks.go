//go:build darwin

package macos

import (
	"github.com/ebitengine/purego/objc"
)

func historyTableMethods() []objc.MethodDef {
	return []objc.MethodDef{
		{Cmd: sel("copy:"), Fn: historyTableCopy},
		{Cmd: sel("paste:"), Fn: historyTablePaste},
		{Cmd: sel("rightMouseDown:"), Fn: historyTableRightMouseDown},
		{Cmd: sel("mouseDown:"), Fn: historyTableMouseDown},
		{Cmd: sel("activatePendingClick:"), Fn: historyTableActivatePendingClick},
		{Cmd: sel("keyDown:"), Fn: historyTableKeyDown},
		{Cmd: sel("menuNeedsUpdate:"), Fn: historyTableMenuNeedsUpdate},
		{Cmd: sel("drawContextMenuHighlightForRow:"), Fn: historyTableSuppressContextMenuHighlight},
	}
}

// historyTableSuppressContextMenuHighlight stops AppKit from outlining the
// right-clicked row while its menu is open. The list already shows the
// selected card, and the system outline spans the full row width, reading as
// a stray white frame from the window edge inward.
func historyTableSuppressContextMenuHighlight(objc.ID, objc.SEL, int64) {}

func historyTableCopy(self objc.ID, _ objc.SEL, _ objc.ID) {
	if p := panelsByTable[self]; p != nil && p.selection != "" {
		p.cx.Emit("copy", p.selection)
	}
}

func historyTablePaste(self objc.ID, _ objc.SEL, _ objc.ID) {
	if p := panelsByTable[self]; p != nil && p.selection != "" {
		p.cx.Emit("activate", p.selection)
	}
}

func historyTableRightMouseDown(self objc.ID, _ objc.SEL, event objc.ID) {
	if p := panelsByTable[self]; p != nil {
		row := historyRowAtEvent(self, event)
		if row >= 0 && row < len(p.rows) {
			p.selectRow(row)
		}
	}
	sendSuper[objc.ID](self, historyTableClass, sel("rightMouseDown:"), event)
}

func historyTableMouseDown(self objc.ID, _ objc.SEL, event objc.ID) {
	left := objc.Send[int64](event, sel("buttonNumber")) == 0
	var row, clicks int
	if left {
		row = historyRowAtEvent(self, event)
		clicks = int(objc.Send[int64](event, sel("clickCount")))
	}
	p := panelsByTable[self]
	if left && row < 0 && p != nil && panelDragWindow(p, p.view.Send(sel("window")), nil) != 0 {
		return
	}
	sendSuper[objc.ID](self, historyTableClass, sel("mouseDown:"), event)
	if p == nil || !left {
		return
	}
	switch {
	case clicks == 2 && row >= 0 && row < len(p.rows):
		cancelPendingClick(self)
		p.selectRow(row)
		p.cx.Emit("activateDoubleClick", p.rows[row].ID)
	case clicks == 1 && row >= 0 && row < len(p.rows):
		cancelPendingClick(self)
		p.pendingClick = p.rows[row].ID
		p.selectRow(row)
		self.Send(sel("performSelector:withObject:afterDelay:"),
			sel("activatePendingClick:"), objc.ID(0), doubleClickDelay())
	}
}

func historyTableActivatePendingClick(self objc.ID, _ objc.SEL, _ objc.ID) {
	p := panelsByTable[self]
	if p == nil || p.pendingClick == "" {
		return
	}
	id := p.pendingClick
	p.pendingClick = ""
	p.cx.Emit("activateSingleClick", id)
}

func historyTableKeyDown(self objc.ID, _ objc.SEL, event objc.ID) {
	p := panelsByTable[self]
	if p == nil {
		sendSuper[objc.ID](self, historyTableClass, sel("keyDown:"), event)
		return
	}
	key := objc.Send[uint16](event, sel("keyCode"))
	flags := uint64(objc.Send[uint](event, sel("modifierFlags")))
	// Match the legacy active item: fall back to the first row when the
	// user has not selected anything yet, so Return/Copy act on the top item.
	index := p.selectedIndex()
	if index < 0 && len(p.rows) > 0 {
		index = 0
	}
	row := HistoryRow{}
	if index >= 0 && index < len(p.rows) {
		row = p.rows[index]
	}
	action, handled := historyKeyAction(key, flags, row, p.behavior.SpacePreview)
	target := p.selection
	if target == "" {
		target = row.ID
	}
	switch {
	case !handled:
		sendSuper[objc.ID](self, historyTableClass, sel("keyDown:"), event)
	case action == "focusSearch":
		p.focusSearch()
	case action == "firstRow", action == "lastRow", action == "pageUp",
		action == "pageDown", action == "previousRow", action == "nextRow":
		p.applyNavigation(action)
	case action == "delete":
		// Deletion stays tied to an explicit selection so an empty list
		// focus never removes the first item by accident.
		if p.selection != "" && row.CanDelete && (!row.DeleteConfirm || confirmDelete()) {
			p.cx.Emit("delete", p.selection)
		}
	default:
		p.cx.Emit(action, target)
	}
}

func historyTableMenuNeedsUpdate(self objc.ID, _ objc.SEL, menu objc.ID) {
	p := panelsByTable[self]
	if p == nil {
		return
	}
	row := int(objc.Send[int64](self, sel("selectedRow")))
	if row < 0 || row >= len(p.rows) {
		menu.Send(sel("removeAllItems"))
		return
	}
	p.populateMenu(menu, p.rows[row])
}
