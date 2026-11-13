//go:build darwin

package macos

import (
	"strings"

	"github.com/ebitengine/purego/objc"
)

// HistoryRow is one row the native table draws.
type HistoryRow struct {
	ID             string
	GroupID        string
	Kind           string
	SubKind        string
	Title          string
	Detail         string
	ImagePath      string
	SourceIconPath string
	Note           string
	Favorite       bool
	Pinned         bool
}

const (
	rowIconTag     = 100
	rowTitleTag    = 101
	rowDetailTag   = 102
	rowBadgeTag    = 103
	rowIdentifier  = "ecopaste.history-row"
	columnIdentity = "history"
	rowHeight      = 58.0
)

const (
	keyF             uint16 = 3
	keyC             uint16 = 8
	keyV             uint16 = 9
	keyP             uint16 = 35
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
	keyDownArrow     uint16 = 125
	keyUpArrow       uint16 = 126
)

var (
	historyTarget     objc.ID
	historyTableClass objc.Class
	historyRowClass   objc.Class
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
				action, handled := historyKeyAction(key, flags, row)
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
					if confirmDelete() {
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

	historyRowClass, err = objc.RegisterClass("EcoPasteHistoryRow", class("NSTableCellView"), nil, nil,
		[]objc.MethodDef{{Cmd: sel("setFrameSize:"), Fn: func(self objc.ID, _ objc.SEL, size CGSize) {
			sendSuper[objc.ID](self, historyRowClass, sel("setFrameSize:"), size)
			layoutHistoryRow(self, size)
		}}})
	if err != nil {
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
			fillHistoryRow(cell, p.rows[row])
			return cell
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
		}},
		{Cmd: sel("tableView:willDisplayCell:forTableColumn:row:"), Fn: func(_ objc.ID, _ objc.SEL, table, _ objc.ID, _ objc.ID, row int64) {
			p := panelsByTable[table]
			if p != nil && int(row) >= len(p.rows)-8 {
				p.cx.Emit("loadMore", nil)
			}
		}},
		{Cmd: sel("rowActivated:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			p := panelsByTable[sender]
			if p == nil {
				return
			}
			row := int(objc.Send[int64](sender, sel("clickedRow")))
			p.selectRow(row)
			if row >= 0 && row < len(p.rows) {
				p.cx.Emit("activate", p.rows[row].ID)
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
			if p != nil && confirmDelete() {
				p.cx.Emit("delete", p.selection)
			}
		}},
	})
	if err != nil {
		panic(err)
	}
	historyTarget = objc.ID(target).Send(sel("new"))
}

func newHistoryTable(p *Panel) objc.ID {
	scroll := objc.ID(class("NSScrollView")).Send(sel("new"))
	scroll.Send(sel("setHasVerticalScroller:"), true)
	scroll.Send(sel("setAutohidesScrollers:"), true)
	scroll.Send(sel("setDrawsBackground:"), false)
	scroll.Send(sel("setBorderType:"), int64(0))

	table := objc.ID(historyTableClass).Send(sel("new"))
	p.table = table
	table.Send(sel("setBackgroundColor:"), objc.ID(classNSColor).Send(sel("clearColor")))
	table.Send(sel("setRowHeight:"), rowHeight)
	table.Send(sel("setIntercellSpacing:"), CGSize{})
	table.Send(sel("setHeaderView:"), objc.ID(0))
	table.Send(sel("setGridStyleMask:"), int64(0))
	table.Send(sel("setAllowsMultipleSelection:"), false)
	table.Send(sel("setAllowsEmptySelection:"), true)
	table.Send(sel("setAllowsColumnReordering:"), false)
	table.Send(sel("setAllowsColumnResizing:"), false)
	table.Send(sel("setSelectionHighlightStyle:"), int64(1))
	table.Send(sel("setFocusRingType:"), int64(1))
	table.Send(sel("setStyle:"), int64(3))
	table.Send(sel("setDelegate:"), historyTarget)
	table.Send(sel("setDataSource:"), historyTarget)
	table.Send(sel("setTarget:"), historyTarget)
	table.Send(sel("setDoubleAction:"), sel("rowActivated:"))
	table.Send(sel("setAccessibilityLabel:"), nsString("剪贴板历史"))
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

func historyKeyAction(key uint16, flags uint64, row HistoryRow) (string, bool) {
	command := flags&commandModifier != 0
	option := flags&alternateModifier != 0
	unmodified := flags&(shiftModifier|controlModifier|alternateModifier|commandModifier) == 0
	switch {
	case key == keyReturn || key == keyEnter:
		return "activate", true
	case key == keyP && unmodified:
		return "togglePinned", true
	case key == keySpace && unmodified:
		return "preview", true
	case key == keyEscape:
		return "hide", true
	case key == keyDelete || key == keyForwardDelete:
		return "delete", true
	case key == keyHome && unmodified:
		return "firstRow", true
	case key == keyEnd && unmodified:
		return "lastRow", true
	case (key == keyPageUp || key == keyUpArrow && flags&functionModifier != 0) && unmodified:
		return "pageUp", true
	case (key == keyPageDown || key == keyDownArrow && flags&functionModifier != 0) && unmodified:
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

func newHistoryRow() objc.ID {
	cell := objc.ID(historyRowClass).Send(sel("new"))
	cell.Send(sel("setIdentifier:"), nsString(rowIdentifier))

	icon := objc.ID(class("NSImageView")).Send(sel("new"))
	icon.Send(sel("setTag:"), int64(rowIconTag))
	icon.Send(sel("setImageScaling:"), int64(3))
	icon.Send(sel("setAccessibilityElement:"), false)
	cell.Send(sel("addSubview:"), icon)
	icon.Send(sel("release"))

	badge := objc.ID(class("NSImageView")).Send(sel("new"))
	badge.Send(sel("setTag:"), int64(rowBadgeTag))
	badge.Send(sel("setImageScaling:"), int64(3))
	badge.Send(sel("setAccessibilityElement:"), false)
	cell.Send(sel("addSubview:"), badge)
	badge.Send(sel("release"))

	title := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
	title.Send(sel("setTag:"), int64(rowTitleTag))
	title.Send(sel("setFont:"), objc.ID(classNSFont).Send(sel("boldSystemFontOfSize:"), 13.0))
	title.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("labelColor")))
	title.Send(sel("setLineBreakMode:"), int64(4))
	title.Send(sel("setMaximumNumberOfLines:"), int64(1))
	cell.Send(sel("addSubview:"), title)

	detail := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
	detail.Send(sel("setTag:"), int64(rowDetailTag))
	detail.Send(sel("setFont:"), systemFont(11))
	detail.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	detail.Send(sel("setLineBreakMode:"), int64(4))
	detail.Send(sel("setMaximumNumberOfLines:"), int64(1))
	cell.Send(sel("addSubview:"), detail)

	return cell.Send(sel("autorelease"))
}

func fillHistoryRow(cell objc.ID, row HistoryRow) {
	icon := cell.Send(sel("viewWithTag:"), int64(rowIconTag))
	badge := cell.Send(sel("viewWithTag:"), int64(rowBadgeTag))
	title := cell.Send(sel("viewWithTag:"), int64(rowTitleTag))
	detail := cell.Send(sel("viewWithTag:"), int64(rowDetailTag))
	title.Send(sel("setStringValue:"), nsString(row.Title))
	detail.Send(sel("setStringValue:"), nsString(row.Detail))

	image := objc.ID(0)
	if row.ImagePath != "" {
		image = loadImage(row.ImagePath)
	}
	sourceIcon := objc.ID(0)
	if row.SourceIconPath != "" {
		sourceIcon = loadImage(row.SourceIconPath)
	}
	badge.Send(sel("setHidden:"), true)
	switch {
	case image != 0:
		icon.Send(sel("setImage:"), image)
		icon.Send(sel("setContentTintColor:"), objc.ID(0))
		if sourceIcon != 0 {
			badge.Send(sel("setImage:"), sourceIcon)
			badge.Send(sel("setContentTintColor:"), objc.ID(0))
			badge.Send(sel("setHidden:"), false)
		}
	case sourceIcon != 0:
		icon.Send(sel("setImage:"), sourceIcon)
		icon.Send(sel("setContentTintColor:"), objc.ID(0))
		badge.Send(sel("setImage:"), symbolImage(row.Kind))
		badge.Send(sel("setContentTintColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
		badge.Send(sel("setHidden:"), false)
	default:
		icon.Send(sel("setImage:"), symbolImage(row.Kind))
		icon.Send(sel("setContentTintColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	}
}

func loadImage(path string) objc.ID {
	image := objc.ID(class("NSImage")).Send(sel("alloc")).Send(sel("initWithContentsOfFile:"), nsString(path))
	if image == 0 {
		return 0
	}
	return image.Send(sel("autorelease"))
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

func layoutHistoryRow(cell objc.ID, size CGSize) {
	const left = 14.0
	const iconSize = 32.0
	const badgeSize = 15.0
	const textX = 58.0
	icon := cell.Send(sel("viewWithTag:"), int64(rowIconTag))
	badge := cell.Send(sel("viewWithTag:"), int64(rowBadgeTag))
	title := cell.Send(sel("viewWithTag:"), int64(rowTitleTag))
	detail := cell.Send(sel("viewWithTag:"), int64(rowDetailTag))
	iconY := (size.Height - iconSize) / 2
	setFrame(icon, left, iconY, iconSize, iconSize)
	setFrame(badge, left+iconSize-badgeSize, iconY+1, badgeSize, badgeSize)
	setFrame(title, textX, size.Height-30, size.Width-textX-14, 18)
	setFrame(detail, textX, 12, size.Width-textX-14, 15)
}

func (p *Panel) populateMenu(menu objc.ID, row HistoryRow) {
	menu.Send(sel("removeAllItems"))
	addMenuItem(menu, p.table, "粘贴", "menuPaste:", "arrow.down.doc", "")
	addMenuItem(menu, p.table, "复制", "menuCopy:", "doc.on.doc", "c")
	if row.Kind == "text" {
		plainPaste := addMenuItem(menu, p.table, "纯文本粘贴", "menuPastePlain:", "textformat", "v")
		setMenuModifiers(plainPaste, commandModifier|alternateModifier)
		plainCopy := addMenuItem(menu, p.table, "复制为纯文本", "menuCopyPlain:", "doc.plaintext", "c")
		setMenuModifiers(plainCopy, commandModifier|alternateModifier)
	}
	if row.Kind == "files" {
		pathPaste := addMenuItem(menu, p.table, "粘贴为路径", "menuPastePath:", "point.topleft.down.to.point.bottomright.curvepath", "v")
		setMenuModifiers(pathPaste, commandModifier|alternateModifier)
	}
	switch row.SubKind {
	case "url":
		addMenuItem(menu, p.table, "打开链接", "menuOpen:", "arrow.up.right.square", "")
	case "email":
		addMenuItem(menu, p.table, "发送邮件", "menuOpen:", "envelope", "")
	}
	if row.SubKind == "path" || row.Kind == "files" {
		addMenuItem(menu, p.table, "在 Finder 中显示", "menuReveal:", "folder", "")
	}
	if row.Kind == "image" {
		addMenuItem(menu, p.table, "存储图片…", "menuSaveImage:", "square.and.arrow.down", "")
	}
	note := "添加备注…"
	if strings.TrimSpace(row.Note) != "" {
		note = "编辑备注…"
	}
	addMenuItem(menu, p.table, note, "menuNote:", "note.text", "")
	menu.Send(sel("addItem:"), objc.ID(class("NSMenuItem")).Send(sel("separatorItem")))

	favorite := "收藏"
	if row.Favorite {
		favorite = "取消收藏"
	}
	addMenuItem(menu, p.table, favorite, "menuFavorite:", "star", "")
	pin := "置顶"
	if row.Pinned {
		pin = "取消置顶"
	}
	addMenuItem(menu, p.table, pin, "menuPin:", "pin", "")
	moveItem := objc.ID(class("NSMenuItem")).Send(sel("alloc")).
		Send(sel("initWithTitle:action:keyEquivalent:"), nsString("移动到分组"), objc.SEL(0), nsString(""))
	moveItem.Send(sel("setImage:"), systemSymbol("folder"))
	submenu := objc.ID(class("NSMenu")).Send(sel("new"))
	addMoveGroupItem(submenu, p, "未分组", "", row.GroupID == "")
	if len(p.groups) > 0 {
		submenu.Send(sel("addItem:"), objc.ID(class("NSMenuItem")).Send(sel("separatorItem")))
	}
	for _, group := range p.groups {
		addMoveGroupItem(submenu, p, group.Name, group.ID, row.GroupID == group.ID)
	}
	moveItem.Send(sel("setSubmenu:"), submenu)
	submenu.Send(sel("release"))
	menu.Send(sel("addItem:"), moveItem)
	moveItem.Send(sel("release"))
	menu.Send(sel("addItem:"), objc.ID(class("NSMenuItem")).Send(sel("separatorItem")))
	addMenuItem(menu, p.table, "删除", "menuDelete:", "trash", "")
}

func addMoveGroupItem(menu objc.ID, p *Panel, title, groupID string, selected bool) {
	item := objc.ID(class("NSMenuItem")).Send(sel("alloc")).
		Send(sel("initWithTitle:action:keyEquivalent:"), nsString(title), sel("menuMoveGroup:"), nsString(""))
	item.Send(sel("setTarget:"), historyTarget)
	item.Send(sel("setRepresentedObject:"), p.table)
	tag := int64(0)
	if groupID != "" {
		for index, group := range p.groups {
			if group.ID == groupID {
				tag = int64(index + 1)
				break
			}
		}
	}
	item.Send(sel("setTag:"), tag)
	if selected {
		item.Send(sel("setState:"), int64(1))
	}
	menu.Send(sel("addItem:"), item)
	item.Send(sel("release"))
}

func addMenuItem(menu, table objc.ID, title, action, icon, key string) objc.ID {
	item := objc.ID(class("NSMenuItem")).Send(sel("alloc")).Send(sel("initWithTitle:action:keyEquivalent:"),
		nsString(title), sel(action), nsString(key))
	item.Send(sel("setTarget:"), historyTarget)
	item.Send(sel("setRepresentedObject:"), table)
	item.Send(sel("setImage:"), systemSymbol(icon))
	menu.Send(sel("addItem:"), item)
	item.Send(sel("release"))
	return item
}

const (
	shiftModifier     uint64 = 1 << 17
	controlModifier   uint64 = 1 << 18
	alternateModifier uint64 = 1 << 19
	commandModifier   uint64 = 1 << 20
	functionModifier  uint64 = 1 << 23
)

func setMenuModifiers(item objc.ID, modifiers uint64) {
	item.Send(sel("setKeyEquivalentModifierMask:"), modifiers)
}

func menuPanel(item objc.ID) *Panel {
	table := objc.Send[objc.ID](item, sel("representedObject"))
	return panelsByTable[table]
}

func menuAction(event string) func(objc.ID, objc.SEL, objc.ID) {
	return func(_ objc.ID, _ objc.SEL, sender objc.ID) {
		if p := menuPanel(sender); p != nil {
			p.cx.Emit(event, p.selection)
		}
	}
}

func confirmDelete() bool {
	alert := objc.ID(class("NSAlert")).Send(sel("new"))
	defer alert.Send(sel("release"))
	alert.Send(sel("setMessageText:"), nsString("删除这条记录？"))
	alert.Send(sel("setInformativeText:"), nsString("删除后无法恢复。"))
	alert.Send(sel("addButtonWithTitle:"), nsString("删除"))
	alert.Send(sel("addButtonWithTitle:"), nsString("取消"))
	return alert.Send(sel("runModal")) == 1000
}

func systemSymbol(name string) objc.ID {
	return objc.ID(class("NSImage")).Send(sel("imageWithSystemSymbolName:accessibilityDescription:"),
		nsString(name), nsString(""))
}
