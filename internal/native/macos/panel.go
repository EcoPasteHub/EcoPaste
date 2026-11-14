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
	cx                            *core.NativeContext
	behavior                      PanelBehavior
	view                          objc.ID
	search                        objc.ID
	pin                           objc.ID
	filter                        objc.ID
	groupsPopup                   objc.ID
	groupActions                  objc.ID
	divider                       objc.ID
	scroll                        objc.ID
	table                         objc.ID
	empty                         objc.ID
	emptyIcon                     objc.ID
	emptyDetail                   objc.ID
	footerDivider                 objc.ID
	footer                        objc.ID
	hint                          objc.ID
	preview                       objc.ID
	previewPopover                objc.ID
	previewController             objc.ID
	previewDelegate               objc.ID
	observer                      objc.ID
	timer                         objc.ID
	monitor                       objc.ID
	previewTitle                  objc.ID
	previewMeta                   objc.ID
	previewDivider                objc.ID
	previewBackdrop               objc.ID
	previewText                   objc.ID
	previewScroll                 objc.ID
	previewImage                  objc.ID
	previewImageSize              CGSize
	onboardingData                PanelOnboarding
	onboardingActive              bool
	onboardingStep                int
	onboardingView                objc.ID
	onboardingIcon                objc.ID
	onboardingHeaderDivider       objc.ID
	onboardingFooterDivider       objc.ID
	onboardingTitle               objc.ID
	onboardingDetail              objc.ID
	onboardingBody                objc.ID
	onboardingProgress            objc.ID
	onboardingProgressBar         objc.ID
	onboardingAccessibility       objc.ID
	onboardingFullDisk            objc.ID
	onboardingAccessibilityButton objc.ID
	onboardingFullDiskButton      objc.ID
	onboardingShortcutOpen        objc.ID
	onboardingShortcutPreference  objc.ID
	onboardingLegacyNormal        objc.ID
	onboardingLegacyFavorite      objc.ID
	onboardingLegacyNormalOn      bool
	onboardingLegacyFavoriteOn    bool
	onboardingNote                objc.ID
	onboardingBack                objc.ID
	onboardingExtra               objc.ID
	onboardingNext                objc.ID
	drag                          func(string) (DragPayload, bool)
	dragFiles                     []string
	actions                       []string
	hoverRow                      int
	hoveredRow                    int
	hoverBar                      objc.ID
	hoverButtons                  []objc.ID
	hoverSpecs                    []hoverAction
	hoverBarWidth                 float64
	originalRow                   int
	query                         string
	rows                          []HistoryRow
	total                         int64
	loading                       bool
	errText                       string
	selection                     string
	pendingClick                  string
	previewing                    bool
	filterValue                   string
	groupID                       string
	groups                        []GroupOption
	display                       PanelDisplay
	pinned                        bool
	dragging                      bool
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
	}
	p.view.Send(sel("setAutoresizesSubviews:"), false)

	p.search = objc.ID(classNSSearchField).Send(sel("new"))
	p.search.Send(sel("setPlaceholderString:"), nsString(i18n.T(i18n.PanelSearchPlaceholder)))
	p.search.Send(sel("setFont:"), systemFont(13))
	p.search.Send(sel("setSendsSearchStringImmediately:"), true)
	p.search.Send(sel("setSendsWholeSearchString:"), false)
	p.search.Send(sel("setTarget:"), searchTarget)
	p.search.Send(sel("setAction:"), sel("searchChanged:"))

	p.pin = objc.ID(class("NSButton")).Send(sel("alloc")).Send(sel("initWithFrame:"), CGRect{})
	p.pin.Send(sel("setBordered:"), false)
	p.pin.Send(sel("setImage:"), systemSymbol("pin"))
	p.pin.Send(sel("setTarget:"), searchTarget)
	p.pin.Send(sel("setAction:"), sel("pinPressed:"))
	p.pin.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.PanelPinWindow)))

	p.filter = objc.ID(class("NSSegmentedControl")).Send(sel("alloc")).
		Send(sel("initWithFrame:"), CGRect{})
	p.filter.Send(sel("setSegmentCount:"), int64(5))
	for index, label := range []string{
		i18n.T(i18n.PanelAll), i18n.T(i18n.PanelFavorites), i18n.T(i18n.PanelTextCategory),
		i18n.T(i18n.PanelImageCategory), i18n.T(i18n.PanelFilesCategory),
	} {
		p.filter.Send(sel("setLabel:forSegment:"), nsString(label), int64(index))
	}
	p.filter.Send(sel("setTrackingMode:"), int64(0))
	p.filter.Send(sel("setSelectedSegment:"), int64(0))
	p.filter.Send(sel("setTarget:"), searchTarget)
	p.filter.Send(sel("setAction:"), sel("filterChanged:"))
	p.filter.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.PanelCategoryLabel)))

	p.groupsPopup = objc.ID(class("NSPopUpButton")).Send(sel("alloc")).
		Send(sel("initWithFrame:pullsDown:"), CGRect{}, false)
	p.groupsPopup.Send(sel("setTarget:"), searchTarget)
	p.groupsPopup.Send(sel("setAction:"), sel("groupChanged:"))
	p.groupsPopup.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.PanelGroupsLabel)))
	p.rebuildGroupMenu()

	p.groupActions = objc.ID(class("NSButton")).Send(sel("alloc")).Send(sel("initWithFrame:"), CGRect{})
	p.groupActions.Send(sel("setBordered:"), false)
	p.groupActions.Send(sel("setImage:"), systemSymbol("ellipsis.circle"))
	p.groupActions.Send(sel("setTarget:"), searchTarget)
	p.groupActions.Send(sel("setAction:"), sel("groupActions:"))
	p.groupActions.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.PanelGroupActions)))

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

	p.hint = objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(i18n.T(i18n.PanelHint)))
	p.hint.Send(sel("setFont:"), systemFont(11))
	p.hint.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("tertiaryLabelColor")))
	p.hint.Send(sel("setAlignment:"), int64(2))
	p.hint.Send(sel("setLineBreakMode:"), int64(4))
	p.hint.Send(sel("setMaximumNumberOfLines:"), int64(1))

	p.scroll = newHistoryTable(p)
	p.buildPreview()

	for _, v := range []objc.ID{p.search, p.pin, p.filter, p.groupsPopup, p.groupActions,
		p.divider, p.scroll, p.empty, p.emptyIcon, p.emptyDetail, p.footerDivider, p.footer, p.hint} {
		p.view.Send(sel("addSubview:"), v)
	}
	p.buildHoverBar()
	p.buildOnboarding()

	panelsBySearch[p.search] = p
	panelsByControl[p.pin] = p
	panelsByControl[p.filter] = p
	panelsByControl[p.groupsPopup] = p
	panelsByControl[p.groupActions] = p
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
		return event
	})
	p.monitor = objc.ID(class("NSEvent")).Send(sel("addLocalMonitorForEventsMatchingMask:handler:"),
		uint64(1<<10), objc.ID(monitorBlock))
	monitorBlock.Release()

	return p
}

func registerPanelTargets() {
	var err error
	onboardingViewClass, err = objc.RegisterClass("EcoPasteOnboardingView", class("NSVisualEffectView"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("acceptsFirstResponder"), Fn: func(_ objc.ID, _ objc.SEL) bool { return true }},
			{Cmd: sel("keyDown:"), Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
				p := panelsByOnboarding[self]
				if p == nil {
					return
				}
				switch objc.Send[uint16](event, sel("keyCode")) {
				case keyReturn, keyEnter:
					p.onboardingAdvance()
				}
			}},
		})
	if err != nil {
		panic(err)
	}
	panelViewClass, err = objc.RegisterClass("EcoPastePanelView", class("NSView"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("viewDidMoveToWindow"), Fn: func(self objc.ID, _ objc.SEL) {
				sendSuper[objc.ID](self, panelViewClass, sel("viewDidMoveToWindow"))
				if p := panelsByView[self]; p != nil {
					applyPlacement(p)
				}
			}},
			{Cmd: sel("setFrameSize:"), Fn: func(self objc.ID, _ objc.SEL, size CGSize) {
				sendSuper[objc.ID](self, panelViewClass, sel("setFrameSize:"), size)
				if p := panelsByView[self]; p != nil {
					layoutPanel(p, size.Width, size.Height)
				}
			}},
			{Cmd: sel("keyDown:"), Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
				p := panelsByView[self]
				if p != nil && p.previewing && objc.Send[uint16](event, sel("keyCode")) == 53 {
					p.cx.Emit("hidePreview", nil)
					return
				}
				sendSuper[objc.ID](self, panelViewClass, sel("keyDown:"), event)
			}},
			{Cmd: sel("cancelOperation:"), Fn: func(self objc.ID, _ objc.SEL, sender objc.ID) {
				p := panelsByView[self]
				if p != nil && p.previewing {
					p.cx.Emit("hidePreview", nil)
					return
				}
				if p != nil {
					p.cx.Emit("hide", nil)
				}
			}},
		})
	if err != nil {
		panic(err)
	}

	target, err := objc.RegisterClass("EcoPasteSearchTarget", class("NSObject"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("searchChanged:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
				if p := panelsBySearch[sender]; p != nil {
					p.query = goString(sender.Send(sel("stringValue")))
					p.cx.Emit("query", p.query)
				}
			}},
			{Cmd: sel("pinPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
				if p := panelsByControl[sender]; p != nil {
					p.togglePinned()
				}
			}},
			{Cmd: sel("filterChanged:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
				p := panelsByControl[sender]
				if p == nil {
					return
				}
				index := int(objc.Send[int64](sender, sel("selectedSegment")))
				filters := []string{"", "favorite", "text", "image", "files"}
				if index >= 0 && index < len(filters) {
					p.cx.Emit("filter", filters[index])
				}
			}},
			{Cmd: sel("groupChanged:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
				p := panelsByControl[sender]
				if p == nil {
					return
				}
				index := int(objc.Send[int64](sender, sel("indexOfSelectedItem")))
				groupID := ""
				if index > 0 && index-1 < len(p.groups) {
					groupID = p.groups[index-1].ID
				}
				p.cx.Emit("group", groupID)
			}},
			{Cmd: sel("groupActions:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
				p := panelsByControl[sender]
				if p == nil {
					return
				}
				p.showGroupActions(sender)
			}},
			{Cmd: sel("hoverActionPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
				if p := panelsByControl[sender]; p != nil {
					p.runHoverAction(int(objc.Send[int64](sender, sel("tag"))))
				}
			}},
			{Cmd: sel("onboardingPrevPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
				if p := panelsByControl[sender]; p != nil {
					p.onboardingPrev()
				}
			}},
			{Cmd: sel("onboardingNextPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
				if p := panelsByControl[sender]; p != nil {
					p.onboardingAdvance()
				}
			}},
			{Cmd: sel("onboardingExtraPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
				p := panelsByControl[sender]
				if p == nil {
					return
				}
				switch p.onboardingStep {
				case 2, 3:
					p.cx.Emit("openSettings", onboardingSettingsTab(p.onboardingStep))
				case onboardingLegacyStep:
					p.onboardingSkipLegacy()
				}
			}},
			{Cmd: sel("onboardingLegacyNormalPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
				p := panelsByControl[sender]
				if p == nil {
					return
				}
				p.onboardingLegacyNormalOn = objc.Send[int64](sender, sel("state")) == 1
				p.updateLegacyNext()
			}},
			{Cmd: sel("onboardingLegacyFavoritePressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
				p := panelsByControl[sender]
				if p == nil {
					return
				}
				p.onboardingLegacyFavoriteOn = objc.Send[int64](sender, sel("state")) == 1
				p.updateLegacyNext()
			}},
			{Cmd: sel("onboardingPermissionPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
				if objc.Send[int64](sender, sel("tag")) == 0 {
					OpenAccessibilitySettings()
					return
				}
				OpenFullDiskAccessSettings()
			}},
			{Cmd: sel("menuNewGroup:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
				if p := menuPanel(sender); p != nil {
					p.cx.Emit("newGroup", nil)
				}
			}},
			{Cmd: sel("menuRenameGroup:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
				if p := menuPanel(sender); p != nil && p.groupID != "" {
					p.cx.Emit("renameGroup", p.groupID)
				}
			}},
			{Cmd: sel("menuDeleteGroup:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
				if p := menuPanel(sender); p != nil && p.groupID != "" {
					p.cx.Emit("deleteGroup", p.groupID)
				}
			}},
		})
	if err != nil {
		panic(err)
	}
	searchTarget = objc.ID(target).Send(sel("new"))

	observerClass, err = objc.RegisterClass("EcoPastePanelObserver", class("NSObject"), nil, nil,
		panelObserverMethods())
	if err != nil {
		panic(err)
	}
	registerHistoryTargets()
	registerPreviewTargets()
}

// View implements core.NativeComponent.
func (p *Panel) View() any { return p.view }

// SetProp implements core.NativeComponent.
func (p *Panel) SetProp(name string, value any) {
	switch name {
	case "query":
		if s, ok := value.(string); ok && s != p.query {
			p.query = s
			p.search.Send(sel("setStringValue:"), nsString(s))
		}
	case "placeholder":
		if s, ok := value.(string); ok {
			p.search.Send(sel("setPlaceholderString:"), nsString(s))
		}
	case "rows":
		rows, _ := value.([]HistoryRow)
		p.hideHover()
		p.rows = rows
		p.table.Send(sel("reloadData"))
		p.restoreSelection()
		p.updateStatus()
	case "total":
		if n, ok := int64Value(value); ok {
			p.total = n
			p.updateStatus()
		}
	case "loading":
		if b, ok := value.(bool); ok {
			p.loading = b
			p.updateStatus()
		}
	case "error":
		if s, ok := value.(string); ok {
			p.errText = s
			p.updateStatus()
		}
	case "preview":
		if content, ok := value.(PreviewContent); ok {
			p.setPreview(content)
		}
	case "groups":
		if groups, ok := value.([]GroupOption); ok {
			p.groups = groups
			p.rebuildGroupMenu()
			frame := bounds(p.view)
			if frame.Size.Width > 0 {
				layoutPanel(p, frame.Size.Width, frame.Size.Height)
			}
		}
	case "group":
		if id, ok := value.(string); ok {
			p.setGroupSelection(id)
		}
	case "filter":
		if filter, ok := value.(string); ok {
			p.setFilterSelection(filter)
		}
	case "behavior":
		if behavior, ok := value.(PanelBehavior); ok {
			p.behavior = behavior
		}
	case "drag":
		if provider, ok := value.(func(string) (DragPayload, bool)); ok {
			p.drag = provider
		}
	case "display":
		if display, ok := value.(PanelDisplay); ok {
			p.display = display
			p.table.Send(sel("setRowHeight:"), tableRowHeight(display))
			p.table.Send(sel("reloadData"))
		}
	case "actions":
		if actions, ok := value.([]string); ok {
			p.actions = actions
			p.hideHover()
		}
	case "onboarding":
		if data, ok := value.(PanelOnboarding); ok {
			p.setOnboarding(data)
		}
	}
}

// Command implements core.NativeComponent.
func (p *Panel) Command(name string, args any) {
	switch name {
	case "focusSearch":
		p.focusSearch()
	case "clearSearch":
		p.clearSearch()
	case "reload":
		p.table.Send(sel("reloadData"))
	case "scrollTop":
		if len(p.rows) > 0 {
			p.table.Send(sel("scrollRowToVisible:"), int64(0))
		}
	case "showAccessibilityAlert":
		showAccessibilityAlert()
	case "toggleWindowPinned":
		p.togglePinned()
	case "resignKey":
		p.view.Send(sel("window")).Send(sel("resignKeyWindow"))
	case "makeKey":
		p.view.Send(sel("window")).Send(sel("makeKeyWindow"))
	}
}

func (p *Panel) focusSearch() {
	p.view.Send(sel("window")).Send(sel("makeFirstResponder:"), p.search)
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
	if p.monitor != 0 {
		objc.ID(class("NSEvent")).Send(sel("removeMonitor:"), p.monitor)
	}
	delete(panelsByObserver, p.observer)
	delete(panelsBySearch, p.search)
	delete(panelsByControl, p.pin)
	delete(panelsByControl, p.filter)
	delete(panelsByControl, p.groupsPopup)
	delete(panelsByControl, p.groupActions)
	for _, button := range p.hoverButtons {
		delete(panelsByControl, button)
	}
	for _, button := range []objc.ID{
		p.onboardingAccessibilityButton, p.onboardingFullDiskButton,
		p.onboardingLegacyNormal, p.onboardingLegacyFavorite,
		p.onboardingBack, p.onboardingExtra, p.onboardingNext,
	} {
		delete(panelsByControl, button)
	}
	if p.hoverBar != 0 {
		delete(panelsByControl, p.hoverBar)
	}
	delete(panelsByView, p.view)
	delete(panelsByOnboarding, p.onboardingView)
	delete(panelsByTable, p.table)
	delete(panelsByPreviewView, p.preview)
	delete(panelsByPopover, p.previewDelegate)
	p.closePreview()
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
	p.filter.Send(sel("release"))
	p.groupsPopup.Send(sel("release"))
	p.groupActions.Send(sel("release"))
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

func (p *Panel) rebuildGroupMenu() {
	if p.groupsPopup == 0 {
		return
	}
	p.groupsPopup.Send(sel("removeAllItems"))
	p.groupsPopup.Send(sel("addItemWithTitle:"), nsString(i18n.T(i18n.PanelAllGroups)))
	for _, group := range p.groups {
		p.groupsPopup.Send(sel("addItemWithTitle:"), nsString(group.Name))
	}
	p.setGroupSelection(p.groupID)
}

func (p *Panel) setGroupSelection(id string) {
	p.groupID = id
	index := int64(0)
	for i, group := range p.groups {
		if group.ID == id {
			index = int64(i + 1)
			break
		}
	}
	p.groupsPopup.Send(sel("selectItemAtIndex:"), index)
}

func (p *Panel) setFilterSelection(filter string) {
	p.filterValue = filter
	index := int64(0)
	for i, value := range []string{"", "favorite", "text", "image", "files"} {
		if value == filter {
			index = int64(i)
			break
		}
	}
	p.filter.Send(sel("setSelectedSegment:"), index)
}

func (p *Panel) showGroupActions(anchor objc.ID) {
	menu := objc.ID(class("NSMenu")).Send(sel("new"))
	defer menu.Send(sel("release"))
	menu.Send(sel("setAutoenablesItems:"), false)
	addGroupAction(menu, p, i18n.T(i18n.PanelNewGroup), "menuNewGroup:", true)
	addGroupAction(menu, p, i18n.T(i18n.PanelRenameGroup), "menuRenameGroup:", p.groupID != "")
	addGroupAction(menu, p, i18n.T(i18n.PanelDeleteGroup), "menuDeleteGroup:", p.groupID != "")
	menu.Send(sel("popUpMenuPositioningItem:atLocation:inView:"), objc.ID(0),
		CGPoint{X: 0, Y: bounds(anchor).Size.Height}, anchor)
}

func addGroupAction(menu objc.ID, p *Panel, title, action string, enabled bool) {
	item := objc.ID(class("NSMenuItem")).Send(sel("alloc")).
		Send(sel("initWithTitle:action:keyEquivalent:"), nsString(title), sel(action), nsString(""))
	item.Send(sel("setTarget:"), searchTarget)
	item.Send(sel("setRepresentedObject:"), p.table)
	item.Send(sel("setEnabled:"), enabled)
	menu.Send(sel("addItem:"), item)
	item.Send(sel("release"))
}

func (p *Panel) updateStatus() {
	if p.onboardingActive {
		p.hint.Send(sel("setHidden:"), true)
		p.footer.Send(sel("setHidden:"), true)
		p.footerDivider.Send(sel("setHidden:"), true)
		p.setEmptyState(emptyState{})
		return
	}
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

func (p *Panel) setEmptyState(state emptyState) {
	visible := state.Title != ""
	p.empty.Send(sel("setStringValue:"), nsString(state.Title))
	p.empty.Send(sel("setHidden:"), !visible)
	p.emptyDetail.Send(sel("setStringValue:"), nsString(state.Detail))
	p.emptyDetail.Send(sel("setHidden:"), !visible || state.Detail == "")
	p.emptyIcon.Send(sel("setHidden:"), !visible)
	if !visible {
		return
	}
	color := objc.ID(classNSColor).Send(sel("secondaryLabelColor"))
	iconColor := objc.ID(classNSColor).Send(sel("tertiaryLabelColor"))
	if state.Error {
		color = objc.ID(classNSColor).Send(sel("systemRedColor"))
		iconColor = color
	}
	p.empty.Send(sel("setTextColor:"), color)
	p.emptyIcon.Send(sel("setImage:"), systemSymbol(state.Symbol))
	p.emptyIcon.Send(sel("setContentTintColor:"), iconColor)
}

type emptyState struct {
	Symbol string
	Title  string
	Detail string
	Error  bool
}

func emptyStateFor(errText string, rows int, query string, loading bool) emptyState {
	switch {
	case rows > 0:
		return emptyState{}
	case errText != "":
		return emptyState{Symbol: "exclamationmark.triangle", Title: errText, Error: true}
	case loading:
		return emptyState{Symbol: "clock", Title: i18n.T(i18n.PanelLoading)}
	case query != "":
		return emptyState{
			Symbol: "magnifyingglass", Title: i18n.T(i18n.PanelNoMatches),
			Detail: i18n.T(i18n.PanelNoMatchesDetail),
		}
	default:
		return emptyState{
			Symbol: "doc.on.clipboard", Title: i18n.T(i18n.PanelEmpty),
			Detail: i18n.T(i18n.PanelEmptyDetail),
		}
	}
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
