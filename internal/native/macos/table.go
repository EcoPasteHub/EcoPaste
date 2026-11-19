//go:build darwin

package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

// HistoryFile is one file entry a files card lists on its own line.
type HistoryFile struct {
	Name     string
	IconPath string
	IsDir    bool
	IsImage  bool
	Missing  bool
}

// HistoryRow is one row the native table draws.
type HistoryRow struct {
	ID             string
	GroupID        string
	Kind           string
	SubKind        string
	Title          string
	Preview        string
	Detail         string
	SourceIconPath string
	ImagePath      string
	ImageWidth     int
	ImageHeight    int
	Files          []HistoryFile
	Note           string
	Favorite       bool
	Pinned         bool
	CanDelete      bool
	DeleteConfirm  bool
}

const (
	rowIconTag       = 100
	rowTitleTag      = 101
	rowDetailTag     = 102
	rowPreviewTag    = 104
	rowImageTag      = 105
	rowFileIconBase  = 200
	rowFileLabelBase = 210
	rowIdentifier    = "ecopaste.history-row"
	columnIdentity   = "history"
	rowHeight        = 62.0
	rowSpacing       = 6.0

	// rowLeadingInset is the gap the table leaves above the first card,
	// on top of the card's own inset.
	rowLeadingInset = 3.0

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
	table.Send(sel("setRowHeight:"), rowHeight)
	table.Send(sel("setIntercellSpacing:"), CGSize{Width: 0, Height: rowSpacing})
	table.Send(sel("setHeaderView:"), objc.ID(0))
	table.Send(sel("setGridStyleMask:"), int64(0))
	table.Send(sel("setAllowsMultipleSelection:"), false)
	table.Send(sel("setAllowsEmptySelection:"), false)
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
// which made wheel scrolling work only until the next layout pass. Row
// heights vary with each card's content, so the document height sums them.
func (p *Panel) syncTableFrame() {
	if p.table == 0 || p.scroll == 0 {
		return
	}
	viewport := bounds(p.scroll.Send(sel("contentView")))
	if len(p.rowHeights) != len(p.rows) || p.rowHeightsWidth != p.rowWidth() {
		p.recomputeRowHeights()
	}
	spacing := objc.Send[CGSize](p.table, sel("intercellSpacing")).Height
	height := historyTableContentHeight(p.rowHeights, spacing)
	if height < viewport.Size.Height {
		height = viewport.Size.Height
	}
	setFrame(p.table, 0, 0, viewport.Size.Width, height)
	if column := p.table.Send(sel("tableColumnWithIdentifier:"), nsString(columnIdentity)); column != 0 {
		column.Send(sel("setWidth:"), viewport.Size.Width)
	}
}

// historyTableContentHeight returns the height the table's rows occupy.
// NSTableView gives every row a slot of its own height plus the intercell
// spacing and centers the cell inside that slot, so the last row's slot
// ends spacing points below the row itself. The document view has to
// cover that trailing slot: with a shorter height the list stops one
// point short of the last card and clips its bottom edge when scrolled
// to the end.
func historyTableContentHeight(heights []float64, spacing float64) float64 {
	if len(heights) == 0 {
		return 0
	}
	total := 0.0
	for _, height := range heights {
		total += height + spacing
	}
	return total
}

// rowWidth is the visible list width a card lays out against.
func (p *Panel) rowWidth() float64 {
	if p.scroll != 0 {
		if width := bounds(p.scroll.Send(sel("contentView"))).Size.Width; width > 0 {
			return width
		}
	}
	return 420
}

// recomputeRowHeights derives each card's natural height from its content
// and the configured density. Callers reload the table afterwards so the
// delegate reports the new heights.
func (p *Panel) recomputeRowHeights() {
	width := p.rowWidth()
	heights := make([]float64, len(p.rows))
	for index, row := range p.rows {
		heights[index] = historyRowHeight(row, p.display, width)
	}
	p.rowHeights = heights
	p.rowHeightsWidth = width
}

// setRows swaps in new rows together with their measured heights. The
// loadMore path only extends the current list, so the heights already
// measured for the shared prefix are reused and just the new tail is
// measured. A full replace (search, group or category switch) still reuses
// nothing but resets to one page, so the cost stays bounded by the page.
func (p *Panel) setRows(rows []HistoryRow) {
	width := p.rowWidth()
	heights := make([]float64, len(rows))
	reused := reusedRowHeights(p.rows, p.rowHeights, rows, p.rowHeightsWidth, width)
	if reused > 0 {
		copy(heights, p.rowHeights[:reused])
	}
	for index := reused; index < len(rows); index++ {
		heights[index] = historyRowHeight(rows[index], p.display, width)
	}
	p.rows = rows
	p.rowHeights = heights
	p.rowHeightsWidth = width
}

// reusedRowHeights reports how many leading heights can be kept for rows.
// Heights are only reusable when the list was measured at the same width
// and the new rows extend the previous ones without changing their prefix,
// which is exactly how the next history page is appended.
func reusedRowHeights(previous []HistoryRow, previousHeights []float64, rows []HistoryRow, previousWidth, width float64) int {
	if len(previousHeights) != len(previous) || previousWidth != width || !rowsExtend(previous, rows) {
		return 0
	}
	return len(previous)
}

// rowsExtend reports whether previous is a prefix of rows by row identity,
// which is how the panel appends the next history page.
func rowsExtend(previous, rows []HistoryRow) bool {
	if len(rows) < len(previous) {
		return false
	}
	for index := range previous {
		if previous[index].ID != rows[index].ID {
			return false
		}
	}
	return true
}

// canAppendRows reports whether rows only extends the loaded list, so the
// table can insert the new tail instead of reloading the rows already on
// screen.
func (p *Panel) canAppendRows(rows []HistoryRow) bool {
	if len(p.rows) == 0 || len(rows) <= len(p.rows) {
		return false
	}
	if len(p.rowHeights) != len(p.rows) || p.rowHeightsWidth != p.rowWidth() {
		return false
	}
	return rowsExtend(p.rows, rows)
}

// appendRows inserts the next page in place. The existing cells keep their
// views, so paging further into history no longer flashes a reload of rows
// that did not change. The new tail's heights are measured first because the
// table asks for them while it lays the inserted rows out.
func (p *Panel) appendRows(rows []HistoryRow) {
	oldLen := len(p.rows)
	p.setRows(rows)
	indexes := objc.ID(class("NSMutableIndexSet")).Send(sel("alloc")).Send(sel("init"))
	for index := oldLen; index < len(rows); index++ {
		indexes.Send(sel("addIndex:"), int64(index))
	}
	p.table.Send(sel("insertRowsAtIndexes:withAnimation:"), indexes, uint64(0))
	indexes.Send(sel("release"))
	p.syncTableFrame()
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
	if !handled {
		return false
	}
	switch action {
	case "toggleWindowPinned":
		// Window pin is not tied to a row, so it works even on an empty list.
		p.cx.Emit("toggleWindowPinned", nil)
		return true
	case "newGroup":
		p.showNewGroupPopover(p.newGroupAnchor())
		return true
	case "editNote":
		if row.ID == "" {
			return false
		}
		p.showNotePopover(row.ID)
		return true
	case "toggleFavorite", "togglePinned", "open", "reveal", "pastePlain", "activate":
		if row.ID == "" {
			return false
		}
		p.cx.Emit(action, row.ID)
		return true
	default:
		return false
	}
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
	shiftModifier      uint64 = 1 << 17
	controlModifier    uint64 = 1 << 18
	alternateModifier  uint64 = 1 << 19
	commandModifier    uint64 = 1 << 20
	numericPadModifier uint64 = 1 << 21
	functionModifier   uint64 = 1 << 23
)

func systemSymbol(name string) objc.ID {
	return objc.ID(class("NSImage")).Send(sel("imageWithSystemSymbolName:accessibilityDescription:"),
		nsString(name), nsString(""))
}
