//go:build darwin

package macos

import (
	"fmt"
	"sync"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
	"github.com/nsgo-dev/nsgo/core"
)

// SettingsComponent is the native settings window's content view.
const SettingsComponent = "ecopaste.settings"

// SettingsValues is the view state of the settings window.
type SettingsValues struct {
	DockIcon               bool
	TrayIcon               bool
	AutoStart              bool
	Theme                  string
	Language               string
	DataDir                string
	CanRestoreDataDir      bool
	CopyPlain              bool
	CopyThenHideWindow     bool
	CopySound              bool
	PastePlain             bool
	UpdateOnReuse          bool
	AutoPaste              string
	MiddleClick            string
	PasteFilesAsPath       bool
	AutoFavorite           bool
	ShowOriginalPreview    bool
	DeleteConfirm          bool
	DeleteFavoriteConfirm  bool
	DeletePinnedConfirm    bool
	DeleteFavoriteItems    bool
	DeletePinnedItems      bool
	DeleteFavoriteOnlyFav  bool
	Sort                   string
	WindowPosition         string
	CaptureText            bool
	CaptureHTML            bool
	CaptureRTF             bool
	CaptureImage           bool
	CaptureFiles           bool
	CaptureOrder           []string
	MaxTextMB              uint32
	MaxImageMB             uint32
	TextMaxLines           uint32
	ImageMaxHeight         uint32
	FileMaxCount           uint32
	ScrollToTopOnOpen      bool
	RangeOnOpen            string
	CategoryOnOpen         string
	GroupOnOpen            string
	SearchFocus            bool
	SearchClear            bool
	Groups                 []SettingsGroup
	Retention              string
	MaxCount               uint32
	CleanupIntervalHours   uint32
	CollectSecrets         bool
	RedactSecrets          bool
	ExcludedAppIDs         []string
	Apps                   []SettingsApp
	ItemActions            []string
	ItemActionOrder        []string
	OpenClipboardShortcut  string
	OpenPreferenceShortcut string
	SpacePreview           bool
}

// SettingsApp is one row in the source application filter list.
type SettingsApp struct {
	ID       string
	Name     string
	IconPath string
	Excluded bool
}

// SettingsGroup is one clipboard group offered by the open-selection
// popup.
type SettingsGroup struct {
	ID   string
	Name string
}

// SettingsChange is one native control's new value.
type SettingsChange struct {
	Path  string
	Value any
}

// SettingsView is the AppKit settings window content.
type SettingsView struct {
	cx                    *core.NativeContext
	view                  objc.ID
	tabs                  objc.ID
	sidebar               objc.ID
	sidebarTable          objc.ID
	sectionTitle          objc.ID
	footer                objc.ID
	searchField           objc.ID
	searchResults         objc.ID
	dataDir               objc.ID
	change                objc.ID
	restore               objc.ID
	reset                 objc.ID
	accessibilityStatus   objc.ID
	fullDiskStatus        objc.ID
	diagnosticsDir        objc.ID
	diagnosticsHint       objc.ID
	permissionTimer       objc.ID
	appsTable             objc.ID
	appsEmpty             objc.ID
	apps                  []SettingsApp
	actionsTable          objc.ID
	actionUp              objc.ID
	actionDown            objc.ID
	actions               []settingsAction
	captureTable          objc.ID
	captureUp             objc.ID
	captureDown           objc.ID
	captureOrder          []string
	displayPreviewBox     objc.ID
	displayPreviewDivider objc.ID
	displayPreviewRows    []objc.ID
	groupField            *settingsField
	extraExcluded         []string
	shortcutFields        []*shortcutControl
	lastValues            SettingsValues
	fields                []*settingsField
	currentTab            string
	currentTabTitle       string
	searchEntries         []settingsSearchEntry
	searchMatches         []settingsSearchEntry
	searchHighlight       objc.ID
	searchHighlightTimer  objc.ID
}

type settingsField struct {
	path       string
	kind       string
	control    objc.ID
	valueLabel objc.ID
	options    []string
	titles     []string
	suffix     string
	get        func(SettingsValues) any
}

const (
	settingsWidth    = 720.0
	settingsHeight   = 560.0
	settingsSidebarW = 200.0
	settingsFooterH  = 32.0
	settingsTabAreaH = 470.0
	tabContentW      = 520.0
	tabContentH      = 470.0
)

var settingsTabTitleKeys = map[string]i18n.Key{
	"general":     i18n.SettingsTabGeneral,
	"data":        i18n.SettingsTabData,
	"capture":     i18n.SettingsTabCapture,
	"display":     i18n.SettingsTabDisplay,
	"window":      i18n.SettingsTabWindow,
	"history":     i18n.SettingsTabHistory,
	"permissions": i18n.SettingsTabPermissions,
	"apps":        i18n.SettingsTabApps,
	"actions":     i18n.SettingsTabActions,
	"interaction": i18n.SettingsTabInteraction,
	"deletion":    i18n.SettingsTabDeletion,
	"shortcuts":   i18n.SettingsTabShortcuts,
	"preview":     i18n.SettingsTabPreview,
	"update":      i18n.SettingsTabUpdate,
}

func settingsTabTitle(name string) string {
	if key, ok := settingsTabTitleKeys[name]; ok {
		return i18n.T(key)
	}
	return name
}

var (
	registerSettingsOnce sync.Once
	settingsTarget       objc.ID
	settingsViews        = map[objc.ID]*SettingsView{}
	settingsTimers       = map[objc.ID]*SettingsView{}
	settingsTabMu        sync.Mutex
	settingsOpenViews    = map[*SettingsView]struct{}{}
	pendingSettingsTab   string
)

var settingsTabIdentifiers = map[string]struct{}{
	"general":     {},
	"data":        {},
	"capture":     {},
	"display":     {},
	"window":      {},
	"history":     {},
	"permissions": {},
	"apps":        {},
	"actions":     {},
	"interaction": {},
	"deletion":    {},
	"shortcuts":   {},
	"preview":     {},
	"update":      {},
}

// SelectSettingsTab shows the named tab when the settings window is open,
// or on the next open otherwise. It reports whether the name is known.
func SelectSettingsTab(name string) bool {
	if _, ok := settingsTabIdentifiers[name]; !ok {
		return false
	}
	settingsTabMu.Lock()
	if len(settingsOpenViews) == 0 {
		pendingSettingsTab = name
		settingsTabMu.Unlock()
		return true
	}
	views := make([]*SettingsView, 0, len(settingsOpenViews))
	for view := range settingsOpenViews {
		views = append(views, view)
	}
	settingsTabMu.Unlock()
	for _, view := range views {
		view.selectTab(name)
	}
	return true
}

// NewSettings builds the native settings controls.
func NewSettings(cx *core.NativeContext) core.NativeComponent {
	registerSettingsOnce.Do(registerSettingsTargets)
	v := &SettingsView{
		cx:   cx,
		view: objc.ID(class("NSView")).Send(sel("new")),
	}
	v.view.Send(sel("setAutoresizesSubviews:"), false)

	v.tabs = objc.ID(class("NSTabView")).Send(sel("new"))
	v.tabs.Send(sel("setTabViewType:"), int64(6))
	v.tabs.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsSidebarW, Y: settingsFooterH},
		Size:   CGSize{Width: settingsWidth - settingsSidebarW, Height: settingsTabAreaH},
	})
	v.tabs.Send(sel("setAutoresizingMask:"), int64(18))
	v.view.Send(sel("addSubview:"), v.tabs)

	v.footer = objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(i18n.T(i18n.SettingsFooter)))
	v.footer.Send(sel("setFont:"), systemFont(11))
	v.footer.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("tertiaryLabelColor")))
	v.footer.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsSidebarW + 20, Y: 8},
		Size:   CGSize{Width: settingsWidth - settingsSidebarW - 200, Height: 16},
	})
	v.view.Send(sel("addSubview:"), v.footer)

	v.reset = objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(i18n.T(i18n.SettingsReset)), settingsTarget, sel("resetSettingsPressed:"))
	v.reset.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsWidth - 168, Y: 2},
		Size:   CGSize{Width: 150, Height: 24},
	})
	v.reset.Send(sel("setBezelStyle:"), int64(1))
	v.view.Send(sel("addSubview:"), v.reset)
	settingsViews[v.reset] = v

	v.buildGeneral()
	v.buildData()
	v.buildCapture()
	v.buildDisplay()
	v.buildActions()
	v.buildInteraction()
	v.buildDeletion()
	v.buildWindow()
	v.buildHistory()
	v.buildPermissions()
	v.buildApps()
	v.buildShortcuts()
	v.buildPreview()
	v.buildUpdate()
	v.buildSidebar()
	v.buildSettingsHeader()
	v.buildSearch()
	settingsTabMu.Lock()
	settingsOpenViews[v] = struct{}{}
	tab := pendingSettingsTab
	pendingSettingsTab = ""
	settingsTabMu.Unlock()
	if tab == "" {
		tab = "general"
	}
	v.selectTab(tab)
	return v
}

func (v *SettingsView) selectTab(name string) {
	v.applyTab(name)
	v.selectSidebarRow(name)
}

func (v *SettingsView) applyTab(name string) {
	v.tabs.Send(sel("selectTabViewItemWithIdentifier:"), nsString(name))
	if v.sectionTitle != 0 {
		v.sectionTitle.Send(sel("setStringValue:"), nsString(settingsTabTitle(name)))
	}
}

// View implements core.NativeComponent.
func (v *SettingsView) View() any { return v.view }

// SetProp implements core.NativeComponent.
func (v *SettingsView) SetProp(name string, value any) {
	switch name {
	case "values":
		if values, ok := value.(SettingsValues); ok {
			v.lastValues = values
			v.apply(values)
		}
	case "error":
		text, _ := value.(string)
		if text == "" {
			v.footer.Send(sel("setStringValue:"), nsString(i18n.T(i18n.SettingsFooter)))
			v.footer.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("tertiaryLabelColor")))
			return
		}
		v.footer.Send(sel("setStringValue:"), nsString(text))
		v.footer.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("systemRedColor")))
		v.apply(v.lastValues)
	}
}

// Command implements core.NativeComponent.
func (v *SettingsView) Command(string, any) {}

func (v *SettingsView) apply(values SettingsValues) {
	v.dataDir.Send(sel("setStringValue:"), nsString(values.DataDir))
	v.dataDir.Send(sel("setToolTip:"), nsString(values.DataDir))
	v.applyDiagnostics(values)
	v.restore.Send(sel("setEnabled:"), values.CanRestoreDataDir)
	v.applyApps(values)
	v.applyActions(values)
	v.applyCaptureOrder(values)
	v.applyShortcuts(values)
	v.applyGroups(values)
	for _, field := range v.fields {
		switch field.kind {
		case "bool":
			on, _ := field.get(values).(bool)
			field.control.Send(sel("setState:"), int64(boolInt(on)))
		case "popup":
			value, _ := field.get(values).(string)
			selectPopup(field, value)
		case "stepper":
			n, _ := field.get(values).(uint32)
			field.control.Send(sel("setDoubleValue:"), float64(n))
			v.setStepperLabel(field, n)
		}
	}
	v.applyDisplayPreview(values)
}

// applyGroups rebuilds the open-group popup from the current clipboard
// groups instead of static options.
func (v *SettingsView) applyGroups(values SettingsValues) {
	field := v.groupField
	if field == nil {
		return
	}
	selected, _ := field.get(values).(string)
	options := []string{"preserve", "all"}
	titles := []string{i18n.T(i18n.SettingsPreserve), i18n.T(i18n.SettingsAllGroups)}
	for _, group := range values.Groups {
		options = append(options, "group:"+group.ID)
		titles = append(titles, group.Name)
	}
	field.options = options
	field.titles = titles
	field.control.Send(sel("removeAllItems"))
	for _, title := range titles {
		field.control.Send(sel("addItemWithTitle:"), nsString(title))
	}
	selectPopup(field, selected)
}

func selectPopup(field *settingsField, value string) {
	for i, option := range field.options {
		if option == value {
			field.control.Send(sel("selectItemAtIndex:"), int64(i))
			return
		}
	}
	field.options = append(field.options, value)
	field.titles = append(field.titles, value)
	field.control.Send(sel("addItemWithTitle:"), nsString(value))
	field.control.Send(sel("selectItemAtIndex:"), int64(len(field.options)-1))
}

func (v *SettingsView) setStepperLabel(field *settingsField, value uint32) {
	if field.valueLabel == 0 {
		return
	}
	text := fmt.Sprintf("%d%s", value, field.suffix)
	if value == 0 {
		text = i18n.T(i18n.SettingsUnlimited)
	}
	field.valueLabel.Send(sel("setStringValue:"), nsString(text))
}

func boolInt(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

// Close implements core.NativeComponent.
func (v *SettingsView) Close() {
	settingsTabMu.Lock()
	delete(settingsOpenViews, v)
	settingsTabMu.Unlock()
	if v.permissionTimer != 0 {
		v.permissionTimer.Send(sel("invalidate"))
		delete(settingsTimers, v.permissionTimer)
		v.permissionTimer = 0
	}
	if v.searchHighlightTimer != 0 {
		v.searchHighlightTimer.Send(sel("invalidate"))
		delete(settingsTimers, v.searchHighlightTimer)
		v.searchHighlightTimer = 0
	}
	for _, field := range v.fields {
		delete(settingsViews, field.control)
	}
	for control, owner := range settingsViews {
		if owner == v {
			delete(settingsViews, control)
		}
	}
	if v.appsTable != 0 {
		delete(settingsAppsByTable, v.appsTable)
	}
	if v.actionsTable != 0 {
		delete(settingsActionsByTable, v.actionsTable)
	}
	for control, owner := range settingsAppControls {
		if owner == v {
			delete(settingsAppControls, control)
		}
	}
	for control, owner := range settingsActionControls {
		if owner == v {
			delete(settingsActionControls, control)
		}
	}
	for control, binding := range settingsShortcutBindings {
		if binding.view == v {
			delete(settingsShortcutBindings, control)
			delete(settingsShortcutRecording, control)
		}
	}
	delete(settingsViews, v.change)
	v.view.Send(sel("release"))
}
