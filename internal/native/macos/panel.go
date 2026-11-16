//go:build darwin

package macos

import (
	"sync"
	"sync/atomic"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
	"github.com/nsgo-dev/nsgo"
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
	divider            objc.ID
	scroll             objc.ID
	table              objc.ID
	empty              objc.ID
	emptyIcon          objc.ID
	emptyKubai         objc.ID
	emptyKubaiIcon     *nsgo.View
	emptyDetail        objc.ID
	footerDivider      objc.ID
	footer             objc.ID
	preview            objc.ID
	previewPopover     objc.ID
	previewController  objc.ID
	previewDelegate    objc.ID
	observer           objc.ID
	timer              objc.ID
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
	total              int64
	loading            bool
	errText            string
	selection          string
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
	searchCellClass   objc.Class
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
	cell := objc.ID(searchCellClass).Send(sel("alloc")).Send(sel("initTextCell:"), nsString(""))
	p.search.Send(sel("setCell:"), cell)
	cell.Send(sel("release"))
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

	p.emptyIcon = objc.ID(class("NSImageView")).Send(sel("new"))
	p.emptyIcon.Send(sel("setImageScaling:"), int64(3))
	p.emptyIcon.Send(sel("setImageAlignment:"), int64(5))
	p.emptyIcon.Send(sel("setContentTintColor:"), objc.ID(classNSColor).Send(sel("tertiaryLabelColor")))

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
		p.clearButton, p.shortcutButton, p.divider, p.scroll, p.empty, p.emptyIcon, p.emptyDetail,
		p.footerDivider, p.footer} {
		p.view.Send(sel("addSubview:"), v)
	}
	p.buildHoverBar()
	p.buildShortcutHelp()

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
	p.timer = objc.ID(class("NSTimer")).Send(sel("scheduledTimerWithTimeInterval:target:selector:userInfo:repeats:"),
		0.1, p.observer, sel("pollSearch:"), objc.ID(0), true)
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
	p.query = ""
}

// Close implements core.NativeComponent.
func (p *Panel) Close() {
	center := objc.ID(class("NSNotificationCenter")).Send(sel("defaultCenter"))
	center.Send(sel("removeObserver:"), p.observer)
	cancelPendingClick(p.table)
	p.timer.Send(sel("invalidate"))
	endPanelDrag(p)
	if p.monitor != 0 {
		objc.ID(class("NSEvent")).Send(sel("removeMonitor:"), p.monitor)
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
	}
	if p.hoverBar != 0 {
		delete(panelsByControl, p.hoverBar)
	}
	delete(panelsByView, p.view)
	delete(panelsByTable, p.table)
	delete(panelsByPreviewView, p.preview)
	delete(panelsByPopover, p.previewDelegate)
	p.closePreview()
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
}

func (p *Panel) setFilterSelection(filter string) {
	p.filterValue = filter
	if p.nav != nil {
		p.nav.applyFilter(filter)
	}
}

func (p *Panel) updateStatus() {
	state := emptyState{}
	if !p.previewing {
		state = emptyStateFor(p.errText, len(p.rows), p.query, p.loading)
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

func (p *Panel) restoreSelection() {
	if p.selection == "" {
		return
	}
	for i, row := range p.rows {
		if row.ID == p.selection {
			indexes := objc.ID(class("NSIndexSet")).Send(sel("indexSetWithIndex:"), int64(i))
			p.table.Send(sel("selectRowIndexes:byExtendingSelection:"), indexes, false)
			return
		}
	}
	p.selection = ""
}

func (p *Panel) selectRow(index int) {
	if index < 0 || index >= len(p.rows) {
		return
	}
	p.selection = p.rows[index].ID
	indexes := objc.ID(class("NSIndexSet")).Send(sel("indexSetWithIndex:"), int64(index))
	p.table.Send(sel("selectRowIndexes:byExtendingSelection:"), indexes, false)
	p.table.Send(sel("scrollRowToVisible:"), int64(index))
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
	height := objc.Send[float64](p.table, sel("rowHeight"))
	if height <= 0 {
		height = rowHeight
	}
	visible := objc.Send[CGRect](p.table, sel("visibleRect"))
	size := int(visible.Size.Height / height)
	if size < 1 {
		return 1
	}
	return size
}

// pollLoadMore asks for the next page when the last visible row nears the
// end of the loaded rows. NSTableView's willDisplayCell callback is not
// reliable for programmatic selection and native scroll gestures, so the
// panel checks its visible row on the existing poll.
func (p *Panel) pollLoadMore() {
	if p.loading || len(p.rows) == 0 {
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
