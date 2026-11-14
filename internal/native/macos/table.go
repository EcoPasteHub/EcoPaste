//go:build darwin

package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

// HistoryRow is one row the native table draws.
type HistoryRow struct {
	ID             string
	GroupID        string
	Kind           string
	SubKind        string
	Title          string
	Preview        string
	Detail         string
	ImagePath      string
	SourceIconPath string
	Note           string
	Favorite       bool
	Pinned         bool
	CanDelete      bool
	DeleteConfirm  bool
}

const (
	rowIconTag     = 100
	rowTitleTag    = 101
	rowDetailTag   = 102
	rowBadgeTag    = 103
	rowPreviewTag  = 104
	rowIdentifier  = "ecopaste.history-row"
	columnIdentity = "history"
	rowHeight      = 62.0
	rowSpacing     = 6.0

	// NSTableViewStylePlain removes the leading content padding that
	// NSTableViewStyleFullWidth adds to every row.
	nsTableViewStylePlain = 4
)

type edgeInset struct {
	Top, Left, Bottom, Right float64
}

func edgeInsets(top, left, bottom, right float64) edgeInset {
	return edgeInset{Top: top, Left: left, Bottom: bottom, Right: right}
}

const (
	keyF             uint16 = 3
	keyC             uint16 = 8
	keyV             uint16 = 9
	keyT             uint16 = 17
	keyO             uint16 = 31
	keyP             uint16 = 35
	keyD             uint16 = 2
	keyM             uint16 = 46
	keyN             uint16 = 45
	keyReturn        uint16 = 36
	keySpace         uint16 = 49
	keyDelete        uint16 = 51
	keyEscape        uint16 = 53
	keyEnter         uint16 = 76
	keyHome          uint16 = 115
	keyPageUp        uint16 = 116
	keyForwardDelete uint16 = 117
	keyEnd           uint16 = 119
	keyPageDown      uint16 = 121
	keyLeftArrow     uint16 = 123
	keyRightArrow    uint16 = 124
	keyDownArrow     uint16 = 125
	keyUpArrow       uint16 = 126
)

var (
	historyTarget     objc.ID
	historyTableClass objc.Class
	panelsByTable     = map[objc.ID]*Panel{}
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

func finishDrag(p *Panel, accepted bool) {
	if p == nil || !p.dragging {
		return
	}
	p.dragging = false
	window := p.view.Send(sel("window"))
	if window == 0 {
		return
	}
	visible := objc.Send[bool](window, sel("isVisible"))
	key := objc.Send[bool](window, sel("isKeyWindow"))
	app := objc.ID(class("NSApplication")).Send(sel("sharedApplication"))
	active := objc.Send[bool](app, sel("isActive"))
	if shouldHideAfterDrag(visible, accepted, key, active, p.pinned, p.onboardingActive, suppressPanelBlur.Load()) {
		p.cx.Emit("blurHide", nil)
	}
}

func shouldHideAfterDrag(visible, accepted, key, active, pinned, onboarding, suppressed bool) bool {
	return visible && (accepted || !key || !active) && !pinned && !onboarding && !suppressed
}

func newHistoryTable(p *Panel) objc.ID {
	scroll := objc.ID(class("NSScrollView")).Send(sel("new"))
	scroll.Send(sel("setHasVerticalScroller:"), true)
	scroll.Send(sel("setAutohidesScrollers:"), true)
	scroll.Send(sel("setDrawsBackground:"), false)
	scroll.Send(sel("setBorderType:"), int64(0))
	scroll.Send(sel("setAutomaticallyAdjustsContentInsets:"), false)
	scroll.Send(sel("setContentInsets:"), edgeInsets(0, 0, 0, 0))

	table := objc.ID(historyTableClass).Send(sel("new"))
	p.table = table
	table.Send(sel("setBackgroundColor:"), objc.ID(classNSColor).Send(sel("clearColor")))
	table.Send(sel("setRowHeight:"), tableRowHeight(p.display))
	table.Send(sel("setIntercellSpacing:"), CGSize{Width: 0, Height: rowSpacing})
	table.Send(sel("setHeaderView:"), objc.ID(0))
	table.Send(sel("setGridStyleMask:"), int64(0))
	table.Send(sel("setAllowsMultipleSelection:"), false)
	table.Send(sel("setAllowsEmptySelection:"), true)
	table.Send(sel("setAllowsColumnReordering:"), false)
	table.Send(sel("setAllowsColumnResizing:"), false)
	table.Send(sel("setSelectionHighlightStyle:"), int64(0))
	table.Send(sel("setFocusRingType:"), int64(1))
	table.Send(sel("setStyle:"), int64(nsTableViewStylePlain))
	table.Send(sel("setDraggingSourceOperationMask:forLocal:"), uint64(1), true)
	table.Send(sel("setDraggingSourceOperationMask:forLocal:"), uint64(1), false)
	table.Send(sel("setDelegate:"), historyTarget)
	table.Send(sel("setDataSource:"), historyTarget)
	table.Send(sel("setTarget:"), historyTarget)
	table.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.PanelHistoryLabel)))
	menu := objc.ID(class("NSMenu")).Send(sel("new"))
	menu.Send(sel("setAutoenablesItems:"), false)
	menu.Send(sel("setDelegate:"), table)
	table.Send(sel("setMenu:"), menu)
	menu.Send(sel("release"))

	column := objc.ID(class("NSTableColumn")).Send(sel("new"))
	column.Send(sel("setIdentifier:"), nsString(columnIdentity))
	column.Send(sel("setResizingMask:"), int64(1))
	column.Send(sel("setWidth:"), float64(420))
	table.Send(sel("addTableColumn:"), column)
	column.Send(sel("release"))
	table.Send(sel("setColumnAutoresizingStyle:"), int64(1))

	scroll.Send(sel("setDocumentView:"), table)
	panelsByTable[table] = p
	return scroll
}

// syncTableFrame keeps the table's document height in step with the rows
// it holds. Sizing it to the viewport would leave nothing to scroll,
// which made wheel scrolling work only until the next layout pass.
func (p *Panel) syncTableFrame() {
	if p.table == 0 || p.scroll == 0 {
		return
	}
	viewport := bounds(p.scroll.Send(sel("contentView")))
	rowHeight := objc.Send[float64](p.table, sel("rowHeight"))
	if rowHeight <= 0 {
		rowHeight = tableRowHeight(p.display)
	}
	spacing := objc.Send[CGSize](p.table, sel("intercellSpacing")).Height
	height := historyTableContentHeight(len(p.rows), rowHeight, spacing)
	if height < viewport.Size.Height {
		height = viewport.Size.Height
	}
	setFrame(p.table, 0, 0, viewport.Size.Width, height)
	if column := p.table.Send(sel("tableColumnWithIdentifier:"), nsString(columnIdentity)); column != 0 {
		column.Send(sel("setWidth:"), viewport.Size.Width)
	}
}

func historyTableContentHeight(rows int, rowHeight, spacing float64) float64 {
	if rows <= 0 {
		return 0
	}
	return float64(rows)*rowHeight + float64(rows-1)*spacing
}

func cancelPendingClick(table objc.ID) {
	if table == 0 {
		return
	}
	objc.ID(class("NSObject")).Send(sel("cancelPreviousPerformRequestsWithTarget:selector:object:"),
		table, sel("activatePendingClick:"), objc.ID(0))
}

func doubleClickDelay() float64 {
	delay := objc.Send[float64](objc.ID(class("NSEvent")), sel("doubleClickInterval"))
	if delay < 0.2 {
		delay = 0.2
	}
	return delay + 0.05
}

func historyKeyAction(key uint16, flags uint64, row HistoryRow, spacePreview bool) (string, bool) {
	command := flags&commandModifier != 0
	option := flags&alternateModifier != 0
	function := flags&functionModifier != 0
	unmodified := flags&(shiftModifier|controlModifier|alternateModifier|commandModifier) == 0
	commandOnly := commandShortcut(flags)
	switch {
	case key == keyReturn || key == keyEnter:
		return "activate", true
	case key == keyP && commandOnly:
		return "toggleWindowPinned", true
	case key == keyT && commandOnly:
		return "togglePinned", true
	case key == keyD && commandOnly:
		return "toggleFavorite", true
	case key == keyM && commandOnly:
		return "editNote", true
	case key == keyN && commandOnly:
		return "newGroup", true
	case key == keyO && commandOnly && rowOpenable(row):
		if row.Kind == "files" || row.SubKind == "path" {
			return "reveal", true
		}
		return "open", true
	case key == keySpace && unmodified && spacePreview:
		return "preview", true
	case key == keyEscape:
		return "hide", true
	case key == keyDelete || key == keyForwardDelete:
		return "delete", true
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

func loadImage(path string) objc.ID {
	image := objc.ID(class("NSImage")).Send(sel("alloc")).Send(sel("initWithContentsOfFile:"), nsString(path))
	if image == 0 {
		return 0
	}
	return image.Send(sel("autorelease"))
}

// handleCommandKey claims the item shortcuts before the Window menu or
// the focused text field can take them. Menu key equivalents win over
// the table's keyDown, so Command-M would otherwise become Minimize.
// The search field keeps native editing keys; only the item-level
// commands are routed here, with the same first-row fallback as the
// legacy list.
func (p *Panel) handleCommandKey(key uint16, flags uint64) bool {
	if p.previewing || p.onboardingActive || !commandShortcut(flags) {
		return false
	}
	window := p.view.Send(sel("window"))
	if window == 0 || !objc.Send[bool](window, sel("isKeyWindow")) {
		return false
	}
	row := HistoryRow{}
	if index := p.activeIndex(); index >= 0 {
		row = p.rows[index]
	}
	action, handled := historyKeyAction(key, flags, row, p.behavior.SpacePreview)
	switch action {
	case "toggleFavorite", "editNote", "open", "reveal", "newGroup":
	default:
		return false
	}
	if !handled {
		return false
	}
	if action != "newGroup" && row.ID == "" {
		return false
	}
	p.cx.Emit(action, row.ID)
	return true
}

// activeIndex prefers the selected row and falls back to the first one,
// matching the legacy list's active-item rule.
func (p *Panel) activeIndex() int {
	if index := p.selectedIndex(); index >= 0 {
		return index
	}
	if len(p.rows) > 0 {
		return 0
	}
	return -1
}

func symbolImage(kind string) objc.ID {
	name := "doc.text"
	switch kind {
	case "image":
		name = "photo"
	case "files":
		name = "folder"
	}
	return systemSymbol(name)
}

const (
	shiftModifier     uint64 = 1 << 17
	controlModifier   uint64 = 1 << 18
	alternateModifier uint64 = 1 << 19
	commandModifier   uint64 = 1 << 20
	functionModifier  uint64 = 1 << 23
)

func systemSymbol(name string) objc.ID {
	return objc.ID(class("NSImage")).Send(sel("imageWithSystemSymbolName:accessibilityDescription:"),
		nsString(name), nsString(""))
}
