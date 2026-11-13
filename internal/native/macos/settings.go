//go:build darwin

package macos

import (
	"fmt"
	"sync"

	"github.com/ebitengine/purego/objc"
	"github.com/nsgo-dev/nsgo/core"
)

// SettingsComponent is the native settings window's content view.
const SettingsComponent = "ecopaste.settings"

// SettingsValues is the view state of the settings window.
type SettingsValues struct {
	DockIcon               bool
	AutoStart              bool
	Theme                  string
	DataDir                string
	CanRestoreDataDir      bool
	CopyPlain              bool
	CopyThenHideWindow     bool
	PastePlain             bool
	UpdateOnReuse          bool
	Sort                   string
	WindowPosition         string
	CaptureText            bool
	CaptureHTML            bool
	CaptureRTF             bool
	CaptureImage           bool
	CaptureFiles           bool
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
	OpenClipboardShortcut  string
	OpenPreferenceShortcut string
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
	permissionTimer     objc.ID
	appsTable           objc.ID
	appsEmpty           objc.ID
	apps                []SettingsApp
	groupField          *settingsField
	extraExcluded       []string
	shortcutFields      []*shortcutControl
	lastValues          SettingsValues
	fields              []*settingsField
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
	settingsWidth  = 560.0
	settingsHeight = 560.0
	tabContentW    = 520.0
	tabContentH    = 470.0
)

var (
	registerSettingsOnce sync.Once
	settingsTarget       objc.ID
	settingsViews        = map[objc.ID]*SettingsView{}
	settingsTimers       = map[objc.ID]*SettingsView{}
)

// NewSettings builds the native settings controls.
func NewSettings(cx *core.NativeContext) core.NativeComponent {
	registerSettingsOnce.Do(registerSettingsTargets)
	v := &SettingsView{
		cx:   cx,
		view: objc.ID(class("NSView")).Send(sel("new")),
	}
	v.view.Send(sel("setAutoresizesSubviews:"), false)

	v.tabs = objc.ID(class("NSTabView")).Send(sel("new"))
	v.tabs.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 0, Y: 32}, Size: CGSize{Width: settingsWidth, Height: settingsHeight - 32}})
	v.tabs.Send(sel("setAutoresizingMask:"), int64(18))
	v.view.Send(sel("addSubview:"), v.tabs)

	v.footer = objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString("更改会立即保存"))
	v.footer.Send(sel("setFont:"), systemFont(11))
	v.footer.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("tertiaryLabelColor")))
	v.footer.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 20, Y: 8}, Size: CGSize{Width: settingsWidth - 210, Height: 16}})
	v.view.Send(sel("addSubview:"), v.footer)

	v.reset = objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString("恢复默认设置…"), settingsTarget, sel("resetSettingsPressed:"))
	v.reset.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: settingsWidth - 168, Y: 2}, Size: CGSize{Width: 150, Height: 24}})
	v.reset.Send(sel("setBezelStyle:"), int64(1))
	v.view.Send(sel("addSubview:"), v.reset)
	settingsViews[v.reset] = v

	v.buildGeneral()
	v.buildCapture()
	v.buildDisplay()
	v.buildWindow()
	v.buildHistory()
	v.buildPermissions()
	v.buildApps()
	v.buildShortcuts()
	return v
}

func (v *SettingsView) buildGeneral() {
	content := v.newTab("通用")
	y := 450.0
	v.addSwitch(content, y, "显示 Dock 图标", "general.dockIcon",
		func(s SettingsValues) bool { return s.DockIcon })
	y -= 42
	v.addSwitch(content, y, "复制后隐藏面板", "copyThenHideWindow",
		func(s SettingsValues) bool { return s.CopyThenHideWindow })
	v.addSwitch(content, y-42, "复制文本为纯文本", "copyPlain",
		func(s SettingsValues) bool { return s.CopyPlain })
	v.addSwitch(content, y-84, "粘贴文本为纯文本", "pastePlain",
		func(s SettingsValues) bool { return s.PastePlain })
	v.addSwitch(content, y-126, "复用后刷新排序", "updateOnReuse",
		func(s SettingsValues) bool { return s.UpdateOnReuse })
	v.addSwitch(content, y-168, "登录时启动", "general.autoStart",
		func(s SettingsValues) bool { return s.AutoStart })
	v.addPopup(content, y-210, "排序方式", "sort",
		[]settingsOption{
			{Title: "最近使用", Value: "updatedAtDesc"},
			{Title: "创建时间", Value: "createdAtDesc"},
			{Title: "使用次数", Value: "useCountDesc"},
		},
		func(s SettingsValues) string { return s.Sort })
	v.addPopup(content, y-252, "面板打开位置", "window.position",
		[]settingsOption{
			{Title: "跟随光标", Value: "followCursor"},
			{Title: "屏幕居中", Value: "center"},
			{Title: "记住位置", Value: "remember"},
		},
		func(s SettingsValues) string { return s.WindowPosition })
	v.addPopup(content, y-294, "外观主题", "appearance.theme",
		[]settingsOption{
			{Title: "跟随系统", Value: "auto"},
			{Title: "浅色", Value: "light"},
			{Title: "深色", Value: "dark"},
		},
		func(s SettingsValues) string { return s.Theme })
	v.buildDataDir(content, y-336)
}

func (v *SettingsView) buildDataDir(content objc.ID, y float64) {
	v.addLabel(content, y, "数据目录")
	v.dataDir = objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
	v.dataDir.Send(sel("setFont:"), systemFont(11))
	v.dataDir.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("tertiaryLabelColor")))
	v.dataDir.Send(sel("setLineBreakMode:"), int64(4))
	v.dataDir.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: y - 20}, Size: CGSize{Width: tabContentW - 240, Height: 16}})
	content.Send(sel("addSubview:"), v.dataDir)

	v.restore = objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString("恢复默认…"), settingsTarget, sel("defaultDataDirPressed:"))
	v.restore.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: tabContentW - 204, Y: y - 24}, Size: CGSize{Width: 100, Height: 28}})
	content.Send(sel("addSubview:"), v.restore)
	settingsViews[v.restore] = v

	v.change = objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString("更改…"), settingsTarget, sel("dataDirPressed:"))
	v.change.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: tabContentW - 94, Y: y - 24}, Size: CGSize{Width: 70, Height: 28}})
	content.Send(sel("addSubview:"), v.change)
	settingsViews[v.change] = v
}

func (v *SettingsView) buildCapture() {
	content := v.newTab("采集")
	y := 408.0
	v.addSwitch(content, y, "纯文本", "capture.text",
		func(s SettingsValues) bool { return s.CaptureText })
	v.addSwitch(content, y-38, "HTML", "capture.html",
		func(s SettingsValues) bool { return s.CaptureHTML })
	v.addSwitch(content, y-76, "RTF", "capture.rtf",
		func(s SettingsValues) bool { return s.CaptureRTF })
	v.addSwitch(content, y-114, "图片", "capture.image",
		func(s SettingsValues) bool { return s.CaptureImage })
	v.addSwitch(content, y-152, "文件", "capture.files",
		func(s SettingsValues) bool { return s.CaptureFiles })
	v.addStepper(content, y-202, "文本大小上限", "capture.maxTextMb", 1, 1024, " MB",
		func(s SettingsValues) uint32 { return s.MaxTextMB })
	v.addStepper(content, y-244, "图片大小上限", "capture.maxImageMb", 10, 4096, " MB",
		func(s SettingsValues) uint32 { return s.MaxImageMB })
}

func (v *SettingsView) buildDisplay() {
	content := v.newTab("显示")
	y := 408.0
	v.addStepperRange(content, y, "文本显示行数", "display.textMaxLines", 1, 5, 1, " 行",
		func(s SettingsValues) uint32 { return s.TextMaxLines })
	v.addStepperRange(content, y-42, "图片最大高度", "display.imageMaxHeight", 20, 100, 4, " px",
		func(s SettingsValues) uint32 { return s.ImageMaxHeight })
	v.addStepperRange(content, y-84, "文件名显示数量", "display.fileMaxCount", 1, 5, 1, " 个",
		func(s SettingsValues) uint32 { return s.FileMaxCount })
}

func (v *SettingsView) buildWindow() {
	content := v.newTab("窗口")
	y := 408.0
	v.addSwitch(content, y, "打开时滚动到顶部", "window.scrollToTopOnOpen",
		func(s SettingsValues) bool { return s.ScrollToTopOnOpen })
	v.addPopup(content, y-42, "打开时选中范围", "window.selectRangeOnOpen",
		[]settingsOption{
			{Title: "保持当前", Value: "preserve"},
			{Title: "全部", Value: "all"},
			{Title: "收藏", Value: "favorite"},
		},
		func(s SettingsValues) string { return s.RangeOnOpen })
	v.addPopup(content, y-84, "打开时选中分类", "window.selectCategoryOnOpen",
		[]settingsOption{
			{Title: "保持当前", Value: "preserve"},
			{Title: "全部", Value: "all"},
			{Title: "文本", Value: "text"},
			{Title: "图片", Value: "image"},
			{Title: "文件", Value: "files"},
		},
		func(s SettingsValues) string { return s.CategoryOnOpen })
	v.groupField = v.addPopup(content, y-126, "打开时选中分组", "window.selectGroupOnOpen",
		[]settingsOption{
			{Title: "保持当前", Value: "preserve"},
			{Title: "全部分组", Value: "all"},
		},
		func(s SettingsValues) string { return s.GroupOnOpen })
}

func (v *SettingsView) buildHistory() {
	content := v.newTab("历史与隐私")
	y := 408.0
	v.addPopup(content, y, "保留时长", "history.retention",
		[]settingsOption{
			{Title: "永久", Value: "forever"},
			{Title: "1 小时", Value: "1h"},
			{Title: "6 小时", Value: "6h"},
			{Title: "1 天", Value: "1d"},
			{Title: "3 天", Value: "3d"},
			{Title: "7 天", Value: "7d"},
			{Title: "2 周", Value: "2w"},
			{Title: "1 个月", Value: "1mo"},
			{Title: "3 个月", Value: "3mo"},
			{Title: "1 年", Value: "365d"},
		},
		func(s SettingsValues) string { return s.Retention })
	v.addStepper(content, y-42, "最大记录数", "history.maxCount", 100, 100000, " 条",
		func(s SettingsValues) uint32 { return s.MaxCount })
	v.addStepper(content, y-84, "自动清理间隔", "history.cleanupIntervalHours", 1, 168, " 小时",
		func(s SettingsValues) uint32 { return s.CleanupIntervalHours })
	v.addSwitch(content, y-134, "收录敏感内容", "sensitive.collectSecrets",
		func(s SettingsValues) bool { return s.CollectSecrets })
	v.addSwitch(content, y-176, "脱敏显示", "sensitive.redactSecrets",
		func(s SettingsValues) bool { return s.RedactSecrets })
}

func (v *SettingsView) buildPermissions() {
	content := v.newTab("权限")
	intro := objc.ID(classNSTextField).Send(sel("labelWithString:"),
		nsString("这些权限由 macOS 管理，授权后立即生效。"))
	intro.Send(sel("setFont:"), systemFont(11))
	intro.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	intro.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: 438}, Size: CGSize{Width: tabContentW - 48, Height: 18}})
	content.Send(sel("addSubview:"), intro)

	y := 388.0
	v.accessibilityStatus = v.addPermission(content, y, "辅助功能",
		"自动粘贴需要模拟 Command-V 快捷键。", "accessibilityPermissionPressed:")
	v.fullDiskStatus = v.addPermission(content, y-80, "完全磁盘访问",
		"读取受系统保护的目录和文件时需要。", "fullDiskAccessPermissionPressed:")
	v.refreshPermissionLabels()
	v.startPermissionTimer()
}

func (v *SettingsView) addPermission(content objc.ID, y float64, title, detail, action string) objc.ID {
	v.addLabel(content, y, title)
	detailLabel := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(detail))
	detailLabel.Send(sel("setFont:"), systemFont(11))
	detailLabel.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	detailLabel.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: y - 20}, Size: CGSize{Width: tabContentW - 200, Height: 16}})
	content.Send(sel("addSubview:"), detailLabel)

	status := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString("检查中…"))
	status.Send(sel("setFont:"), systemFont(11))
	status.Send(sel("setAlignment:"), int64(2))
	status.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: tabContentW - 300, Y: y}, Size: CGSize{Width: 130, Height: 18}})
	content.Send(sel("addSubview:"), status)

	button := objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString("打开系统设置…"), settingsTarget, sel(action))
	button.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: tabContentW - 160, Y: y - 5}, Size: CGSize{Width: 136, Height: 28}})
	button.Send(sel("setAccessibilityLabel:"), nsString("打开"+title+"系统设置"))
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
	text, color := "未授权", "systemRedColor"
	if granted {
		text, color = "已授权", "systemGreenColor"
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

func (v *SettingsView) newTab(title string) objc.ID {
	item := objc.ID(class("NSTabViewItem")).Send(sel("alloc")).Send(sel("initWithIdentifier:"), nsString(title))
	item.Send(sel("setLabel:"), nsString(title))
	content := objc.ID(class("NSView")).Send(sel("new"))
	content.Send(sel("setFrame:"), CGRect{Size: CGSize{Width: tabContentW, Height: tabContentH}})
	item.Send(sel("setView:"), content)
	v.tabs.Send(sel("addTabViewItem:"), item)
	item.Send(sel("release"))
	content.Send(sel("release"))
	return content
}

func (v *SettingsView) addSwitch(content objc.ID, y float64, title, path string, get func(SettingsValues) bool) {
	v.addLabel(content, y, title)
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
			parent, ok := OpenDataDirectoryPanel("选择 EcoPaste 数据目录")
			if !ok {
				return
			}
			if !Confirm("更改数据目录？",
				"历史、设置和资源会复制到所选目录，然后重启 EcoPaste。旧目录数据会保留。", "迁移并重启") {
				return
			}
			v.cx.Emit("change", SettingsChange{Path: "general.dataDir", Value: parent})
		}},
		{Cmd: sel("defaultDataDirPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			v := settingsViews[sender]
			if v == nil {
				return
			}
			if !Confirm("恢复默认数据目录？",
				"历史、设置和资源会复制回默认目录，然后重启 EcoPaste。现有默认目录内容会保留为备份。", "迁移并重启") {
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
		{Cmd: sel("addAppPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			v := settingsViews[sender]
			if v == nil {
				return
			}
			path, ok := OpenApplicationPanel("选择要忽略来源的应用")
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
		{Cmd: sel("resetSettingsPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			v := settingsViews[sender]
			if v == nil {
				return
			}
			if !Confirm("恢复默认设置？",
				"所有偏好会恢复为默认值。剪贴板历史、数据目录与备份不受影响。", "恢复默认") {
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
			v.footer.Send(sel("setStringValue:"), nsString("更改会立即保存"))
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
	v.restore.Send(sel("setEnabled:"), values.CanRestoreDataDir)
	v.applyApps(values)
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
	titles := []string{"保持当前", "全部分组"}
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
		text = "不限"
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
	if v.permissionTimer != 0 {
		v.permissionTimer.Send(sel("invalidate"))
		delete(settingsTimers, v.permissionTimer)
		v.permissionTimer = 0
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
	for control, owner := range settingsAppControls {
		if owner == v {
			delete(settingsAppControls, control)
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
