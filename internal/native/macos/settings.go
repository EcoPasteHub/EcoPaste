//go:build darwin

package macos

import (
	"fmt"
	"strings"
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
	cx                   *core.NativeContext
	view                 objc.ID
	tabs                 objc.ID
	footer               objc.ID
	searchField          objc.ID
	searchResults        objc.ID
	dataDir              objc.ID
	change               objc.ID
	restore              objc.ID
	reset                objc.ID
	accessibilityStatus  objc.ID
	fullDiskStatus       objc.ID
	diagnosticsDir       objc.ID
	diagnosticsHint      objc.ID
	permissionTimer      objc.ID
	appsTable            objc.ID
	appsEmpty            objc.ID
	apps                 []SettingsApp
	actionsTable         objc.ID
	actions              []settingsAction
	captureTable         objc.ID
	captureOrder         []string
	groupField           *settingsField
	extraExcluded        []string
	shortcutFields       []*shortcutControl
	lastValues           SettingsValues
	fields               []*settingsField
	currentTab           string
	currentTabTitle      string
	searchEntries        []settingsSearchEntry
	searchMatches        []settingsSearchEntry
	searchHighlight      objc.ID
	searchHighlightTimer objc.ID
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
	settingsWidth    = 560.0
	settingsHeight   = 600.0
	settingsTabAreaH = 528.0
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
	v.tabs.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 0, Y: 32}, Size: CGSize{Width: settingsWidth, Height: settingsTabAreaH}})
	v.tabs.Send(sel("setAutoresizingMask:"), int64(18))
	v.view.Send(sel("addSubview:"), v.tabs)

	v.footer = objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(i18n.T(i18n.SettingsFooter)))
	v.footer.Send(sel("setFont:"), systemFont(11))
	v.footer.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("tertiaryLabelColor")))
	v.footer.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 20, Y: 8}, Size: CGSize{Width: settingsWidth - 210, Height: 16}})
	v.view.Send(sel("addSubview:"), v.footer)

	v.reset = objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(i18n.T(i18n.SettingsReset)), settingsTarget, sel("resetSettingsPressed:"))
	v.reset.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: settingsWidth - 168, Y: 2}, Size: CGSize{Width: 150, Height: 24}})
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
	v.buildSearch()
	settingsTabMu.Lock()
	settingsOpenViews[v] = struct{}{}
	tab := pendingSettingsTab
	pendingSettingsTab = ""
	settingsTabMu.Unlock()
	if tab != "" {
		v.selectTab(tab)
	}
	return v
}

func (v *SettingsView) buildGeneral() {
	content := v.newTab("general", settingsTabTitle("general"))
	y := 450.0
	v.addSwitch(content, y, i18n.T(i18n.SettingsDockIcon), "general.dockIcon",
		func(s SettingsValues) bool { return s.DockIcon })
	v.addSwitch(content, y-38, i18n.T(i18n.SettingsTrayIcon), "general.trayIcon",
		func(s SettingsValues) bool { return s.TrayIcon })
	v.addSwitch(content, y-76, i18n.T(i18n.SettingsCopyThenHide), "copyThenHideWindow",
		func(s SettingsValues) bool { return s.CopyThenHideWindow })
	v.addSwitch(content, y-114, i18n.T(i18n.SettingsCopySound), "feedback.copySound",
		func(s SettingsValues) bool { return s.CopySound })
	v.addSwitch(content, y-152, i18n.T(i18n.SettingsUpdateOnReuse), "updateOnReuse",
		func(s SettingsValues) bool { return s.UpdateOnReuse })
	v.addSwitch(content, y-190, i18n.T(i18n.SettingsAutoStart), "general.autoStart",
		func(s SettingsValues) bool { return s.AutoStart })
	v.addPopup(content, y-228, i18n.T(i18n.SettingsSort), "sort",
		[]settingsOption{
			{Title: i18n.T(i18n.SettingsSortRecent), Value: "updatedAtDesc"},
			{Title: i18n.T(i18n.SettingsSortCreated), Value: "createdAtDesc"},
			{Title: i18n.T(i18n.SettingsSortUseCount), Value: "useCountDesc"},
		},
		func(s SettingsValues) string { return s.Sort })
	v.addPopup(content, y-266, i18n.T(i18n.SettingsPosition), "window.position",
		[]settingsOption{
			{Title: i18n.T(i18n.SettingsPositionFollow), Value: "followCursor"},
			{Title: i18n.T(i18n.SettingsPositionCenter), Value: "center"},
			{Title: i18n.T(i18n.SettingsPositionKeep), Value: "remember"},
		},
		func(s SettingsValues) string { return s.WindowPosition })
	v.addPopup(content, y-304, i18n.T(i18n.SettingsTheme), "appearance.theme",
		[]settingsOption{
			{Title: i18n.T(i18n.SettingsThemeAuto), Value: "auto"},
			{Title: i18n.T(i18n.SettingsThemeLight), Value: "light"},
			{Title: i18n.T(i18n.SettingsThemeDark), Value: "dark"},
		},
		func(s SettingsValues) string { return s.Theme })
	v.addPopup(content, y-342, i18n.T(i18n.SettingsLanguage), "appearance.language",
		[]settingsOption{
			{Title: i18n.T(i18n.SettingsLanguageZh), Value: "zh-CN"},
			{Title: i18n.T(i18n.SettingsLanguageEn), Value: "en-US"},
		},
		func(s SettingsValues) string { return s.Language })
}

func (v *SettingsView) buildData() {
	content := v.newTab("data", settingsTabTitle("data"))
	v.buildDataDir(content, 440)

	y := 330.0
	v.addLabel(content, y, i18n.T(i18n.SettingsBackupIntro))
	v.addSearchEntry(content, i18n.T(i18n.SettingsBackupIntro), "data.backup",
		i18n.T(i18n.SettingsBackupHint), i18n.T(i18n.SettingsBackupKeywords),
		CGRect{Origin: CGPoint{X: 12, Y: y - 70}, Size: CGSize{Width: tabContentW - 24, Height: 106}})
	hint := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(i18n.T(i18n.SettingsBackupHint)))
	hint.Send(sel("setFont:"), systemFont(11))
	hint.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	hint.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: y - 44}, Size: CGSize{Width: tabContentW - 48, Height: 32}})
	content.Send(sel("addSubview:"), hint)

	export := objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(i18n.T(i18n.SettingsExportBackup)), settingsTarget, sel("exportBackupPressed:"))
	export.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: y - 88}, Size: CGSize{Width: 140, Height: 28}})
	export.Send(sel("setBezelStyle:"), int64(1))
	export.Send(sel("setAccessibilityHelp:"), nsString(i18n.T(i18n.SettingsBackupHint)))
	content.Send(sel("addSubview:"), export)
	settingsViews[export] = v

	importButton := objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(i18n.T(i18n.SettingsImportBackup)), settingsTarget, sel("importBackupPressed:"))
	importButton.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 176, Y: y - 88}, Size: CGSize{Width: 140, Height: 28}})
	importButton.Send(sel("setBezelStyle:"), int64(1))
	importButton.Send(sel("setAccessibilityHelp:"), nsString(i18n.T(i18n.SettingsBackupHint)))
	content.Send(sel("addSubview:"), importButton)
	settingsViews[importButton] = v
}

func (v *SettingsView) buildDataDir(content objc.ID, y float64) {
	v.addLabel(content, y, i18n.T(i18n.SettingsDataDir))
	v.addSearchEntry(content, i18n.T(i18n.SettingsDataDir), "general.dataDir",
		i18n.T(i18n.SettingsDataDirKeywords), "",
		CGRect{Origin: CGPoint{X: 12, Y: y - 30}, Size: CGSize{Width: tabContentW - 24, Height: 46}})
	v.dataDir = objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
	v.dataDir.Send(sel("setFont:"), systemFont(11))
	v.dataDir.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("tertiaryLabelColor")))
	v.dataDir.Send(sel("setLineBreakMode:"), int64(4))
	v.dataDir.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: y - 20}, Size: CGSize{Width: tabContentW - 240, Height: 16}})
	content.Send(sel("addSubview:"), v.dataDir)

	v.restore = objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(i18n.T(i18n.SettingsRestoreDefault)), settingsTarget, sel("defaultDataDirPressed:"))
	v.restore.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: tabContentW - 204, Y: y - 24}, Size: CGSize{Width: 100, Height: 28}})
	content.Send(sel("addSubview:"), v.restore)
	settingsViews[v.restore] = v

	v.change = objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(i18n.T(i18n.SettingsChange)), settingsTarget, sel("dataDirPressed:"))
	v.change.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: tabContentW - 94, Y: y - 24}, Size: CGSize{Width: 70, Height: 28}})
	content.Send(sel("addSubview:"), v.change)
	settingsViews[v.change] = v
}

func (v *SettingsView) buildCapture() {
	content := v.newTab("capture", settingsTabTitle("capture"))
	y := 408.0
	v.addSwitch(content, y, i18n.T(i18n.SettingsCaptureText), "capture.text",
		func(s SettingsValues) bool { return s.CaptureText })
	v.addSwitch(content, y-38, i18n.T(i18n.SettingsCaptureHTML), "capture.html",
		func(s SettingsValues) bool { return s.CaptureHTML })
	v.addSwitch(content, y-76, i18n.T(i18n.SettingsCaptureRTF), "capture.rtf",
		func(s SettingsValues) bool { return s.CaptureRTF })
	v.addSwitch(content, y-114, i18n.T(i18n.KindImage), "capture.image",
		func(s SettingsValues) bool { return s.CaptureImage })
	v.addSwitch(content, y-152, i18n.T(i18n.KindFiles), "capture.files",
		func(s SettingsValues) bool { return s.CaptureFiles })
	v.addStepper(content, y-202, i18n.T(i18n.SettingsMaxText), "capture.maxTextMb", 1, 1024, " MB",
		func(s SettingsValues) uint32 { return s.MaxTextMB })
	v.addStepper(content, y-244, i18n.T(i18n.SettingsMaxImage), "capture.maxImageMb", 10, 4096, " MB",
		func(s SettingsValues) uint32 { return s.MaxImageMB })
	v.buildCaptureOrder(content)
}

func (v *SettingsView) buildDisplay() {
	content := v.newTab("display", settingsTabTitle("display"))
	y := 408.0
	v.addStepperRange(content, y, i18n.T(i18n.SettingsTextMaxLines), "display.textMaxLines",
		1, 5, 1, i18n.T(i18n.SettingsUnitLines),
		func(s SettingsValues) uint32 { return s.TextMaxLines })
	v.addStepperRange(content, y-42, i18n.T(i18n.SettingsImageMaxHeight), "display.imageMaxHeight", 20, 100, 4, " px",
		func(s SettingsValues) uint32 { return s.ImageMaxHeight })
	v.addStepperRange(content, y-84, i18n.T(i18n.SettingsFileMaxCount), "display.fileMaxCount",
		1, 5, 1, i18n.T(i18n.SettingsUnitItems),
		func(s SettingsValues) uint32 { return s.FileMaxCount })
}

func (v *SettingsView) buildWindow() {
	content := v.newTab("window", settingsTabTitle("window"))
	y := 408.0
	v.addSwitch(content, y, i18n.T(i18n.SettingsScrollTop), "window.scrollToTopOnOpen",
		func(s SettingsValues) bool { return s.ScrollToTopOnOpen })
	v.addPopup(content, y-42, i18n.T(i18n.SettingsSelectRange), "window.selectRangeOnOpen",
		[]settingsOption{
			{Title: i18n.T(i18n.SettingsPreserve), Value: "preserve"},
			{Title: i18n.T(i18n.SettingsAll), Value: "all"},
			{Title: i18n.T(i18n.PanelFavorites), Value: "favorite"},
		},
		func(s SettingsValues) string { return s.RangeOnOpen })
	v.addPopup(content, y-84, i18n.T(i18n.SettingsSelectCategory), "window.selectCategoryOnOpen",
		[]settingsOption{
			{Title: i18n.T(i18n.SettingsPreserve), Value: "preserve"},
			{Title: i18n.T(i18n.SettingsAll), Value: "all"},
			{Title: i18n.T(i18n.PanelTextCategory), Value: "text"},
			{Title: i18n.T(i18n.PanelImageCategory), Value: "image"},
			{Title: i18n.T(i18n.PanelFilesCategory), Value: "files"},
		},
		func(s SettingsValues) string { return s.CategoryOnOpen })
	v.groupField = v.addPopup(content, y-126, i18n.T(i18n.SettingsSelectGroup), "window.selectGroupOnOpen",
		[]settingsOption{
			{Title: i18n.T(i18n.SettingsPreserve), Value: "preserve"},
			{Title: i18n.T(i18n.SettingsAllGroups), Value: "all"},
		},
		func(s SettingsValues) string { return s.GroupOnOpen })
}

func (v *SettingsView) buildHistory() {
	content := v.newTab("history", settingsTabTitle("history"))
	y := 408.0
	v.addPopup(content, y, i18n.T(i18n.SettingsRetention), "history.retention",
		[]settingsOption{
			{Title: i18n.T(i18n.SettingsRetentionForever), Value: "forever"},
			{Title: i18n.T(i18n.SettingsRetention1Hour), Value: "1h"},
			{Title: i18n.T(i18n.SettingsRetention6Hours), Value: "6h"},
			{Title: i18n.T(i18n.SettingsRetention1Day), Value: "1d"},
			{Title: i18n.T(i18n.SettingsRetention3Days), Value: "3d"},
			{Title: i18n.T(i18n.SettingsRetention7Days), Value: "7d"},
			{Title: i18n.T(i18n.SettingsRetention2Weeks), Value: "2w"},
			{Title: i18n.T(i18n.SettingsRetention1Month), Value: "1mo"},
			{Title: i18n.T(i18n.SettingsRetention3Months), Value: "3mo"},
			{Title: i18n.T(i18n.SettingsRetention1Year), Value: "365d"},
		},
		func(s SettingsValues) string { return s.Retention })
	v.addStepper(content, y-42, i18n.T(i18n.SettingsMaxCount), "history.maxCount",
		100, 100000, i18n.T(i18n.SettingsUnitCount),
		func(s SettingsValues) uint32 { return s.MaxCount })
	v.addStepper(content, y-84, i18n.T(i18n.SettingsCleanupInterval), "history.cleanupIntervalHours",
		1, 168, i18n.T(i18n.SettingsUnitHours),
		func(s SettingsValues) uint32 { return s.CleanupIntervalHours })
	v.addSwitch(content, y-134, i18n.T(i18n.SettingsCollectSecrets), "sensitive.collectSecrets",
		func(s SettingsValues) bool { return s.CollectSecrets })
	v.addSwitch(content, y-176, i18n.T(i18n.SettingsRedactSecrets), "sensitive.redactSecrets",
		func(s SettingsValues) bool { return s.RedactSecrets })
}

func (v *SettingsView) buildPreview() {
	content := v.newTab("preview", settingsTabTitle("preview"))
	intro := objc.ID(classNSTextField).Send(sel("labelWithString:"),
		nsString(i18n.T(i18n.SettingsPreviewIntro)))
	intro.Send(sel("setFont:"), systemFont(11))
	intro.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	intro.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: 438}, Size: CGSize{Width: tabContentW - 48, Height: 18}})
	content.Send(sel("addSubview:"), intro)
	v.addSwitch(content, 396, i18n.T(i18n.SettingsSpacePreview), "preview.spaceEnabled",
		func(s SettingsValues) bool { return s.SpacePreview })
}

func (v *SettingsView) buildPermissions() {
	content := v.newTab("permissions", settingsTabTitle("permissions"))
	intro := objc.ID(classNSTextField).Send(sel("labelWithString:"),
		nsString(i18n.T(i18n.SettingsPermissionsIntro)))
	intro.Send(sel("setFont:"), systemFont(11))
	intro.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	intro.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: 438}, Size: CGSize{Width: tabContentW - 48, Height: 18}})
	content.Send(sel("addSubview:"), intro)

	y := 388.0
	v.accessibilityStatus = v.addPermission(content, y, i18n.T(i18n.SettingsAccessibility),
		i18n.T(i18n.SettingsAccessibilityDetail), "accessibilityPermissionPressed:")
	v.fullDiskStatus = v.addPermission(content, y-80, i18n.T(i18n.SettingsFullDisk),
		i18n.T(i18n.SettingsFullDiskDetail), "fullDiskAccessPermissionPressed:")
	v.buildDiagnostics(content)
	v.refreshPermissionLabels()
	v.startPermissionTimer()
}

func (v *SettingsView) addPermission(content objc.ID, y float64, title, detail, action string) objc.ID {
	v.addLabel(content, y, title)
	path := map[string]string{
		"accessibilityPermissionPressed:":  "permissions.accessibility",
		"fullDiskAccessPermissionPressed:": "permissions.fullDiskAccess",
	}[action]
	v.addSearchEntry(content, title, path, detail, i18n.T(i18n.SettingsPermissionsKeywords),
		CGRect{Origin: CGPoint{X: 12, Y: y - 32}, Size: CGSize{Width: tabContentW - 24, Height: 64}})
	detailLabel := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(detail))
	detailLabel.Send(sel("setFont:"), systemFont(11))
	detailLabel.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	detailLabel.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: y - 20}, Size: CGSize{Width: tabContentW - 200, Height: 16}})
	content.Send(sel("addSubview:"), detailLabel)

	status := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(i18n.T(i18n.SettingsChecking)))
	status.Send(sel("setFont:"), systemFont(11))
	status.Send(sel("setAlignment:"), int64(2))
	status.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: tabContentW - 300, Y: y}, Size: CGSize{Width: 130, Height: 18}})
	content.Send(sel("addSubview:"), status)

	button := objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(i18n.T(i18n.SettingsOpenSystemSettings)), settingsTarget, sel(action))
	button.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: tabContentW - 160, Y: y - 5}, Size: CGSize{Width: 136, Height: 28}})
	button.Send(sel("setAccessibilityLabel:"), nsString(i18n.Tf(i18n.SettingsOpenSystemSettingsA11y, title)))
	content.Send(sel("addSubview:"), button)
	return status
}

func (v *SettingsView) refreshPermissionLabels() {
	v.setPermissionStatus(v.accessibilityStatus, AccessibilityGranted())
	v.setPermissionStatus(v.fullDiskStatus, FullDiskAccessGranted())
}

func (v *SettingsView) setPermissionStatus(label objc.ID, granted bool) {
	if label == 0 {
		return
	}
	text, color := i18n.T(i18n.OnboardingNotGranted), "systemRedColor"
	if granted {
		text, color = i18n.T(i18n.OnboardingGranted), "systemGreenColor"
	}
	label.Send(sel("setStringValue:"), nsString(text))
	label.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel(color)))
}

func (v *SettingsView) startPermissionTimer() {
	timer := objc.ID(class("NSTimer")).Send(sel("scheduledTimerWithTimeInterval:target:selector:userInfo:repeats:"),
		1.5, settingsTarget, sel("permissionTimerFired:"), objc.ID(0), true)
	settingsTimers[timer] = v
	v.permissionTimer = timer
}

type settingsOption struct {
	Title string
	Value string
}

func (v *SettingsView) newTab(identifier, title string) objc.ID {
	v.currentTab = identifier
	v.currentTabTitle = title
	item := objc.ID(class("NSTabViewItem")).Send(sel("alloc")).Send(sel("initWithIdentifier:"), nsString(identifier))
	item.Send(sel("setLabel:"), nsString(title))
	content := objc.ID(class("NSView")).Send(sel("new"))
	content.Send(sel("setFrame:"), CGRect{Size: CGSize{Width: tabContentW, Height: tabContentH}})
	item.Send(sel("setView:"), content)
	v.tabs.Send(sel("addTabViewItem:"), item)
	item.Send(sel("release"))
	content.Send(sel("release"))
	return content
}

func (v *SettingsView) selectTab(name string) {
	v.tabs.Send(sel("selectTabViewItemWithIdentifier:"), nsString(name))
}

func (v *SettingsView) addSwitch(content objc.ID, y float64, title, path string, get func(SettingsValues) bool) {
	v.addLabel(content, y, title)
	v.addSearchEntry(content, title, path, "", "", rowHighlightFrame(y))
	control := objc.ID(class("NSSwitch")).Send(sel("new"))
	control.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: tabContentW - 64, Y: y - 2}, Size: CGSize{Width: 40, Height: 24}})
	control.Send(sel("setTarget:"), settingsTarget)
	control.Send(sel("setAction:"), sel("toggleChanged:"))
	v.addField(content, control, &settingsField{
		path: path, kind: "bool", control: control,
		get: func(s SettingsValues) any { return get(s) },
	})
}

func (v *SettingsView) addPopup(content objc.ID, y float64, title, path string, options []settingsOption, get func(SettingsValues) string) *settingsField {
	v.addLabel(content, y, title)
	values := make([]string, 0, len(options))
	titles := make([]string, 0, len(options))
	control := objc.ID(class("NSPopUpButton")).Send(sel("alloc")).Send(sel("initWithFrame:pullsDown:"),
		CGRect{Origin: CGPoint{X: tabContentW - 260, Y: y - 4}, Size: CGSize{Width: 240, Height: 28}}, false)
	for _, option := range options {
		values = append(values, option.Value)
		titles = append(titles, option.Title)
		control.Send(sel("addItemWithTitle:"), nsString(option.Title))
	}
	v.addSearchEntry(content, title, path, "", strings.Join(titles, " "), rowHighlightFrame(y))
	control.Send(sel("setTarget:"), settingsTarget)
	control.Send(sel("setAction:"), sel("popupChanged:"))
	field := &settingsField{
		path: path, kind: "popup", control: control, options: values, titles: titles,
		get: func(s SettingsValues) any { return get(s) },
	}
	v.addField(content, control, field)
	return field
}

func (v *SettingsView) addStepper(content objc.ID, y float64, title, path string, step, maxValue float64, suffix string, get func(SettingsValues) uint32) {
	v.addStepperRange(content, y, title, path, 0, maxValue, step, suffix, get)
}

func (v *SettingsView) addStepperRange(content objc.ID, y float64, title, path string, minValue, maxValue, step float64, suffix string, get func(SettingsValues) uint32) {
	v.addLabel(content, y, title)
	v.addSearchEntry(content, title, path, suffix, "", rowHighlightFrame(y))
	valueLabel := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
	valueLabel.Send(sel("setFont:"), systemFont(12))
	valueLabel.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	valueLabel.Send(sel("setAlignment:"), int64(2))
	valueLabel.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: tabContentW - 210, Y: y}, Size: CGSize{Width: 120, Height: 18}})
	content.Send(sel("addSubview:"), valueLabel)

	control := objc.ID(class("NSStepper")).Send(sel("new"))
	control.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: tabContentW - 80, Y: y - 5}, Size: CGSize{Width: 19, Height: 27}})
	control.Send(sel("setMinValue:"), minValue)
	control.Send(sel("setMaxValue:"), maxValue)
	control.Send(sel("setIncrement:"), step)
	control.Send(sel("setValueWraps:"), false)
	control.Send(sel("setTarget:"), settingsTarget)
	control.Send(sel("setAction:"), sel("stepperChanged:"))
	v.addField(content, control, &settingsField{
		path: path, kind: "stepper", control: control, valueLabel: valueLabel,
		suffix: suffix,
		get:    func(s SettingsValues) any { return get(s) },
	})
}

func (v *SettingsView) addLabel(content objc.ID, y float64, title string) objc.ID {
	label := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(title))
	label.Send(sel("setFont:"), systemFont(13))
	label.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: y}, Size: CGSize{Width: tabContentW - 300, Height: 18}})
	content.Send(sel("addSubview:"), label)
	return label
}

func (v *SettingsView) addField(content, control objc.ID, field *settingsField) {
	tag := int64(len(v.fields) + 1)
	control.Send(sel("setTag:"), tag)
	v.fields = append(v.fields, field)
	content.Send(sel("addSubview:"), control)
	settingsViews[control] = v
}

func (v *SettingsView) fieldFor(sender objc.ID) *settingsField {
	tag := int(objc.Send[int64](sender, sel("tag")))
	if tag < 1 || tag > len(v.fields) {
		return nil
	}
	return v.fields[tag-1]
}

func registerSettingsTargets() {
	target, err := objc.RegisterClass("EcoPasteSettingsTarget", class("NSObject"), nil, nil, []objc.MethodDef{
		{Cmd: sel("settingsSearchChanged:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			if v := settingsViews[sender]; v != nil {
				v.searchChanged(sender)
			}
		}},
		{Cmd: sel("settingsSearchPicked:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			if v := settingsViews[sender]; v != nil {
				v.searchResultPicked(sender)
			}
		}},
		{Cmd: sel("settingsSearchHighlightExpired:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			if v := settingsTimers[sender]; v != nil {
				delete(settingsTimers, sender)
				v.searchHighlightTimer = 0
				if v.searchHighlight != 0 {
					v.searchHighlight.Send(sel("setHidden:"), true)
				}
			}
		}},
		{Cmd: sel("toggleChanged:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			v := settingsViews[sender]
			if v == nil {
				return
			}
			field := v.fieldFor(sender)
			if field == nil {
				return
			}
			on := objc.Send[int64](sender, sel("state")) == 1
			v.cx.Emit("change", SettingsChange{Path: field.path, Value: on})
		}},
		{Cmd: sel("popupChanged:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			v := settingsViews[sender]
			if v == nil {
				return
			}
			field := v.fieldFor(sender)
			if field == nil {
				return
			}
			index := int(objc.Send[int64](sender, sel("indexOfSelectedItem")))
			if index < 0 || index >= len(field.options) {
				return
			}
			v.cx.Emit("change", SettingsChange{Path: field.path, Value: field.options[index]})
		}},
		{Cmd: sel("stepperChanged:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			v := settingsViews[sender]
			if v == nil {
				return
			}
			field := v.fieldFor(sender)
			if field == nil {
				return
			}
			value := uint32(objc.Send[float64](sender, sel("doubleValue")))
			v.setStepperLabel(field, value)
			v.cx.Emit("change", SettingsChange{Path: field.path, Value: value})
		}},
		{Cmd: sel("dataDirPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			v := settingsViews[sender]
			if v == nil {
				return
			}
			parent, ok := OpenDataDirectoryPanel(i18n.T(i18n.SettingsChooseDataDir))
			if !ok {
				return
			}
			if !Confirm(i18n.T(i18n.SettingsChangeDataDirTitle),
				i18n.T(i18n.SettingsChangeDataDirBody), i18n.T(i18n.SettingsMigrateRestart)) {
				return
			}
			v.cx.Emit("change", SettingsChange{Path: "general.dataDir", Value: parent})
		}},
		{Cmd: sel("defaultDataDirPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			v := settingsViews[sender]
			if v == nil {
				return
			}
			if !Confirm(i18n.T(i18n.SettingsRestoreDirTitle),
				i18n.T(i18n.SettingsRestoreDirBody), i18n.T(i18n.SettingsMigrateRestart)) {
				return
			}
			v.cx.Emit("change", SettingsChange{Path: "general.defaultDataDir", Value: true})
		}},
		{Cmd: sel("accessibilityPermissionPressed:"), Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) {
			OpenAccessibilitySettings()
		}},
		{Cmd: sel("fullDiskAccessPermissionPressed:"), Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) {
			OpenFullDiskAccessSettings()
		}},
		{Cmd: sel("permissionTimerFired:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			if v := settingsTimers[sender]; v != nil {
				v.refreshPermissionLabels()
			}
		}},
		{Cmd: sel("copyDiagnosticsPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			if v := settingsViews[sender]; v != nil {
				v.copyDiagnostics()
			}
		}},
		{Cmd: sel("addAppPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			v := settingsViews[sender]
			if v == nil {
				return
			}
			path, ok := OpenApplicationPanel(i18n.T(i18n.SettingsChooseExcludedApp))
			if !ok {
				return
			}
			v.cx.Emit("change", SettingsChange{Path: "apps.add", Value: path})
		}},
		{Cmd: sel("refreshAppsPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			v := settingsViews[sender]
			if v == nil {
				return
			}
			v.cx.Emit("change", SettingsChange{Path: "apps.refresh", Value: true})
		}},
		{Cmd: sel("exportBackupPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			v := settingsViews[sender]
			if v == nil {
				return
			}
			v.cx.Emit("change", SettingsChange{Path: "backup.export", Value: true})
		}},
		{Cmd: sel("importBackupPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			v := settingsViews[sender]
			if v == nil {
				return
			}
			v.cx.Emit("change", SettingsChange{Path: "backup.import", Value: true})
		}},
		{Cmd: sel("resetSettingsPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			v := settingsViews[sender]
			if v == nil {
				return
			}
			if !Confirm(i18n.T(i18n.SettingsResetTitle),
				i18n.T(i18n.SettingsResetBody), i18n.T(i18n.SettingsResetConfirm)) {
				return
			}
			v.cx.Emit("change", SettingsChange{Path: "settings.reset", Value: true})
		}},
	})
	if err != nil {
		panic(err)
	}
	settingsTarget = objc.ID(target).Send(sel("new"))
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
