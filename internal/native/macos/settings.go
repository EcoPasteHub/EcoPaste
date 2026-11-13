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
	CopyPlain            bool
	CopyThenHideWindow   bool
	PastePlain           bool
	UpdateOnReuse        bool
	Sort                 string
	CaptureText          bool
	CaptureHTML          bool
	CaptureRTF           bool
	CaptureImage         bool
	CaptureFiles         bool
	MaxTextMB            uint32
	MaxImageMB           uint32
	Retention            string
	MaxCount             uint32
	CleanupIntervalHours uint32
	CollectSecrets       bool
	RedactSecrets        bool
}

// SettingsChange is one native control's new value.
type SettingsChange struct {
	Path  string
	Value any
}

// SettingsView is the AppKit settings window content.
type SettingsView struct {
	cx     *core.NativeContext
	view   objc.ID
	tabs   objc.ID
	footer objc.ID
	fields []*settingsField
}

type settingsField struct {
	path       string
	kind       string
	control    objc.ID
	valueLabel objc.ID
	options    []string
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
	v.footer.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 20, Y: 8}, Size: CGSize{Width: settingsWidth - 40, Height: 16}})
	v.view.Send(sel("addSubview:"), v.footer)

	v.buildGeneral()
	v.buildCapture()
	v.buildHistory()
	return v
}

func (v *SettingsView) buildGeneral() {
	content := v.newTab("通用")
	y := 408.0
	v.addSwitch(content, y, "复制后隐藏面板", "copyThenHideWindow",
		func(s SettingsValues) bool { return s.CopyThenHideWindow })
	v.addSwitch(content, y-42, "复制文本为纯文本", "copyPlain",
		func(s SettingsValues) bool { return s.CopyPlain })
	v.addSwitch(content, y-84, "粘贴文本为纯文本", "pastePlain",
		func(s SettingsValues) bool { return s.PastePlain })
	v.addSwitch(content, y-126, "复用后刷新排序", "updateOnReuse",
		func(s SettingsValues) bool { return s.UpdateOnReuse })
	v.addPopup(content, y-168, "排序方式", "sort",
		[]settingsOption{
			{Title: "最近使用", Value: "updatedAtDesc"},
			{Title: "创建时间", Value: "createdAtDesc"},
			{Title: "使用次数", Value: "useCountDesc"},
		},
		func(s SettingsValues) string { return s.Sort })
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

func (v *SettingsView) addPopup(content objc.ID, y float64, title, path string, options []settingsOption, get func(SettingsValues) string) {
	v.addLabel(content, y, title)
	values := make([]string, 0, len(options))
	control := objc.ID(class("NSPopUpButton")).Send(sel("alloc")).Send(sel("initWithFrame:pullsDown:"),
		CGRect{Origin: CGPoint{X: tabContentW - 260, Y: y - 4}, Size: CGSize{Width: 240, Height: 28}}, false)
	for _, option := range options {
		values = append(values, option.Value)
		control.Send(sel("addItemWithTitle:"), nsString(option.Title))
	}
	control.Send(sel("setTarget:"), settingsTarget)
	control.Send(sel("setAction:"), sel("popupChanged:"))
	v.addField(content, control, &settingsField{
		path: path, kind: "popup", control: control, options: values,
		get: func(s SettingsValues) any { return get(s) },
	})
}

func (v *SettingsView) addStepper(content objc.ID, y float64, title, path string, step, maxValue float64, suffix string, get func(SettingsValues) uint32) {
	v.addLabel(content, y, title)
	valueLabel := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
	valueLabel.Send(sel("setFont:"), systemFont(12))
	valueLabel.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	valueLabel.Send(sel("setAlignment:"), int64(2))
	valueLabel.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: tabContentW - 210, Y: y}, Size: CGSize{Width: 120, Height: 18}})
	content.Send(sel("addSubview:"), valueLabel)

	control := objc.ID(class("NSStepper")).Send(sel("new"))
	control.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: tabContentW - 80, Y: y - 5}, Size: CGSize{Width: 19, Height: 27}})
	control.Send(sel("setMinValue:"), 0.0)
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
	}
}

// Command implements core.NativeComponent.
func (v *SettingsView) Command(string, any) {}

func (v *SettingsView) apply(values SettingsValues) {
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

func selectPopup(field *settingsField, value string) {
	for i, option := range field.options {
		if option == value {
			field.control.Send(sel("selectItemAtIndex:"), int64(i))
			return
		}
	}
	field.control.Send(sel("addItemWithTitle:"), nsString(value))
	field.options = append(field.options, value)
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
	for _, field := range v.fields {
		delete(settingsViews, field.control)
	}
	v.view.Send(sel("release"))
}
