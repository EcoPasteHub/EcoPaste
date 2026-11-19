//go:build darwin

package macos

import (
	"sync"
	"sync/atomic"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
	"github.com/nsgo-dev/nsgo/core"
)

// PanelComponent is the native AppKit view that fills the clipboard
// panel.
const PanelComponent = "ecopaste.panel"

// Panel is the AppKit content view of the clipboard panel: search, the
// native history list and a compact footer.
type Panel struct {
	cx                 *core.NativeContext
	behavior           PanelBehavior
	view               objc.ID
	search             objc.ID
	pin                objc.ID
	filterButton       objc.ID
	settingsButton     objc.ID
	clearButton        objc.ID
	shortcutButton     objc.ID
	shortcutPopover    objc.ID
	shortcutView       objc.ID
	shortcutController objc.ID
	shortcutTitle      objc.ID
	shortcutLabels     []objc.ID
	shortcutKeys       []objc.ID
	groupPopover       objc.ID
	groupPopoverView   objc.ID
	groupController    objc.ID
	groupTitle         objc.ID
	groupInput         *textInput
	groupClosing       bool
	groupSubmitted     bool
	notePopover        objc.ID
	notePopoverView    objc.ID
	noteController     objc.ID
	noteTitle          objc.ID
	noteHint           objc.ID
	noteArea           *textArea
	noteSave           objc.ID
	noteCancel         objc.ID
	noteClosing        bool
	noteTargetID       string
	divider            objc.ID
	scroll             objc.ID
	table              objc.ID
	empty              objc.ID
	emptyIcons         map[emptyStateKind]emptyIconView
	emptyDetail        objc.ID
	footerDivider      objc.ID
	footer             objc.ID
	preview            objc.ID
	previewPopover     objc.ID
	previewController  objc.ID
	previewDelegate    objc.ID
	observer           objc.ID
	hoverMonitor       objc.ID
	viewTracking       objc.ID
	monitor            objc.ID
	previewTitle       objc.ID
	previewMeta        objc.ID
	previewDivider     objc.ID
	previewBackdrop    objc.ID
	previewText        objc.ID
	previewScroll      objc.ID
	previewImage       objc.ID
	previewImageSize   CGSize
	drag               func(string) (DragPayload, bool)
	dragFiles          []string
	actions            []string
	hoverRow           int
	hoveredRow         int
	hoverBar           objc.ID
	hoverButtons       []objc.ID
	hoverSpecs         []hoverAction
	hoverBarWidth      float64
	originalRow        int
	query              string
	rows               []HistoryRow
	rowHeights         []float64
	rowHeightsWidth    float64
	total              int64
	loading            bool
	errText            string
	selection          string
	selectionIntent    selectionIntent
	pendingClick       string
	previewing         bool
	filterValue        string
	groupID            string
	groups             []GroupOption
	display            PanelDisplay
	pinned             bool
	dragging           bool
	dragMonitor        objc.ID
	nav                *panelNav
	background         objc.ID
	language           string
}

// PanelBehavior controls focus and query persistence when the panel opens
// and hides.
type PanelBehavior struct {
	DefaultFocus bool
	ClearOnHide  bool
	SpacePreview bool
}

// GroupOption is one custom group in the panel toolbar and item menu.
type GroupOption struct {
	ID   string
	Name string
}

// PanelDisplay is the list density configured in settings.
type PanelDisplay struct {
	TextMaxLines   int
	ImageMaxHeight int
	FileMaxCount   int
	ShowOriginal   bool
}

var (
	suppressPanelBlur atomic.Bool

	registerPanelOnce sync.Once
	searchTarget      objc.ID
	searchFieldClass  objc.Class
	observerClass     objc.Class
	panelViewClass    objc.Class

	panelsBySearch   = map[objc.ID]*Panel{}
	panelsByControl  = map[objc.ID]*Panel{}
	panelsByObserver = map[objc.ID]*Panel{}
	panelsByView     = map[objc.ID]*Panel{}
)

// NewPanel makes the panel's AppKit view.
func NewPanel(cx *core.NativeContext) core.NativeComponent {
	registerPanelOnce.Do(registerPanelTargets)

	p := &Panel{
		cx:          cx,
		view:        objc.ID(panelViewClass).Send(sel("new")),
		display:     PanelDisplay{TextMaxLines: 1, ImageMaxHeight: 32, FileMaxCount: 3},
		behavior:    PanelBehavior{SpacePreview: true},
		originalRow: -1,
		hoverRow:    -1,
		hoveredRow:  -1,
		language:    string(i18n.Current()),
	}
	p.view.Send(sel("setAutoresizesSubviews:"), false)

	p.background = objc.ID(panelBackdropClass).Send(sel("new"))
	p.background.Send(sel("setAutoresizingMask:"), uint64(2|16))
	p.view.Send(sel("addSubview:"), p.background)

	p.search = objc.ID(searchFieldClass).Send(sel("alloc")).Send(sel("initWithFrame:"), CGRect{})
	p.search.Send(sel("setPlaceholderString:"), nsString(i18n.T(i18n.PanelSearchPlaceholder)))
	p.search.Send(sel("setFont:"), systemFont(14))
	p.search.Send(sel("setEditable:"), true)
	p.search.Send(sel("setSelectable:"), true)
	p.search.Send(sel("setBezeled:"), false)
	p.search.Send(sel("setBordered:"), false)
	p.search.Send(sel("setDrawsBackground:"), false)
	p.search.Send(sel("setFocusRingType:"), int64(1))
	p.search.Send(sel("setLineBreakMode:"), int64(4))
	p.search.Send(sel("setUsesSingleLineMode:"), true)
	p.search.Send(sel("setTarget:"), searchTarget)
	p.search.Send(sel("setAction:"), sel("searchChanged:"))

	p.pin = newToolbarButton("pin", "pinPressed:", i18n.PanelPinWindow, false)
	p.filterButton = newToolbarButton("line.3.horizontal.decrease.circle", "filterPressed:", i18n.PanelFilter, true)
	p.settingsButton = newToolbarButton("gearshape", "settingsPressed:", i18n.SettingsWindowTitle, true)
	p.clearButton = newToolbarButton("trash", "clearPressed:", i18n.PanelClearHistory, true)
	p.shortcutButton = newToolbarButton("keyboard", "shortcutHelpPressed:", i18n.PanelShortcutsAction, true)

	p.divider = objc.ID(classNSBox).Send(sel("new"))
	p.divider.Send(sel("setBoxType:"), 2)
	p.divider.Send(sel("setTitlePosition:"), 0)

	p.empty = objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(i18n.T(i18n.PanelEmpty)))
	p.empty.Send(sel("setFont:"), systemFont(13))
	p.empty.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	p.empty.Send(sel("setAlignment:"), int64(1))
	p.empty.Send(sel("setLineBreakMode:"), int64(4))
	p.empty.Send(sel("setMaximumNumberOfLines:"), int64(1))

	p.emptyDetail = objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
	p.emptyDetail.Send(sel("setFont:"), systemFont(11))
	p.emptyDetail.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("tertiaryLabelColor")))
	p.emptyDetail.Send(sel("setAlignment:"), int64(1))
	p.emptyDetail.Send(sel("setLineBreakMode:"), int64(4))
	p.emptyDetail.Send(sel("setMaximumNumberOfLines:"), int64(1))

	p.footerDivider = objc.ID(classNSBox).Send(sel("new"))
	p.footerDivider.Send(sel("setBoxType:"), 2)
	p.footerDivider.Send(sel("setTitlePosition:"), 0)

	p.footer = objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
	p.footer.Send(sel("setFont:"), systemFont(11))
	p.footer.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("tertiaryLabelColor")))

	p.scroll = newHistoryTable(p)
	p.buildPreview()

	for _, v := range []objc.ID{p.search, p.pin, p.filterButton, p.settingsButton,
		p.clearButton, p.shortcutButton, p.divider, p.scroll, p.empty, p.emptyDetail,
		p.footerDivider, p.footer} {
		p.view.Send(sel("addSubview:"), v)
	}
	p.buildEmptyIcons()
	p.buildHoverBar()
	p.buildShortcutHelp()
	p.buildNewGroupPopover()
	p.buildNotePopover()

	panelsBySearch[p.search] = p
	panelsByControl[p.pin] = p
	panelsByControl[p.filterButton] = p
	panelsByControl[p.settingsButton] = p
	panelsByControl[p.clearButton] = p
	panelsByControl[p.shortcutButton] = p
	panelsByView[p.view] = p
	layoutPanel(p, 420, 560)

	p.observer = objc.ID(observerClass).Send(sel("new"))
	panelsByObserver[p.observer] = p
	center := objc.ID(class("NSNotificationCenter")).Send(sel("defaultCenter"))
	center.Send(sel("addObserver:selector:name:object:"), p.observer, sel("windowBecameKey:"),
		nsString(windowDidBecomeKey), objc.ID(0))
	center.Send(sel("addObserver:selector:name:object:"), p.observer, sel("windowResignedKey:"),
		nsString(windowDidResignKey), objc.ID(0))
	center.Send(sel("addObserver:selector:name:object:"), p.observer, sel("windowDidOrderOffScreen:"),
		nsString(windowDidOrderOffScreen), objc.ID(0))
	center.Send(sel("addObserver:selector:name:object:"), p.observer, sel("windowDidOrderOnScreen:"),
		nsString(windowDidOrderOnScreen), objc.ID(0))
	center.Send(sel("addObserver:selector:name:object:"), p.observer, sel("searchTextChanged:"),
		nsString("NSControlTextDidChangeNotification"), p.search)
	clip := p.scroll.Send(sel("contentView"))
	clip.Send(sel("setPostsBoundsChangedNotifications:"), true)
	center.Send(sel("addObserver:selector:name:object:"), p.observer, sel("scrollBoundsChanged:"),
		nsString("NSViewBoundsDidChangeNotification"), clip)
	monitorBlock := objc.NewBlock(func(_ objc.Block, event objc.ID) objc.ID {
		key := objc.Send[uint16](event, sel("keyCode"))
		flags := uint64(objc.Send[uint](event, sel("modifierFlags")))
		if p.handlePreviewKey(key, flags) {
			return 0
		}
		if p.handleCommandKey(key, flags) {
			return 0
		}
		return event
	})
	p.monitor = objc.ID(class("NSEvent")).Send(sel("addLocalMonitorForEventsMatchingMask:handler:"),
		uint64(1<<10), objc.ID(monitorBlock))
	monitorBlock.Release()

	// Hover is driven by mouse-moved events, not a timer: the monitor sees the
	// event even when it lands on the floating action bar over the list.
	hoverBlock := objc.NewBlock(func(_ objc.Block, event objc.ID) objc.ID {
		if objc.Send[objc.ID](event, sel("window")) == p.view.Send(sel("window")) {
			p.updateHoverAtWindowPoint(objc.Send[CGPoint](event, sel("locationInWindow")))
		}
		return event
	})
	p.hoverMonitor = objc.ID(class("NSEvent")).Send(sel("addLocalMonitorForEventsMatchingMask:handler:"),
		uint64(1<<5), objc.ID(hoverBlock))
	hoverBlock.Release()

	return p
}

// View implements core.NativeComponent.
func (p *Panel) View() any { return p.view }

// SetProp implements core.NativeComponent.
func (p *Panel) SetProp(name string, value any) {
	if set, ok := panelPropertySetters[name]; ok {
		set(p, value)
	}
}

// Command implements core.NativeComponent.
func (p *Panel) Command(name string, args any) {
	if run, ok := panelCommands[name]; ok {
		run(p)
	}
}

func (p *Panel) focusSearch() {
	p.view.Send(sel("window")).Send(sel("makeFirstResponder:"), p.search)
}

// newToolbarButton builds one icon-only toolbar action. Toolbar actions other
// than the pin use the secondary label tint so they recede behind the list.
func newToolbarButton(symbol, action string, label i18n.Key, tinted bool) objc.ID {
	button := newIconButton()
	button.Send(sel("setBordered:"), false)
	button.Send(sel("setImage:"), systemSymbol(symbol))
	if tinted {
		button.Send(sel("setContentTintColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	}
	button.Send(sel("setTarget:"), searchTarget)
	button.Send(sel("setAction:"), sel(action))
	button.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(label)))
	return button
}

func (p *Panel) clearSearch() {
	p.search.Send(sel("setStringValue:"), nsString(""))
	if p.query != "" {
		p.query = ""
		p.markSelectionFirst()
	}
}

// Close implements core.NativeComponent.
func (p *Panel) Close() {
	center := objc.ID(class("NSNotificationCenter")).Send(sel("defaultCenter"))
	center.Send(sel("removeObserver:"), p.observer)
	cancelPendingClick(p.table)
	if p.viewTracking != 0 {
		p.view.Send(sel("removeTrackingArea:"), p.viewTracking)
		p.viewTracking = 0
	}
	endPanelDrag(p)
	if p.monitor != 0 {
		objc.ID(class("NSEvent")).Send(sel("removeMonitor:"), p.monitor)
	}
	if p.hoverMonitor != 0 {
		objc.ID(class("NSEvent")).Send(sel("removeMonitor:"), p.hoverMonitor)
		p.hoverMonitor = 0
	}
	delete(panelsByObserver, p.observer)
	delete(panelsBySearch, p.search)
	delete(panelsByControl, p.pin)
	delete(panelsByControl, p.filterButton)
	delete(panelsByControl, p.settingsButton)
	delete(panelsByControl, p.clearButton)
	delete(panelsByControl, p.shortcutButton)
	for _, button := range p.hoverButtons {
		delete(panelsByControl, button)
		discardIconButton(button)
	}
	for _, button := range []objc.ID{p.pin, p.filterButton, p.settingsButton, p.clearButton, p.shortcutButton} {
		discardIconButton(button)
	}
	if p.hoverBar != 0 {
		delete(panelsByControl, p.hoverBar)
	}
	clearRowHoverStates()
	delete(panelsByView, p.view)
	delete(panelsByTable, p.table)
	delete(panelsByPreviewView, p.preview)
	delete(panelsByPopover, p.previewDelegate)
	p.closePreview()
	p.closeEmptyIcons()
	if p.shortcutPopover != 0 {
		p.shortcutPopover.Send(sel("close"))
		p.shortcutPopover.Send(sel("release"))
	}
	if p.shortcutController != 0 {
		p.shortcutController.Send(sel("release"))
	}
	if p.shortcutView != 0 {
		p.shortcutView.Send(sel("release"))
	}
	if p.groupPopover != 0 {
		delete(panelsByGroupPopover, p.groupPopover)
		p.groupPopover.Send(sel("close"))
		p.groupPopover.Send(sel("release"))
	}
	if p.groupController != 0 {
		delete(panelsByGroupController, p.groupController)
		p.groupController.Send(sel("release"))
	}
	if p.groupInput != nil {
		p.groupInput.close()
		p.groupInput = nil
	}
	if p.groupPopoverView != 0 {
		p.groupPopoverView.Send(sel("release"))
	}
	if p.notePopover != 0 {
		delete(panelsByNotePopover, p.notePopover)
		if noteActivePanel == p {
			noteActivePanel = nil
		}
		p.notePopover.Send(sel("close"))
		p.notePopover.Send(sel("release"))
	}
	if p.noteController != 0 {
		delete(panelsByNoteController, p.noteController)
		p.noteController.Send(sel("release"))
	}
	if p.noteArea != nil {
		p.noteArea.close()
		p.noteArea = nil
	}
	if p.notePopoverView != 0 {
		p.notePopoverView.Send(sel("release"))
	}
	if p.previewPopover != 0 {
		p.previewPopover.Send(sel("release"))
	}
	if p.previewController != 0 {
		p.previewController.Send(sel("release"))
	}
	if p.previewDelegate != 0 {
		p.previewDelegate.Send(sel("release"))
	}
	p.observer.Send(sel("release"))
	if p.preview != 0 {
		p.preview.Send(sel("release"))
	}
	p.divider.Send(sel("release"))
	p.scroll.Send(sel("release"))
	p.search.Send(sel("release"))
	p.pin.Send(sel("release"))
	p.filterButton.Send(sel("release"))
	p.settingsButton.Send(sel("release"))
	p.clearButton.Send(sel("release"))
	p.shortcutButton.Send(sel("release"))
	if p.nav != nil {
		p.nav.close()
		p.nav = nil
	}
	p.background.Send(sel("release"))
	p.view.Send(sel("release"))
}

func (p *Panel) togglePinned() {
	p.pinned = !p.pinned
	name := "pin"
	label := i18n.T(i18n.PanelPinWindow)
	tint := objc.ID(0)
	if p.pinned {
		name = "pin.fill"
		label = i18n.T(i18n.PanelUnpinWindow)
		tint = objc.ID(classNSColor).Send(sel("controlAccentColor"))
	}
	p.pin.Send(sel("setImage:"), systemSymbol(name))
	p.pin.Send(sel("setContentTintColor:"), tint)
	p.pin.Send(sel("setAccessibilityLabel:"), nsString(label))
	p.cx.Emit("pinChanged", p.pinned)
}

func (p *Panel) setGroupSelection(id string) {
	p.groupID = id
	p.updateStatus()
}

func (p *Panel) setFilterSelection(filter string) {
	p.filterValue = filter
	if p.nav != nil {
		p.nav.applyFilter(filter)
	}
	p.updateStatus()
}

func (p *Panel) updateStatus() {
	state := emptyState{}
	if !p.previewing {
		state = emptyStateFor(len(p.rows), p.query, p.filterValue, p.groupID)
	}
	p.setEmptyState(state)
	p.footerDivider.Send(sel("setHidden:"), false)
	if p.errText != "" {
		p.footer.Send(sel("setStringValue:"), nsString(p.errText))
		p.footer.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("systemRedColor")))
	} else if p.loading {
		p.footer.Send(sel("setStringValue:"), nsString(i18n.T(i18n.PanelLoading)))
		p.footer.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("tertiaryLabelColor")))
	} else {
		p.footer.Send(sel("setStringValue:"), nsString(i18n.Tf(i18n.PanelCount, p.total)))
		p.footer.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	}
	frame := bounds(p.view)
	layoutPanel(p, frame.Size.Width, frame.Size.Height)
}

// selectionIntent tells the next list update how to place the selection.
type selectionIntent uint8

const (
	// selectionKeep preserves the current row, or the row that took its
	// position when it is gone (deleting a row selects the next one).
	selectionKeep selectionIntent = iota
	// selectionFirst always selects the top row (switching groups, categories
	// or searches, and returning to the top on open).
	selectionFirst
)

// selectionIndex chooses the row to select once the list is replaced. rows is
// the new list, currentID the id selected before the change and previousIndex
// its old position (-1 when nothing was selected). The list always keeps one
// row selected when it is not empty.
func selectionIndex(rows []HistoryRow, currentID string, previousIndex int, intent selectionIntent) int {
	if len(rows) == 0 {
		return -1
	}
	if intent == selectionFirst {
		return 0
	}
	for index, row := range rows {
		if row.ID == currentID {
			return index
		}
	}
	if previousIndex < 0 {
		return 0
	}
	if previousIndex >= len(rows) {
		return len(rows) - 1
	}
	return previousIndex
}

// markSelectionFirst asks the next list update to select the top row.
func (p *Panel) markSelectionFirst() {
	p.selectionIntent = selectionFirst
}

// syncSelection reconciles the table selection after the rows change so the
// list never ends up without a selected row. previousID and previousIndex are
// the selection captured before the rows were replaced.
func (p *Panel) syncSelection(previousID string, previousIndex int, intent selectionIntent) {
	index := selectionIndex(p.rows, previousID, previousIndex, intent)
	if index < 0 {
		p.table.Send(sel("deselectAll:"), objc.ID(0))
		if p.selection != "" {
			p.selection = ""
			p.cx.Emit("select", "")
		}
		return
	}
	// A forced-first placement happens when the list is replaced by a new
	// category, group or search. Reveal the top row so the viewport starts at
	// the beginning instead of keeping the old scroll offset over the new
	// rows, which otherwise looks like a stuck or shifted list.
	p.selectRowIndex(index, intent == selectionFirst)
	if p.selection != previousID {
		p.cx.Emit("select", p.selection)
	}
}

// ensureSelection re-applies a valid selection after the table reloads its
// rows or row heights without the row set changing.
func (p *Panel) ensureSelection() {
	p.syncSelection(p.selection, p.selectedIndex(), selectionKeep)
}

// selectRowIndex highlights one row and records it as the selection. scroll
// reveals it when asked.
func (p *Panel) selectRowIndex(index int, scroll bool) {
	if index < 0 || index >= len(p.rows) {
		return
	}
	p.selection = p.rows[index].ID
	indexes := objc.ID(class("NSIndexSet")).Send(sel("indexSetWithIndex:"), int64(index))
	p.table.Send(sel("selectRowIndexes:byExtendingSelection:"), indexes, false)
	if scroll {
		p.table.Send(sel("scrollRowToVisible:"), int64(index))
	}
}

func (p *Panel) selectRow(index int) {
	// An explicit selection (click, keyboard navigation) wins over a pending
	// automatic placement such as the return-to-top reset from opening.
	p.selectionIntent = selectionKeep
	p.selectRowIndex(index, true)
}

// selectRowClamped keeps a computed target inside the list instead of
// silently doing nothing at the edges, so Page Up/Down and single-step
// movement stop on the first or last row.
func (p *Panel) selectRowClamped(index int) {
	if len(p.rows) == 0 {
		return
	}
	if index < 0 {
		index = 0
	}
	if index >= len(p.rows) {
		index = len(p.rows) - 1
	}
	p.selectRow(index)
}

// nextRowIndex returns the row reached by moving delta rows from current,
// clamped to the list bounds. With nothing selected it activates the first
// row, matching the legacy active item.
func nextRowIndex(current, delta, count int) int {
	if count <= 0 {
		return -1
	}
	if current < 0 {
		return 0
	}
	next := current + delta
	if next < 0 {
		return 0
	}
	if next >= count {
		return count - 1
	}
	return next
}

// moveSelection moves one row, clamping to the list bounds.
func (p *Panel) moveSelection(delta int) {
	if len(p.rows) == 0 {
		return
	}
	p.selectRow(nextRowIndex(p.selectedIndex(), delta, len(p.rows)))
}

// applyNavigation performs a navigation action shared by the table key
// handler and the preview event monitor.
func (p *Panel) applyNavigation(action string) {
	switch action {
	case "firstRow":
		p.selectRowClamped(0)
	case "lastRow":
		p.selectRowClamped(len(p.rows) - 1)
	case "pageUp", "pageDown":
		current := p.selectedIndex()
		if current < 0 {
			current = 0
		}
		delta := p.pageSize()
		if action == "pageUp" {
			delta = -delta
		}
		p.selectRowClamped(current + delta)
	case "previousRow":
		p.moveSelection(-1)
	case "nextRow":
		p.moveSelection(1)
	}
}

func (p *Panel) selectedIndex() int {
	for index, row := range p.rows {
		if row.ID == p.selection {
			return index
		}
	}
	return -1
}

func (p *Panel) pageSize() int {
	visible := objc.Send[CGRect](p.table, sel("visibleRect")).Size.Height
	if visible <= 0 || len(p.rowHeights) == 0 {
		return 1
	}
	spacing := objc.Send[CGSize](p.table, sel("intercellSpacing")).Height
	total := 0.0
	size := 0
	for _, height := range p.rowHeights {
		if total+height > visible {
			break
		}
		total += height + spacing
		size++
	}
	if size < 1 {
		return 1
	}
	return size
}

// checkLoadMore asks for the next page when the last visible row nears the
// end of the loaded rows. It runs when the list's clip view reports a bounds
// change (scroll, resize or programmatic scroll), so paging follows the user
// instead of a timer.
func (p *Panel) checkLoadMore() {
	if p.loading || len(p.rows) == 0 {
		return
	}
	window := p.view.Send(sel("window"))
	if window == 0 || !objc.Send[bool](window, sel("isVisible")) {
		return
	}
	visible := objc.Send[CGRect](p.table, sel("visibleRect"))
	if visible.Size.Height <= 0 {
		return
	}
	point := CGPoint{
		X: visible.Origin.X + 1,
		Y: visible.Origin.Y + visible.Size.Height - 1,
	}
	row := int(objc.Send[int64](p.table, sel("rowAtPoint:"), point))
	if needsMoreRows(row, len(p.rows)) {
		p.cx.Emit("loadMore", nil)
	}
}

func needsMoreRows(row, count int) bool {
	return row >= 0 && count > 0 && row >= count-8
}

func int64Value(value any) (int64, bool) {
	switch n := value.(type) {
	case int64:
		return n, true
	case int:
		return int64(n), true
	case float64:
		return int64(n), true
	}
	return 0, false
}
