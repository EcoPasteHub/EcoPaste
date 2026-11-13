//go:build darwin

package macos

import (
	"fmt"

	"github.com/ebitengine/purego/objc"
)

// PanelOnboarding is the first-run flow shown inside the clipboard panel.
type PanelOnboarding struct {
	Active             bool
	Step               int
	OpenShortcut       string
	PreferenceShortcut string
	ExcludedApps       int
	Legacy             PanelLegacy
}

// PanelLegacy is the v0.6 import step's detected state.
type PanelLegacy struct {
	Scanning            bool
	Importing           bool
	Found               bool
	Path                string
	ImportableItemCount int
	NormalItemCount     int
	FavoriteItemCount   int
	Imported            bool
	ImportedNormal      int
	ImportedFavorite    int
	Skipped             int
	Error               string
}

const (
	onboardingStepCount  = 6
	onboardingLegacyStep = 4
	onboardingDoneStep   = 5
)

var (
	onboardingViewClass objc.Class
	panelsByOnboarding  = map[objc.ID]*Panel{}
)

// (Cmd: onboardingPrevPressed) etc. is registered on the shared panel
// target; the buttons themselves carry no per-view state.

func (p *Panel) buildOnboarding() {
	view := objc.ID(onboardingViewClass).Send(sel("new"))
	view.Send(sel("setHidden:"), true)
	view.Send(sel("setMaterial:"), int64(12))
	view.Send(sel("setBlendingMode:"), int64(1))
	view.Send(sel("setState:"), int64(1))
	p.onboardingView = view

	p.onboardingTitle = onboardingLabel(20, true)
	p.onboardingTitle.Send(sel("setMaximumNumberOfLines:"), int64(2))
	view.Send(sel("addSubview:"), p.onboardingTitle)

	p.onboardingDetail = onboardingLabel(13, false)
	p.onboardingDetail.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	p.onboardingDetail.Send(sel("setMaximumNumberOfLines:"), int64(2))
	view.Send(sel("addSubview:"), p.onboardingDetail)

	p.onboardingBody = onboardingLabel(12, false)
	p.onboardingBody.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	p.onboardingBody.Send(sel("setMaximumNumberOfLines:"), int64(6))
	view.Send(sel("addSubview:"), p.onboardingBody)

	p.onboardingAccessibility = onboardingLabel(12, false)
	p.onboardingFullDisk = onboardingLabel(12, false)
	view.Send(sel("addSubview:"), p.onboardingAccessibility)
	view.Send(sel("addSubview:"), p.onboardingFullDisk)

	p.onboardingAccessibilityButton = onboardingButton("打开设置", "onboardingPermissionPressed:")
	p.onboardingAccessibilityButton.Send(sel("setTag:"), int64(0))
	p.onboardingFullDiskButton = onboardingButton("打开设置", "onboardingPermissionPressed:")
	p.onboardingFullDiskButton.Send(sel("setTag:"), int64(1))
	view.Send(sel("addSubview:"), p.onboardingAccessibilityButton)
	view.Send(sel("addSubview:"), p.onboardingFullDiskButton)

	p.onboardingShortcutOpen = onboardingLabel(13, false)
	p.onboardingShortcutPreference = onboardingLabel(13, false)
	view.Send(sel("addSubview:"), p.onboardingShortcutOpen)
	view.Send(sel("addSubview:"), p.onboardingShortcutPreference)

	p.onboardingLegacyNormal = onboardingCheckbox("普通记录", "onboardingLegacyNormalPressed:")
	p.onboardingLegacyFavorite = onboardingCheckbox("收藏记录", "onboardingLegacyFavoritePressed:")
	p.onboardingLegacyNormalOn = true
	p.onboardingLegacyFavoriteOn = true
	p.onboardingLegacyNormal.Send(sel("setState:"), int64(1))
	p.onboardingLegacyFavorite.Send(sel("setState:"), int64(1))
	view.Send(sel("addSubview:"), p.onboardingLegacyNormal)
	view.Send(sel("addSubview:"), p.onboardingLegacyFavorite)

	p.onboardingNote = onboardingLabel(11, false)
	p.onboardingNote.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("tertiaryLabelColor")))
	p.onboardingNote.Send(sel("setMaximumNumberOfLines:"), int64(3))
	view.Send(sel("addSubview:"), p.onboardingNote)

	p.onboardingProgress = onboardingLabel(11, false)
	p.onboardingProgress.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("tertiaryLabelColor")))
	view.Send(sel("addSubview:"), p.onboardingProgress)

	p.onboardingBack = onboardingButton("上一步", "onboardingPrevPressed:")
	p.onboardingExtra = onboardingButton("", "onboardingExtraPressed:")
	p.onboardingNext = onboardingButton("继续", "onboardingNextPressed:")
	view.Send(sel("addSubview:"), p.onboardingBack)
	view.Send(sel("addSubview:"), p.onboardingExtra)
	view.Send(sel("addSubview:"), p.onboardingNext)

	for _, button := range []objc.ID{
		p.onboardingAccessibilityButton, p.onboardingFullDiskButton,
		p.onboardingLegacyNormal, p.onboardingLegacyFavorite,
		p.onboardingBack, p.onboardingExtra, p.onboardingNext,
	} {
		panelsByControl[button] = p
	}
	panelsByOnboarding[view] = p
	p.view.Send(sel("addSubview:"), view)
}

func onboardingLabel(size float64, bold bool) objc.ID {
	label := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
	if bold {
		label.Send(sel("setFont:"), objc.ID(classNSFont).Send(sel("boldSystemFontOfSize:"), size))
	} else {
		label.Send(sel("setFont:"), systemFont(size))
	}
	label.Send(sel("setLineBreakMode:"), int64(0))
	label.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("labelColor")))
	return label
}

func onboardingButton(title, action string) objc.ID {
	button := objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(title), searchTarget, sel(action))
	button.Send(sel("setBezelStyle:"), int64(1))
	button.Send(sel("setFont:"), systemFont(12))
	return button
}

func onboardingCheckbox(title, action string) objc.ID {
	button := objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(title), searchTarget, sel(action))
	button.Send(sel("setButtonType:"), int64(3))
	button.Send(sel("setFont:"), systemFont(12))
	return button
}

func (p *Panel) setOnboarding(data PanelOnboarding) {
	p.onboardingData = data
	active := data.Active
	step := data.Step
	if step < 0 {
		step = 0
	}
	if step >= onboardingStepCount {
		step = onboardingStepCount - 1
	}
	if active == p.onboardingActive && step == p.onboardingStep {
		p.applyOnboardingText()
		return
	}
	p.onboardingActive = active
	p.onboardingStep = step
	p.search.Send(sel("setHidden:"), active)
	p.pin.Send(sel("setHidden:"), active)
	p.filter.Send(sel("setHidden:"), active)
	p.groupsPopup.Send(sel("setHidden:"), active)
	p.groupActions.Send(sel("setHidden:"), active)
	p.divider.Send(sel("setHidden:"), active)
	p.scroll.Send(sel("setHidden:"), active)
	p.empty.Send(sel("setHidden:"), active)
	p.footer.Send(sel("setHidden:"), active)
	p.hint.Send(sel("setHidden:"), active)
	if active {
		p.hideHover()
	}
	p.onboardingView.Send(sel("setHidden:"), !active)
	if active {
		p.applyOnboardingText()
		if window := p.view.Send(sel("window")); window != 0 {
			window.Send(sel("makeFirstResponder:"), p.onboardingView)
		}
		return
	}
	p.updateStatus()
	p.makeSearchFirstResponder()
}

func (p *Panel) applyOnboardingText() {
	step := p.onboardingStep
	p.onboardingProgress.Send(sel("setStringValue:"),
		nsString(fmt.Sprintf("第 %d 步，共 %d 步", step+1, onboardingStepCount)))
	p.onboardingExtra.Send(sel("setHidden:"), true)
	p.onboardingBack.Send(sel("setHidden:"), step == 0)
	next := "继续"
	if step == 0 {
		next = "开始"
	}
	if step == onboardingStepCount-1 {
		next = "开始使用"
	}
	p.onboardingNext.Send(sel("setTitle:"), nsString(next))
	p.onboardingAccessibility.Send(sel("setHidden:"), true)
	p.onboardingFullDisk.Send(sel("setHidden:"), true)
	p.onboardingAccessibilityButton.Send(sel("setHidden:"), true)
	p.onboardingFullDiskButton.Send(sel("setHidden:"), true)
	p.onboardingShortcutOpen.Send(sel("setHidden:"), true)
	p.onboardingShortcutPreference.Send(sel("setHidden:"), true)
	p.onboardingLegacyNormal.Send(sel("setHidden:"), true)
	p.onboardingLegacyFavorite.Send(sel("setHidden:"), true)
	p.onboardingNext.Send(sel("setEnabled:"), true)
	p.onboardingBody.Send(sel("setHidden:"), false)
	p.onboardingNote.Send(sel("setHidden:"), true)

	switch step {
	case 0:
		p.onboardingTitle.Send(sel("setStringValue:"), nsString("欢迎使用 EcoPaste"))
		p.onboardingDetail.Send(sel("setStringValue:"), nsString("简单、原生、安全的 macOS 剪贴板管理器。"))
		p.onboardingBody.Send(sel("setStringValue:"), nsString(
			"EcoPaste 会自动记录你复制的文本、图片和文件。按快捷键随时打开面板，选中记录后回车即可粘贴。\n\n接下来的几步会帮你完成权限、快捷键和隐私设置。"))
	case 1:
		p.onboardingTitle.Send(sel("setStringValue:"), nsString("授予必要权限"))
		p.onboardingDetail.Send(sel("setStringValue:"), nsString("自动粘贴和完整读取需要两项系统权限。"))
		p.onboardingBody.Send(sel("setHidden:"), true)
		p.onboardingAccessibilityButton.Send(sel("setHidden:"), false)
		p.onboardingFullDiskButton.Send(sel("setHidden:"), false)
		p.onboardingAccessibility.Send(sel("setHidden:"), false)
		p.onboardingFullDisk.Send(sel("setHidden:"), false)
		p.onboardingNote.Send(sel("setHidden:"), false)
		p.onboardingNote.Send(sel("setStringValue:"), nsString(
			"没有任何权限也能浏览和复制历史；自动粘贴需要辅助功能权限，完整磁盘访问用于读取部分应用复制的文件。授权后返回本窗口，状态会自动刷新。"))
		p.refreshOnboardingPermissions()
	case 2:
		p.onboardingTitle.Send(sel("setStringValue:"), nsString("记住全局快捷键"))
		p.onboardingDetail.Send(sel("setStringValue:"), nsString("在任何应用里都能快速唤起 EcoPaste。"))
		p.onboardingBody.Send(sel("setHidden:"), true)
		p.onboardingShortcutOpen.Send(sel("setHidden:"), false)
		p.onboardingShortcutPreference.Send(sel("setHidden:"), false)
		p.onboardingShortcutOpen.Send(sel("setStringValue:"),
			nsString(fmt.Sprintf("打开剪贴板面板    %s", onboardingShortcut(p.onboardingData.OpenShortcut))))
		p.onboardingShortcutPreference.Send(sel("setStringValue:"),
			nsString(fmt.Sprintf("打开设置    %s", onboardingShortcut(p.onboardingData.PreferenceShortcut))))
		p.onboardingExtra.Send(sel("setTitle:"), nsString("修改快捷键"))
		p.onboardingExtra.Send(sel("setHidden:"), false)
		p.onboardingNote.Send(sel("setHidden:"), false)
		p.onboardingNote.Send(sel("setStringValue:"), nsString("快捷键可以随时在设置的“快捷键”页重新录制。"))
	case 3:
		p.onboardingTitle.Send(sel("setStringValue:"), nsString("排除敏感应用"))
		p.onboardingDetail.Send(sel("setStringValue:"), nsString("密码管理器和银行类应用默认不会被记录。"))
		p.onboardingBody.Send(sel("setStringValue:"), nsString(fmt.Sprintf(
			"当前已排除 %d 个应用。\n\n你可以在设置的应用页勾选更多来源，或取消默认排除。", p.onboardingData.ExcludedApps)))
		p.onboardingExtra.Send(sel("setTitle:"), nsString("管理应用"))
		p.onboardingExtra.Send(sel("setHidden:"), false)
	case 4:
		p.applyLegacyText()
	case 5:
		p.onboardingTitle.Send(sel("setStringValue:"), nsString("准备就绪"))
		p.onboardingDetail.Send(sel("setStringValue:"), nsString("EcoPaste 已在菜单栏运行。"))
		p.onboardingBody.Send(sel("setStringValue:"), nsString(
			"按快捷键打开面板，双击或回车即可粘贴到当前应用。\n\n如果自动粘贴没有生效，请回到上一步的权限设置检查辅助功能授权。"))
	}
}

func (p *Panel) applyLegacyText() {
	legacy := p.onboardingData.Legacy
	p.onboardingTitle.Send(sel("setStringValue:"), nsString("导入旧版数据"))
	p.onboardingDetail.Send(sel("setStringValue:"), nsString("把旧版 EcoPaste 的历史记录带到新版本。"))
	p.onboardingBody.Send(sel("setHidden:"), false)
	p.onboardingNote.Send(sel("setHidden:"), false)
	p.onboardingExtra.Send(sel("setHidden:"), true)
	p.onboardingNext.Send(sel("setTitle:"), nsString("继续"))
	p.onboardingNext.Send(sel("setEnabled:"), true)

	showOptions := legacy.Found && legacy.ImportableItemCount > 0 && !legacy.Imported
	if showOptions {
		p.onboardingLegacyNormal.Send(sel("setTitle:"),
			nsString(fmt.Sprintf("普通记录（%d 条）", legacy.NormalItemCount)))
		p.onboardingLegacyFavorite.Send(sel("setTitle:"),
			nsString(fmt.Sprintf("收藏记录（%d 条）", legacy.FavoriteItemCount)))
		p.onboardingLegacyNormal.Send(sel("setState:"), boolInt(p.onboardingLegacyNormalOn))
		p.onboardingLegacyFavorite.Send(sel("setState:"), boolInt(p.onboardingLegacyFavoriteOn))
	}

	switch {
	case legacy.Importing:
		p.onboardingBody.Send(sel("setStringValue:"), nsString("正在导入，请稍候…"))
		p.onboardingNote.Send(sel("setStringValue:"), nsString("导入过程中请保持 EcoPaste 运行。"))
		p.onboardingNext.Send(sel("setTitle:"), nsString("正在导入…"))
		p.onboardingNext.Send(sel("setEnabled:"), false)
	case legacy.Scanning:
		p.onboardingBody.Send(sel("setStringValue:"), nsString("正在扫描旧版数据…"))
		p.onboardingNote.Send(sel("setStringValue:"), nsString("只会读取旧版数据库，不会修改原数据。"))
		p.onboardingNext.Send(sel("setTitle:"), nsString("正在扫描…"))
		p.onboardingNext.Send(sel("setEnabled:"), false)
	case legacy.Imported:
		body := fmt.Sprintf("已导入 %d 条普通记录和 %d 条收藏记录。",
			legacy.ImportedNormal, legacy.ImportedFavorite)
		if legacy.Skipped > 0 {
			body += fmt.Sprintf("\n\n有 %d 条记录因重复或内容无效被跳过。", legacy.Skipped)
		}
		p.onboardingBody.Send(sel("setStringValue:"), nsString(body))
		p.onboardingNote.Send(sel("setStringValue:"), nsString("导入的记录会出现在剪贴板列表中，原有的分组与使用次数不会迁移。"))
	case legacy.Error != "":
		p.onboardingBody.Send(sel("setStringValue:"), nsString("导入没有完成："+legacy.Error))
		p.onboardingNote.Send(sel("setStringValue:"), nsString("可以重试，或跳过这一步稍后再导入。"))
		if showOptions {
			p.onboardingLegacyNormal.Send(sel("setHidden:"), false)
			p.onboardingLegacyFavorite.Send(sel("setHidden:"), false)
			p.onboardingExtra.Send(sel("setTitle:"), nsString("跳过"))
			p.onboardingExtra.Send(sel("setHidden:"), false)
			p.onboardingNext.Send(sel("setTitle:"), nsString("重试"))
		}
	case showOptions:
		p.onboardingBody.Send(sel("setStringValue:"), nsString(fmt.Sprintf(
			"在 %s 找到 %d 条可导入记录。", legacy.Path, legacy.ImportableItemCount)))
		p.onboardingNote.Send(sel("setStringValue:"), nsString(
			"导入会沿用当前采集与敏感内容设置；重复和无效记录会自动跳过。"))
		p.onboardingLegacyNormal.Send(sel("setHidden:"), false)
		p.onboardingLegacyFavorite.Send(sel("setHidden:"), false)
		p.onboardingExtra.Send(sel("setTitle:"), nsString("跳过"))
		p.onboardingExtra.Send(sel("setHidden:"), false)
		p.onboardingNext.Send(sel("setTitle:"), nsString("导入并继续"))
		p.updateLegacyNext()
	case legacy.Found:
		p.onboardingBody.Send(sel("setStringValue:"), nsString("找到旧版数据目录，但没有可导入的记录。"))
		p.onboardingNote.Send(sel("setStringValue:"), nsString("继续即可完成引导。"))
	default:
		p.onboardingBody.Send(sel("setStringValue:"), nsString("未发现旧版数据。"))
		p.onboardingNote.Send(sel("setStringValue:"), nsString("如果你是首次使用 EcoPaste，可以直接继续。"))
	}
}

func (p *Panel) updateLegacyNext() {
	if p.onboardingStep != onboardingLegacyStep {
		return
	}
	legacy := p.onboardingData.Legacy
	if legacy.Scanning || legacy.Importing {
		p.onboardingNext.Send(sel("setEnabled:"), false)
		return
	}
	enabled := true
	if legacy.Found && legacy.ImportableItemCount > 0 && !legacy.Imported {
		enabled = p.onboardingLegacyNormalOn || p.onboardingLegacyFavoriteOn
	}
	p.onboardingNext.Send(sel("setEnabled:"), enabled)
}

func onboardingShortcut(value string) string {
	if value == "" {
		return "未设置"
	}
	return value
}

func onboardingSettingsTab(step int) string {
	switch step {
	case 2:
		return "shortcuts"
	case 3:
		return "apps"
	default:
		return "general"
	}
}

func (p *Panel) refreshOnboardingPermissions() {
	if !p.onboardingActive || p.onboardingStep != 1 {
		return
	}
	accessibility := AccessibilityGranted()
	fullDisk := FullDiskAccessGranted()
	p.onboardingAccessibility.Send(sel("setStringValue:"),
		nsString("辅助功能    "+onboardingPermissionText(accessibility)))
	p.onboardingFullDisk.Send(sel("setStringValue:"),
		nsString("完全磁盘访问    "+onboardingPermissionText(fullDisk)))
	p.onboardingAccessibilityButton.Send(sel("setHidden:"), accessibility)
	p.onboardingFullDiskButton.Send(sel("setHidden:"), fullDisk)
}

func onboardingPermissionText(granted bool) string {
	if granted {
		return "已授权"
	}
	return "未授权"
}

func (p *Panel) onboardingAdvance() {
	if p.onboardingStep == onboardingLegacyStep {
		legacy := p.onboardingData.Legacy
		if !legacy.Imported && !legacy.Scanning && !legacy.Importing &&
			legacy.Found && legacy.ImportableItemCount > 0 {
			var types []string
			if p.onboardingLegacyNormalOn {
				types = append(types, "normal")
			}
			if p.onboardingLegacyFavoriteOn {
				types = append(types, "favorite")
			}
			if len(types) > 0 {
				p.cx.Emit("legacyImport", types)
			}
			return
		}
	}
	if p.onboardingStep >= onboardingDoneStep {
		p.cx.Emit("onboardingFinish", nil)
		return
	}
	p.onboardingStep++
	p.applyOnboardingText()
	p.cx.Emit("onboardingStep", p.onboardingStep)
}

func (p *Panel) onboardingPrev() {
	if p.onboardingStep == 0 {
		return
	}
	p.onboardingStep--
	p.applyOnboardingText()
	p.cx.Emit("onboardingStep", p.onboardingStep)
}

func (p *Panel) onboardingSkipLegacy() {
	if p.onboardingStep != onboardingLegacyStep {
		return
	}
	p.onboardingStep++
	p.applyOnboardingText()
	p.cx.Emit("onboardingStep", p.onboardingStep)
}

func (p *Panel) layoutOnboarding(width, height float64) {
	if p.onboardingView == 0 {
		return
	}
	const pad = 26.0
	const buttonHeight = 30.0
	setFrame(p.onboardingView, 0, 0, width, height)
	setFrame(p.onboardingTitle, pad, height-pad-34, width-2*pad, 34)
	setFrame(p.onboardingDetail, pad, height-pad-34-6-36, width-2*pad, 36)
	if p.onboardingStep == onboardingLegacyStep {
		setFrame(p.onboardingBody, pad, 300, width-2*pad, 130)
	} else {
		setFrame(p.onboardingBody, pad, 160, width-2*pad, 240)
	}

	rowWidth := width - 2*pad - 116
	setFrame(p.onboardingAccessibility, pad, 330, rowWidth, 22)
	setFrame(p.onboardingFullDisk, pad, 288, rowWidth, 22)
	setFrame(p.onboardingAccessibilityButton, width-pad-104, 324, 104, 28)
	setFrame(p.onboardingFullDiskButton, width-pad-104, 282, 104, 28)
	setFrame(p.onboardingNote, pad, 172, width-2*pad, 52)

	setFrame(p.onboardingShortcutOpen, pad, 330, width-2*pad, 22)
	setFrame(p.onboardingShortcutPreference, pad, 296, width-2*pad, 22)
	setFrame(p.onboardingLegacyNormal, pad+2, 262, width-2*pad, 24)
	setFrame(p.onboardingLegacyFavorite, pad+2, 232, width-2*pad, 24)

	setFrame(p.onboardingProgress, pad, 30, 120, 18)
	setFrame(p.onboardingNext, width-pad-110, 24, 110, buttonHeight)
	setFrame(p.onboardingBack, width-pad-110-8-90, 24, 90, buttonHeight)
	setFrame(p.onboardingExtra, width-pad-110-8-90-8-110, 24, 110, buttonHeight)
}
