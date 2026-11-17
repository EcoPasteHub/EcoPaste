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
// which made wheel scrolling work only until the next layout pass. Row
// heights vary with each card's content, so the document height sums them.
func (p *Panel) syncTableFrame() {
	if p.table == 0 || p.scroll == 0 {
		return
	}
	viewport := bounds(p.scroll.Send(sel("contentView")))
	if len(p.rowHeights) != len(p.rows) {
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

func historyTableContentHeight(heights []float64, spacing float64) float64 {
	if len(heights) == 0 {
		return 0
	}
	total := 0.0
	for index, height := range heights {
		total += height
		if index > 0 {
			total += spacing
		}
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
