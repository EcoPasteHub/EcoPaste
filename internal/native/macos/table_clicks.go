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
		{Cmd: sel("otherMouseDown:"), Fn: historyTableOtherMouseDown},
		{Cmd: sel("otherMouseUp:"), Fn: historyTableOtherMouseUp},
		{Cmd: sel("mouseDown:"), Fn: historyTableMouseDown},
		{Cmd: sel("activatePendingClick:"), Fn: historyTableActivatePendingClick},
		{Cmd: sel("keyDown:"), Fn: historyTableKeyDown},
		{Cmd: sel("menuNeedsUpdate:"), Fn: historyTableMenuNeedsUpdate},
	}
}

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

func historyTableOtherMouseDown(self objc.ID, _ objc.SEL, event objc.ID) {
	p := panelsByTable[self]
	if p == nil || objc.Send[int64](event, sel("buttonNumber")) != 2 {
		return
	}
	row := historyRowAtEvent(self, event)
	if row < 0 || row >= len(p.rows) {
		return
	}
	p.selectRow(row)
	p.cx.Emit("middleClick", p.rows[row].ID)
}

func historyTableOtherMouseUp(objc.ID, objc.SEL, objc.ID) {}

func historyTableMouseDown(self objc.ID, _ objc.SEL, event objc.ID) {
	left := objc.Send[int64](event, sel("buttonNumber")) == 0
	var row, clicks int
	if left {
		row = historyRowAtEvent(self, event)
		clicks = int(objc.Send[int64](event, sel("clickCount")))
	}
	p := panelsByTable[self]
	if left && row < 0 && p != nil && panelDragWindow(p, p.view.Send(sel("window")), func() {
		self.Send(sel("deselectAll:"), objc.ID(0))
	}) != 0 {
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
	row := HistoryRow{}
	for _, candidate := range p.rows {
		if candidate.ID == p.selection {
			row = candidate
			break
		}
	}
	action, handled := historyKeyAction(key, flags, row, p.behavior.SpacePreview)
	switch {
	case !handled:
		sendSuper[objc.ID](self, historyTableClass, sel("keyDown:"), event)
	case action == "focusSearch":
		p.focusSearch()
	case action == "firstRow":
		p.selectRow(0)
	case action == "lastRow":
		p.selectRow(len(p.rows) - 1)
	case action == "pageUp" || action == "pageDown":
		current := p.selectedIndex()
		if current < 0 {
			current = 0
		}
		delta := p.pageSize()
		if action == "pageUp" {
			delta = -delta
		}
		p.selectRow(current + delta)
	case action == "delete":
		if row.CanDelete && (!row.DeleteConfirm || confirmDelete()) {
			p.cx.Emit("delete", p.selection)
		}
	default:
		p.cx.Emit(action, p.selection)
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
