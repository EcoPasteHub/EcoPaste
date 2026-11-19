//go:build darwin

package macos

import (
	"sync"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
	"github.com/nsgo-dev/nsgo"
	"github.com/nsgo-dev/nsgo/core"
)

// SettingsComponent is the native settings window's content view.
const SettingsComponent = "ecopaste.settings"

// SettingsValues is the view state of the settings window.
type SettingsValues struct {
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
	cx                  *core.NativeContext
	view                objc.ID
	tabs                objc.ID
	footer              objc.ID
	dataDir             objc.ID
	change              objc.ID
	restore             objc.ID
	reset               objc.ID
	accessibilityStatus objc.ID
	fullDiskStatus      objc.ID
	diagnosticsHint     objc.ID
	permissionTimer     objc.ID
	appsTable           objc.ID
	appsEmpty           objc.ID
	apps                []SettingsApp
	actionsTable        objc.ID
	actionUp            objc.ID
	actionDown          objc.ID
	actions             []settingsAction
	captureTable        objc.ID
	captureUp           objc.ID
	captureDown         objc.ID
	captureOrder        []string
	groupField          *settingsField
	extraExcluded       []string
	shortcutRecorders   []*shortcutRecorder
	numberInputs        []*numberInput
	infoButtons         []objc.ID
	lastValues          SettingsValues
	fields              []*settingsField
	currentTab          string
	currentTabTitle     string
	language            string
	texts               []func()
	tabItems            map[string]objc.ID
	aboutIcons          []*nsgo.View
}

type settingsField struct {
	path    string
	kind    string
	control objc.ID
	options []string
	titles  []string
	choices []settingsOption
	title   i18n.Key
	get     func(SettingsValues) any
}

const (
	settingsWidth    = 480.0
	settingsHeight   = 560.0
	settingsFooterH  = 0.0
	settingsTabAreaH = settingsHeight
)

type settingsTabMeta struct {
	title  i18n.Key
	symbol string
}

var settingsTabs = map[string]settingsTabMeta{
	"general":   {i18n.SettingsTabGeneral, "gearshape"},
	"interface": {i18n.SettingsTabInterface, "paintpalette"},
	"apps":      {i18n.SettingsTabApps, "square.grid.2x2"},
	"advanced":  {i18n.SettingsTabAdvanced, "slider.horizontal.3"},
	"about":     {i18n.SettingsTabAbout, "info.circle"},
}

var settingsTabAliases = map[string]string{
	"data":        "advanced",
	"capture":     "advanced",
	"display":     "interface",
	"actions":     "advanced",
	"interaction": "interface",
	"deletion":    "advanced",
	"window":      "interface",
	"history":     "advanced",
	"permissions": "advanced",
	"shortcuts":   "general",
	"preview":     "interface",
	"update":      "about",
}

func canonicalSettingsTab(name string) (string, bool) {
	if _, ok := settingsTabs[name]; ok {
		return name, true
	}
	if tab, ok := settingsTabAliases[name]; ok {
		return tab, true
	}
	return "", false
}

func settingsTabTitle(name string) string {
	tab, ok := canonicalSettingsTab(name)
	if !ok {
		return name
	}
	return i18n.T(settingsTabs[tab].title)
}

// SettingsTabTitle returns the localized title of a primary settings tab.
func SettingsTabTitle(name string) string { return settingsTabTitle(name) }

// InitialSettingsTab returns the tab a new settings window should show.
func InitialSettingsTab() string {
	settingsTabMu.Lock()
	defer settingsTabMu.Unlock()
	if tab, ok := canonicalSettingsTab(pendingSettingsTab); ok {
		return tab
	}
	return "general"
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

// SelectSettingsTab shows the named tab when the settings window is open,
// or on the next open otherwise. It reports whether the name is known.
func SelectSettingsTab(name string) bool {
	tab, ok := canonicalSettingsTab(name)
	if !ok {
		return false
	}
	settingsTabMu.Lock()
	if len(settingsOpenViews) == 0 {
		pendingSettingsTab = tab
		settingsTabMu.Unlock()
		return true
	}
	views := make([]*SettingsView, 0, len(settingsOpenViews))
	for view := range settingsOpenViews {
		views = append(views, view)
	}
	settingsTabMu.Unlock()
	// This runs on the reactive graph when the clipboard panel's gear opens
	// settings; the tab view is AppKit, so switch it on the main thread. A
	// view closed between the snapshot and here is skipped.
	onMain(func() {
		for _, view := range views {
			settingsTabMu.Lock()
			_, open := settingsOpenViews[view]
			settingsTabMu.Unlock()
			if open {
				view.selectTabNotify(tab)
			}
		}
	})
	return true
}

// NewSettings builds the native settings controls.
func NewSettings(cx *core.NativeContext) core.NativeComponent {
	registerSettingsOnce.Do(registerSettingsTargets)
	syncSettingsDockIcon(true)
	v := &SettingsView{
		cx:       cx,
		view:     objc.ID(class("NSView")).Send(sel("new")),
		language: string(i18n.Current()),
		tabItems: map[string]objc.ID{},
	}
	v.view.Send(sel("setAutoresizesSubviews:"), false)

	v.tabs = objc.ID(class("NSTabView")).Send(sel("new"))
	v.tabs.Send(sel("setTabViewType:"), int64(6))
	v.tabs.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: 0, Y: 0},
		Size:   CGSize{Width: settingsWidth, Height: settingsTabAreaH},
	})
	v.tabs.Send(sel("setAutoresizingMask:"), int64(18))
	v.view.Send(sel("addSubview:"), v.tabs)

	v.footer = objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
	v.footer.Send(sel("setFont:"), systemFont(11))
	v.footer.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("systemRedColor")))
	v.footer.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: 28, Y: 6},
		Size:   CGSize{Width: settingsWidth - 56, Height: 16},
	})
	v.footer.Send(sel("setHidden:"), true)
	v.view.Send(sel("addSubview:"), v.footer)

	v.buildGeneralPage()
	v.buildInterfacePage()
	v.buildAppsPage()
	v.buildAdvancedPage()
	v.buildAboutPage()
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
	tab, ok := canonicalSettingsTab(name)
	if !ok {
		return
	}
	v.applyTab(tab)
}

func (v *SettingsView) selectTabNotify(name string) {
	tab, ok := canonicalSettingsTab(name)
	if !ok {
		return
	}
	v.selectTab(tab)
	v.cx.Emit("tabSelected", tab)
}

func (v *SettingsView) applyTab(name string) {
	if v.currentTab == name {
		return
	}
	v.tabs.Send(sel("selectTabViewItemWithIdentifier:"), nsString(name))
	v.currentTab = name
}

// View implements core.NativeComponent.
func (v *SettingsView) View() any { return v.view }

// SetProp implements core.NativeComponent.
func (v *SettingsView) SetProp(name string, value any) {
	switch name {
	case "language":
		if language, ok := value.(string); ok {
			v.applyLanguage(language)
		}
	case "values":
		if values, ok := value.(SettingsValues); ok {
			v.apply(values)
		}
	case "tab":
		if tab, ok := value.(string); ok {
			v.selectTab(tab)
		}
	case "error":
		text, _ := value.(string)
		if text == "" {
			v.footer.Send(sel("setStringValue:"), nsString(""))
			v.footer.Send(sel("setHidden:"), true)
			return
		}
		v.footer.Send(sel("setStringValue:"), nsString(text))
		v.footer.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("systemRedColor")))
		v.footer.Send(sel("setHidden:"), false)
		v.apply(v.lastValues)
	}
}

// Command implements core.NativeComponent.
func (v *SettingsView) Command(name string, args any) {
	switch name {
	case "shortcutAccepted":
		change, ok := args.(SettingsChange)
		if !ok {
			return
		}
		if recorder := v.shortcutRecorder(change.Path); recorder != nil {
			value, _ := change.Value.(string)
			recorder.acceptRecording(value)
		}
	case "shortcutRejected":
		path, _ := args.(string)
		if recorder := v.shortcutRecorder(path); recorder != nil {
			recorder.rejectRecording()
		}
	}
}

func (v *SettingsView) shortcutRecorder(path string) *shortcutRecorder {
	for _, recorder := range v.shortcutRecorders {
		if recorder.path == path {
			return recorder
		}
	}
	return nil
}

// apply reconciles the native controls with the persisted values. Each list and
// popup updates itself only when its data changed, so changing one setting does
// not reload or re-scroll unrelated regions.
func (v *SettingsView) apply(values SettingsValues) {
	prev := v.lastValues
	if v.dataDir != 0 {
		v.dataDir.Send(sel("setStringValue:"), nsString(values.DataDir))
		v.dataDir.Send(sel("setToolTip:"), nsString(values.DataDir))
	}
	if v.restore != 0 {
		v.restore.Send(sel("setEnabled:"), values.CanRestoreDataDir)
	}
	v.applyApps(values)
	v.applyActions(values)
	v.applyCaptureOrder(values)
	v.applyShortcuts(values)
	v.applyNumberInputs(values)
	if !settingsGroupsEqual(prev.Groups, values.Groups) || prev.GroupOnOpen != values.GroupOnOpen {
		v.applyGroups(values)
	}
	for _, field := range v.fields {
		switch field.kind {
		case "bool":
			on, _ := field.get(values).(bool)
			field.control.Send(sel("setState:"), int64(boolInt(on)))
		case "popup":
			value, _ := field.get(values).(string)
			selectPopup(field, value)
			v.layoutPopup(field)
		}
	}
	v.lastValues = values
}

// settingsGroupsEqual reports whether two group lists are identical.
func settingsGroupsEqual(a, b []SettingsGroup) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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
	v.layoutPopup(field)
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
	remaining := len(settingsOpenViews)
	settingsTabMu.Unlock()
	if remaining == 0 {
		syncSettingsDockIcon(false)
	}
	for _, icon := range v.aboutIcons {
		icon.Close()
	}
	v.aboutIcons = nil
	if v.permissionTimer != 0 {
		v.permissionTimer.Send(sel("invalidate"))
		delete(settingsTimers, v.permissionTimer)
		v.permissionTimer = 0
	}
	for control, owner := range settingsViews {
		if owner == v {
			delete(settingsViews, control)
		}
	}
	for _, field := range v.fields {
		if field.kind == "popup" {
			discardSettingsPopup(field.control)
		}
	}
	if v.appsTable != 0 {
		delete(settingsAppsByTable, v.appsTable)
	}
	if v.actionsTable != 0 {
		delete(settingsActionsByTable, v.actionsTable)
	}
	if v.captureTable != 0 {
		delete(settingsCaptureByTable, v.captureTable)
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
	v.closeShortcutRecorders()
	v.closeNumberInputs()
	v.closeSettingsInfoButtons()
	delete(settingsViews, v.change)
	v.view.Send(sel("release"))
}
