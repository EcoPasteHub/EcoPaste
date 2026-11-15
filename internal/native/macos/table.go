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
	key1             uint16 = 18
	key2             uint16 = 19
	key3             uint16 = 20
	key4             uint16 = 21
	key6             uint16 = 22
	key5             uint16 = 23
	key9             uint16 = 25
	key7             uint16 = 26
	key8             uint16 = 28
	key0             uint16 = 29
	keyK             uint16 = 40
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
	if shouldHideAfterDrag(visible, accepted, key, active, p.pinned, suppressPanelBlur.Load()) {
		p.cx.Emit("blurHide", nil)
	}
}

func shouldHideAfterDrag(visible, accepted, key, active, pinned, suppressed bool) bool {
	return visible && (accepted || !key || !active) && !pinned && !suppressed
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
	case (key == keyReturn || key == keyEnter) && commandOnly:
		if row.Kind == "text" {
			return "pastePlain", true
		}
		return "activate", true
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
	if p.previewing || !commandShortcut(flags) {
		return false
	}
	window := p.view.Send(sel("window"))
	if window == 0 || !objc.Send[bool](window, sel("isKeyWindow")) {
		return false
	}
	if key == keyK {
		p.toggleShortcutHelp()
		return true
	}
	row := HistoryRow{}
	if index := p.activeIndex(); index >= 0 {
		row = p.rows[index]
	}
	if index, ok := commandDigitIndex(key); ok {
		if index < len(p.rows) {
			p.cx.Emit("activate", p.rows[index].ID)
		}
		return true
	}
	action, handled := historyKeyAction(key, flags, row, p.behavior.SpacePreview)
	switch action {
	case "toggleFavorite", "editNote", "open", "reveal", "newGroup", "pastePlain", "activate":
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
