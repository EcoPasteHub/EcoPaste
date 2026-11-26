//go:build darwin

package macos

import (
	"github.com/ebitengine/purego/objc"
)

func historyTargetMethods() []objc.MethodDef {
	return []objc.MethodDef{
		{Cmd: sel("numberOfRowsInTableView:"), Fn: historyTargetNumberOfRows},
		{Cmd: sel("tableView:viewForTableColumn:row:"), Fn: historyTargetViewForRow},
		{Cmd: sel("tableView:rowViewForRow:"), Fn: historyTargetRowView},
		{Cmd: sel("tableView:heightOfRow:"), Fn: historyTargetHeightOfRow},
		{Cmd: sel("tableViewSelectionDidChange:"), Fn: historyTargetSelectionDidChange},
		{Cmd: sel("tableView:pasteboardWriterForRow:"), Fn: historyTargetPasteboardWriter},
		{Cmd: sel("tableView:draggingSession:willBeginAtPoint:forRowIndexes:"), Fn: historyTargetDragWillBegin},
		{Cmd: sel("checkDragFinished:"), Fn: historyTargetCheckDragFinished},
		{Cmd: sel("tableView:draggingSession:endedAtPoint:operation:"), Fn: historyTargetDragEnded},
		{Cmd: sel("tableView:willDisplayCell:forTableColumn:row:"), Fn: historyTargetWillDisplayCell},
		{Cmd: sel("menuPaste:"), Fn: menuAction("activate")},
		{Cmd: sel("menuCopy:"), Fn: menuAction("copy")},
		{Cmd: sel("menuCopyPlain:"), Fn: menuAction("copyPlain")},
		{Cmd: sel("menuFavorite:"), Fn: menuAction("toggleFavorite")},
		{Cmd: sel("menuPin:"), Fn: menuAction("togglePinned")},
		{Cmd: sel("menuOpen:"), Fn: menuAction("open")},
		{Cmd: sel("menuReveal:"), Fn: menuAction("reveal")},
		{Cmd: sel("menuSaveImage:"), Fn: menuAction("saveImage")},
		{Cmd: sel("menuNote:"), Fn: historyTargetMenuNote},
		{Cmd: sel("menuMoveGroup:"), Fn: historyTargetMoveGroup},
		{Cmd: sel("menuDelete:"), Fn: historyTargetDelete},
	}
}

func historyTargetMenuNote(_ objc.ID, _ objc.SEL, sender objc.ID) {
	if p := menuPanel(sender); p != nil {
		p.showNotePopover(p.selection)
	}
}

func historyTargetNumberOfRows(_ objc.ID, _ objc.SEL, table objc.ID) int64 {
	if p := panelsByTable[table]; p != nil {
		return int64(len(p.rows))
	}
	return 0
}

func historyTargetViewForRow(_ objc.ID, _ objc.SEL, table, _ objc.ID, row int64) objc.ID {
	p := panelsByTable[table]
	if p == nil || row < 0 || int(row) >= len(p.rows) {
		return 0
	}
	cell := table.Send(sel("makeViewWithIdentifier:owner:"), nsString(rowIdentifier), table)
	if cell == 0 {
		cell = newHistoryRow()
	}
	fillHistoryRow(cell, p.rows[row], p.display, p.originalRow == int(row),
		objc.Send[int64](table, sel("selectedRow")) == row, p.query, int(row)-firstVisibleRow(table), p.commandHeld)
	return cell
}

// historyTargetHeightOfRow returns a card's natural height. The value is
// wrapped in a struct because purego callbacks cannot return a bare
// float64, but a single-field float struct lands in the same register a
// CGFloat return uses.
func historyTargetHeightOfRow(_ objc.ID, _ objc.SEL, table objc.ID, row int64) struct{ V float64 } {
	p := panelsByTable[table]
	if p != nil && row >= 0 && int(row) < len(p.rowHeights) {
		return struct{ V float64 }{p.rowHeights[row]}
	}
	return struct{ V float64 }{rowHeight}
}

func historyTargetRowView(_ objc.ID, _ objc.SEL, _ objc.ID, _ int64) objc.ID {
	rowView := objc.ID(historyRowViewClass).Send(sel("new"))
	return rowView.Send(sel("autorelease"))
}

func historyTargetSelectionDidChange(_ objc.ID, _ objc.SEL, note objc.ID) {
	table := objc.Send[objc.ID](note, sel("object"))
	p := panelsByTable[table]
	if p == nil {
		return
	}
	row := objc.Send[int64](table, sel("selectedRow"))
	if row >= 0 && int(row) < len(p.rows) {
		p.selection = p.rows[row].ID
		p.cx.Emit("select", p.selection)
	}
	// Only the visible cards can have a live cell view. Refreshing just that
	// window keeps a selection change O(visible rows) instead of walking the
	// whole loaded list on every arrow key, which matters for long histories.
	start, end := visibleRowRange(table)
	if end > len(p.rows) {
		end = len(p.rows)
	}
	for index := start; index < end; index++ {
		cell := table.Send(sel("viewAtColumn:row:makeIfNecessary:"), int64(0), int64(index), false)
		if cell != 0 {
			selected := int64(index) == row
			applyHistoryRowText(cell, selected)
			setHistoryRowSelected(cell, selected)
		}
	}
}

// visibleRowRange returns a half-open range covering the rows NSTableView
// currently has on screen, so callers can update cells without touching the
// full list. rowsInRect: reports every row that intersects the visible rect;
// one extra row is added on each side so a partially clipped card at either
// edge is still refreshed.
func visibleRowRange(table objc.ID) (int, int) {
	visible := objc.Send[CGRect](table, sel("visibleRect"))
	if visible.Size.Height <= 0 {
		return 0, 0
	}
	rows := objc.Send[NSRange](table, sel("rowsInRect:"), visible)
	if rows.Length == 0 {
		return 0, 0
	}
	start := int(rows.Location)
	end := start + int(rows.Length) + 1
	if start > 0 {
		start--
	}
	return start, end
}

func historyTargetPasteboardWriter(_ objc.ID, _ objc.SEL, table objc.ID, row int64) objc.ID {
	p := panelsByTable[table]
	if p == nil || row < 0 || int(row) >= len(p.rows) {
		return 0
	}
	return p.pasteboardWriter(p.rows[row])
}

func historyTargetDragWillBegin(self objc.ID, _ objc.SEL, table, session objc.ID, _ CGPoint, _ objc.ID) {
	p := panelsByTable[table]
	if p == nil {
		return
	}
	// A drag starts inside the table's mouse tracking loop,
	// so the queued single-click action must not fire.
	cancelPendingClick(table)
	p.pendingClick = ""
	p.dragging = true
	p.hideHover()
	p.appendDragFiles(session)
	self.Send(sel("performSelector:withObject:afterDelay:"),
		sel("checkDragFinished:"), table, 0.2)
}

func historyTargetCheckDragFinished(self objc.ID, _ objc.SEL, table objc.ID) {
	p := panelsByTable[table]
	if p == nil || !p.dragging {
		return
	}
	pressed := objc.Send[uint64](objc.ID(class("NSEvent")), sel("pressedMouseButtons"))
	if pressed != 0 {
		self.Send(sel("performSelector:withObject:afterDelay:"),
			sel("checkDragFinished:"), table, 0.1)
		return
	}
	finishDrag(p, false)
}

func historyTargetDragEnded(_ objc.ID, _ objc.SEL, table, _ objc.ID, _ CGPoint, operation uint64) {
	finishDrag(panelsByTable[table], operation != 0)
}

func historyTargetWillDisplayCell(_ objc.ID, _ objc.SEL, table, _ objc.ID, _ objc.ID, row int64) {
	p := panelsByTable[table]
	if p != nil && int(row) >= len(p.rows)-8 {
		p.cx.Emit("loadMore", nil)
	}
}

func historyTargetMoveGroup(_ objc.ID, _ objc.SEL, sender objc.ID) {
	p := menuPanel(sender)
	if p == nil {
		return
	}
	tag := int(objc.Send[int64](sender, sel("tag")))
	if tag == 0 {
		p.cx.Emit("moveGroup", "")
		return
	}
	if tag-1 >= 0 && tag-1 < len(p.groups) {
		p.cx.Emit("moveGroup", p.groups[tag-1].ID)
	}
}

func historyTargetDelete(_ objc.ID, _ objc.SEL, sender objc.ID) {
	p := menuPanel(sender)
	if p == nil {
		return
	}
	row := HistoryRow{}
	if index := p.selectedIndex(); index >= 0 {
		row = p.rows[index]
	}
	if row.CanDelete && (!row.DeleteConfirm || confirmDelete()) {
		p.cx.Emit("delete", p.selection)
	}
}
