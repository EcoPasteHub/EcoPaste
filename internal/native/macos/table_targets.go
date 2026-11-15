package macos

import (
	"github.com/ebitengine/purego/objc"
)

func registerHistoryTargets() {
	var err error
	historyTableClass, err = objc.RegisterClass("EcoPasteHistoryTable", class("NSTableView"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("copy:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if p := panelsByTable[self]; p != nil && p.selection != "" {
					p.cx.Emit("copy", p.selection)
				}
			}},
			{Cmd: sel("paste:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if p := panelsByTable[self]; p != nil && p.selection != "" {
					p.cx.Emit("activate", p.selection)
				}
			}},
			{Cmd: sel("rightMouseDown:"), Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
				if p := panelsByTable[self]; p != nil {
					location := objc.Send[CGPoint](event, sel("locationInWindow"))
					local := objc.Send[CGPoint](self, sel("convertPoint:fromView:"), location, objc.ID(0))
					row := int(objc.Send[int64](self, sel("rowAtPoint:"), local))
					if row >= 0 && row < len(p.rows) {
						p.selectRow(row)
					}
				}
				sendSuper[objc.ID](self, historyTableClass, sel("rightMouseDown:"), event)
			}},
			{Cmd: sel("otherMouseDown:"), Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
				p := panelsByTable[self]
				if p == nil || objc.Send[int64](event, sel("buttonNumber")) != 2 {
					return
				}
				location := objc.Send[CGPoint](event, sel("locationInWindow"))
				local := objc.Send[CGPoint](self, sel("convertPoint:fromView:"), location, objc.ID(0))
				row := int(objc.Send[int64](self, sel("rowAtPoint:"), local))
				if row < 0 || row >= len(p.rows) {
					return
				}
				p.selectRow(row)
				p.cx.Emit("middleClick", p.rows[row].ID)
			}},
			{Cmd: sel("otherMouseUp:"), Fn: func(objc.ID, objc.SEL, objc.ID) {}},
			{Cmd: sel("mouseDown:"), Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
				left := objc.Send[int64](event, sel("buttonNumber")) == 0
				var row, clicks int
				if left {
					location := objc.Send[CGPoint](event, sel("locationInWindow"))
					local := objc.Send[CGPoint](self, sel("convertPoint:fromView:"), location, objc.ID(0))
					row = int(objc.Send[int64](self, sel("rowAtPoint:"), local))
					clicks = int(objc.Send[int64](event, sel("clickCount")))
				}
				p := panelsByTable[self]
				if left && row < 0 && p != nil && panelDragWindow(p, p.view.Send(sel("window")), func() {
					self.Send(sel("deselectAll:"), objc.ID(0))
				}) != 0 {
					return
				}
				sendSuper[objc.ID](self, historyTableClass, sel("mouseDown:"), event)
				if p != nil && left {
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
			}},
			{Cmd: sel("activatePendingClick:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				p := panelsByTable[self]
				if p == nil || p.pendingClick == "" {
					return
				}
				id := p.pendingClick
				p.pendingClick = ""
				p.cx.Emit("activateSingleClick", id)
			}},
			{Cmd: sel("keyDown:"), Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
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
			}},
			{Cmd: sel("menuNeedsUpdate:"), Fn: func(self objc.ID, _ objc.SEL, menu objc.ID) {
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
			}},
		})
	if err != nil {
		panic(err)
	}

	if err := registerHistoryRowViews(); err != nil {
		panic(err)
	}

	target, err := objc.RegisterClass("EcoPasteHistoryTarget", class("NSObject"), nil, nil, []objc.MethodDef{
		{Cmd: sel("numberOfRowsInTableView:"), Fn: func(_ objc.ID, _ objc.SEL, table objc.ID) int64 {
			if p := panelsByTable[table]; p != nil {
				return int64(len(p.rows))
			}
			return 0
		}},
		{Cmd: sel("tableView:viewForTableColumn:row:"), Fn: func(_ objc.ID, _ objc.SEL, table, _ objc.ID, row int64) objc.ID {
			p := panelsByTable[table]
			if p == nil || row < 0 || int(row) >= len(p.rows) {
				return 0
			}
			cell := table.Send(sel("makeViewWithIdentifier:owner:"), nsString(rowIdentifier), table)
			if cell == 0 {
				cell = newHistoryRow()
			}
			fillHistoryRow(cell, p.rows[row], p.display, p.originalRow == int(row),
				objc.Send[int64](table, sel("selectedRow")) == row)
			return cell
		}},
		{Cmd: sel("tableView:rowViewForRow:"), Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID, _ int64) objc.ID {
			rowView := objc.ID(historyRowViewClass).Send(sel("new"))
			return rowView.Send(sel("autorelease"))
		}},
		{Cmd: sel("tableViewSelectionDidChange:"), Fn: func(_ objc.ID, _ objc.SEL, note objc.ID) {
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
			for index := range p.rows {
				cell := table.Send(sel("viewAtColumn:row:makeIfNecessary:"), int64(0), int64(index), false)
				if cell != 0 {
					selected := int64(index) == row
					setHistoryRowTextColors(cell, selected)
					setHistoryRowSelected(cell, selected)
				}
			}
		}},
		{Cmd: sel("tableView:pasteboardWriterForRow:"), Fn: func(_ objc.ID, _ objc.SEL, table objc.ID, row int64) objc.ID {
			p := panelsByTable[table]
			if p == nil || row < 0 || int(row) >= len(p.rows) {
				return 0
			}
			return p.pasteboardWriter(p.rows[row])
		}},
		{Cmd: sel("tableView:draggingSession:willBeginAtPoint:forRowIndexes:"), Fn: func(self objc.ID, _ objc.SEL, table, session objc.ID, _ CGPoint, _ objc.ID) {
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
		}},
		{Cmd: sel("checkDragFinished:"), Fn: func(self objc.ID, _ objc.SEL, table objc.ID) {
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
		}},
		{Cmd: sel("tableView:draggingSession:endedAtPoint:operation:"), Fn: func(_ objc.ID, _ objc.SEL, table, _ objc.ID, _ CGPoint, operation uint64) {
			finishDrag(panelsByTable[table], operation != 0)
		}},
		{Cmd: sel("tableView:willDisplayCell:forTableColumn:row:"), Fn: func(_ objc.ID, _ objc.SEL, table, _ objc.ID, _ objc.ID, row int64) {
			p := panelsByTable[table]
			if p != nil && int(row) >= len(p.rows)-8 {
				p.cx.Emit("loadMore", nil)
			}
		}},
		{Cmd: sel("menuPaste:"), Fn: menuAction("activate")},
		{Cmd: sel("menuPastePlain:"), Fn: menuAction("pastePlain")},
		{Cmd: sel("menuPastePath:"), Fn: menuAction("pastePath")},
		{Cmd: sel("menuCopy:"), Fn: menuAction("copy")},
		{Cmd: sel("menuCopyPlain:"), Fn: menuAction("copyPlain")},
		{Cmd: sel("menuFavorite:"), Fn: menuAction("toggleFavorite")},
		{Cmd: sel("menuPin:"), Fn: menuAction("togglePinned")},
		{Cmd: sel("menuOpen:"), Fn: menuAction("open")},
		{Cmd: sel("menuReveal:"), Fn: menuAction("reveal")},
		{Cmd: sel("menuSaveImage:"), Fn: menuAction("saveImage")},
		{Cmd: sel("menuNote:"), Fn: menuAction("editNote")},
		{Cmd: sel("menuMoveGroup:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
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
		}},
		{Cmd: sel("menuDelete:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
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
		}},
	})
	if err != nil {
		panic(err)
	}
	historyTarget = objc.ID(target).Send(sel("new"))
}
