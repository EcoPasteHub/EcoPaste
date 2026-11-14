//go:build darwin

package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
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

	p.onboardingHeaderDivider = objc.ID(classNSBox).Send(sel("new"))
	p.onboardingHeaderDivider.Send(sel("setBoxType:"), 2)
	p.onboardingHeaderDivider.Send(sel("setTitlePosition:"), 0)
	view.Send(sel("addSubview:"), p.onboardingHeaderDivider)

	appIcon := objc.Send[objc.ID](
		objc.ID(class("NSApplication")).Send(sel("sharedApplication")), sel("applicationIconImage"))
	p.onboardingIcon = objc.ID(class("NSImageView")).Send(sel("new"))
	p.onboardingIcon.Send(sel("setImage:"), appIcon)
	p.onboardingIcon.Send(sel("setImageScaling:"), int64(3))
	p.onboardingIcon.Send(sel("setAccessibilityElement:"), false)
	view.Send(sel("addSubview:"), p.onboardingIcon)

	p.onboardingTitle = onboardingLabel(21, true)
	p.onboardingTitle.Send(sel("setMaximumNumberOfLines:"), int64(2))
	view.Send(sel("addSubview:"), p.onboardingTitle)

	p.onboardingDetail = onboardingLabel(13, false)
	p.onboardingDetail.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	p.onboardingDetail.Send(sel("setLineBreakMode:"), int64(0))
	p.onboardingDetail.Send(sel("setMaximumNumberOfLines:"), int64(2))
	view.Send(sel("addSubview:"), p.onboardingDetail)

	p.onboardingBody = onboardingLabel(13, false)
	p.onboardingBody.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	p.onboardingBody.Send(sel("setMaximumNumberOfLines:"), int64(6))
	view.Send(sel("addSubview:"), p.onboardingBody)

	p.onboardingAccessibility = onboardingLabel(12, false)
	p.onboardingFullDisk = onboardingLabel(12, false)
	view.Send(sel("addSubview:"), p.onboardingAccessibility)
	view.Send(sel("addSubview:"), p.onboardingFullDisk)

	p.onboardingAccessibilityButton = onboardingButton(i18n.T(i18n.OnboardingOpenSettings), "onboardingPermissionPressed:")
	p.onboardingAccessibilityButton.Send(sel("setTag:"), int64(0))
	p.onboardingFullDiskButton = onboardingButton(i18n.T(i18n.OnboardingOpenSettings), "onboardingPermissionPressed:")
	p.onboardingFullDiskButton.Send(sel("setTag:"), int64(1))
	view.Send(sel("addSubview:"), p.onboardingAccessibilityButton)
	view.Send(sel("addSubview:"), p.onboardingFullDiskButton)

	p.onboardingShortcutOpen = onboardingLabel(13, false)
	p.onboardingShortcutPreference = onboardingLabel(13, false)
	view.Send(sel("addSubview:"), p.onboardingShortcutOpen)
	view.Send(sel("addSubview:"), p.onboardingShortcutPreference)

	p.onboardingLegacyNormal = onboardingCheckbox(i18n.T(i18n.OnboardingLegacyNormal), "onboardingLegacyNormalPressed:")
	p.onboardingLegacyFavorite = onboardingCheckbox(i18n.T(i18n.OnboardingLegacyFavorite), "onboardingLegacyFavoritePressed:")
	p.onboardingLegacyNormalOn = true
	p.onboardingLegacyFavoriteOn = true
	p.onboardingLegacyNormal.Send(sel("setState:"), int64(1))
	p.onboardingLegacyFavorite.Send(sel("setState:"), int64(1))
	view.Send(sel("addSubview:"), p.onboardingLegacyNormal)
	view.Send(sel("addSubview:"), p.onboardingLegacyFavorite)

	p.onboardingNote = onboardingLabel(11, false)
	p.onboardingNote.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("tertiaryLabelColor")))
	p.onboardingNote.Send(sel("setMaximumNumberOfLines:"), int64(4))
	view.Send(sel("addSubview:"), p.onboardingNote)

	p.onboardingProgress = onboardingLabel(11, false)
	p.onboardingProgress.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("tertiaryLabelColor")))
	view.Send(sel("addSubview:"), p.onboardingProgress)

	p.onboardingProgressBar = objc.ID(class("NSProgressIndicator")).Send(sel("new"))
	p.onboardingProgressBar.Send(sel("setStyle:"), int64(0))
	p.onboardingProgressBar.Send(sel("setIndeterminate:"), false)
	p.onboardingProgressBar.Send(sel("setMinValue:"), 0.0)
	p.onboardingProgressBar.Send(sel("setMaxValue:"), float64(onboardingStepCount))
	p.onboardingProgressBar.Send(sel("setControlSize:"), int64(1))
	p.onboardingProgressBar.Send(sel("setAccessibilityElement:"), false)
	view.Send(sel("addSubview:"), p.onboardingProgressBar)

	p.onboardingFooterDivider = objc.ID(classNSBox).Send(sel("new"))
	p.onboardingFooterDivider.Send(sel("setBoxType:"), 2)
	p.onboardingFooterDivider.Send(sel("setTitlePosition:"), 0)
	view.Send(sel("addSubview:"), p.onboardingFooterDivider)

	p.onboardingBack = onboardingButton(i18n.T(i18n.OnboardingBack), "onboardingPrevPressed:")
	p.onboardingExtra = onboardingButton("", "onboardingExtraPressed:")
	p.onboardingNext = onboardingButton(i18n.T(i18n.OnboardingContinue), "onboardingNextPressed:")
	p.onboardingNext.Send(sel("setKeyEquivalent:"), nsString("\r"))
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
		label.Send(sel("setFont:"), objc.ID(classNSFont).Send(sel("systemFontOfSize:weight:"), size, 0.3))
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
	p.searchIcon.Send(sel("setHidden:"), active)
	p.pin.Send(sel("setHidden:"), active)
	p.filterButton.Send(sel("setHidden:"), active)
	p.settingsButton.Send(sel("setHidden:"), active)
	p.divider.Send(sel("setHidden:"), active)
	p.scroll.Send(sel("setHidden:"), active)
	p.empty.Send(sel("setHidden:"), active)
	p.footer.Send(sel("setHidden:"), active)
	p.hint.Send(sel("setHidden:"), active)
	syncNavVisibility(p)
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
		nsString(i18n.Tf(i18n.OnboardingProgress, step+1, onboardingStepCount)))
	p.onboardingProgressBar.Send(sel("setDoubleValue:"), float64(step+1))
	p.onboardingExtra.Send(sel("setHidden:"), true)
	p.onboardingBack.Send(sel("setHidden:"), step == 0)
	next := i18n.T(i18n.OnboardingContinue)
	if step == 0 {
		next = i18n.T(i18n.OnboardingStart)
	}
	if step == onboardingStepCount-1 {
		next = i18n.T(i18n.OnboardingStartUsing)
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
		p.onboardingTitle.Send(sel("setStringValue:"), nsString(i18n.T(i18n.OnboardingWelcomeTitle)))
		p.onboardingDetail.Send(sel("setStringValue:"), nsString(i18n.T(i18n.OnboardingWelcomeDetail)))
		p.onboardingBody.Send(sel("setStringValue:"), nsString(i18n.T(i18n.OnboardingWelcomeBody)))
	case 1:
		p.onboardingTitle.Send(sel("setStringValue:"), nsString(i18n.T(i18n.OnboardingPermissionTitle)))
		p.onboardingDetail.Send(sel("setStringValue:"), nsString(i18n.T(i18n.OnboardingPermissionDetail)))
		p.onboardingBody.Send(sel("setHidden:"), true)
		p.onboardingAccessibilityButton.Send(sel("setHidden:"), false)
		p.onboardingFullDiskButton.Send(sel("setHidden:"), false)
		p.onboardingAccessibility.Send(sel("setHidden:"), false)
		p.onboardingFullDisk.Send(sel("setHidden:"), false)
		p.onboardingNote.Send(sel("setHidden:"), false)
		p.onboardingNote.Send(sel("setStringValue:"), nsString(i18n.T(i18n.OnboardingPermissionNote)))
		p.refreshOnboardingPermissions()
	case 2:
		p.onboardingTitle.Send(sel("setStringValue:"), nsString(i18n.T(i18n.OnboardingShortcutTitle)))
		p.onboardingDetail.Send(sel("setStringValue:"), nsString(i18n.T(i18n.OnboardingShortcutDetail)))
		p.onboardingBody.Send(sel("setHidden:"), true)
		p.onboardingShortcutOpen.Send(sel("setHidden:"), false)
		p.onboardingShortcutPreference.Send(sel("setHidden:"), false)
		p.onboardingShortcutOpen.Send(sel("setStringValue:"),
			nsString(i18n.Tf(i18n.OnboardingShortcutOpen, onboardingShortcut(p.onboardingData.OpenShortcut))))
		p.onboardingShortcutPreference.Send(sel("setStringValue:"),
			nsString(i18n.Tf(i18n.OnboardingShortcutPreference, onboardingShortcut(p.onboardingData.PreferenceShortcut))))
		p.onboardingExtra.Send(sel("setTitle:"), nsString(i18n.T(i18n.OnboardingShortcutEdit)))
		p.onboardingExtra.Send(sel("setHidden:"), false)
		p.onboardingNote.Send(sel("setHidden:"), false)
		p.onboardingNote.Send(sel("setStringValue:"), nsString(i18n.T(i18n.OnboardingShortcutNote)))
	case 3:
		p.onboardingTitle.Send(sel("setStringValue:"), nsString(i18n.T(i18n.OnboardingExcludedTitle)))
		p.onboardingDetail.Send(sel("setStringValue:"), nsString(i18n.T(i18n.OnboardingExcludedDetail)))
		p.onboardingBody.Send(sel("setStringValue:"),
			nsString(i18n.Tf(i18n.OnboardingExcludedBody, p.onboardingData.ExcludedApps)))
		p.onboardingExtra.Send(sel("setTitle:"), nsString(i18n.T(i18n.OnboardingManageApps)))
		p.onboardingExtra.Send(sel("setHidden:"), false)
	case 4:
		p.applyLegacyText()
	case 5:
		p.onboardingTitle.Send(sel("setStringValue:"), nsString(i18n.T(i18n.OnboardingReadyTitle)))
		p.onboardingDetail.Send(sel("setStringValue:"), nsString(i18n.T(i18n.OnboardingReadyDetail)))
		p.onboardingBody.Send(sel("setStringValue:"), nsString(i18n.T(i18n.OnboardingReadyBody)))
	}
}

func (p *Panel) applyLegacyText() {
	legacy := p.onboardingData.Legacy
	p.onboardingTitle.Send(sel("setStringValue:"), nsString(i18n.T(i18n.OnboardingLegacyTitle)))
	p.onboardingDetail.Send(sel("setStringValue:"), nsString(i18n.T(i18n.OnboardingLegacyDetail)))
	p.onboardingBody.Send(sel("setHidden:"), false)
	p.onboardingNote.Send(sel("setHidden:"), false)
	p.onboardingExtra.Send(sel("setHidden:"), true)
	p.onboardingNext.Send(sel("setTitle:"), nsString(i18n.T(i18n.OnboardingContinue)))
	p.onboardingNext.Send(sel("setEnabled:"), true)

	showOptions := legacy.Found && legacy.ImportableItemCount > 0 && !legacy.Imported
	if showOptions {
		p.onboardingLegacyNormal.Send(sel("setTitle:"),
			nsString(i18n.Tf(i18n.OnboardingLegacyNormalCount, legacy.NormalItemCount)))
		p.onboardingLegacyFavorite.Send(sel("setTitle:"),
			nsString(i18n.Tf(i18n.OnboardingLegacyFavoriteCount, legacy.FavoriteItemCount)))
		p.onboardingLegacyNormal.Send(sel("setState:"), boolInt(p.onboardingLegacyNormalOn))
		p.onboardingLegacyFavorite.Send(sel("setState:"), boolInt(p.onboardingLegacyFavoriteOn))
	}

	switch {
	case legacy.Importing:
		p.onboardingBody.Send(sel("setStringValue:"), nsString(i18n.T(i18n.OnboardingLegacyImportingBody)))
		p.onboardingNote.Send(sel("setStringValue:"), nsString(i18n.T(i18n.OnboardingLegacyImportingNote)))
		p.onboardingNext.Send(sel("setTitle:"), nsString(i18n.T(i18n.OnboardingLegacyImportingButton)))
		p.onboardingNext.Send(sel("setEnabled:"), false)
	case legacy.Scanning:
		p.onboardingBody.Send(sel("setStringValue:"), nsString(i18n.T(i18n.OnboardingLegacyScanningBody)))
		p.onboardingNote.Send(sel("setStringValue:"), nsString(i18n.T(i18n.OnboardingLegacyScanningNote)))
		p.onboardingNext.Send(sel("setTitle:"), nsString(i18n.T(i18n.OnboardingLegacyScanningButton)))
		p.onboardingNext.Send(sel("setEnabled:"), false)
	case legacy.Imported:
		body := i18n.Tf(i18n.OnboardingLegacyImportedBody,
			legacy.ImportedNormal, legacy.ImportedFavorite)
		if legacy.Skipped > 0 {
			body += i18n.Tf(i18n.OnboardingLegacySkippedBody, legacy.Skipped)
		}
		p.onboardingBody.Send(sel("setStringValue:"), nsString(body))
		p.onboardingNote.Send(sel("setStringValue:"), nsString(i18n.T(i18n.OnboardingLegacyImportedNote)))
	case legacy.Error != "":
		p.onboardingBody.Send(sel("setStringValue:"), nsString(i18n.Tf(i18n.OnboardingLegacyErrorBody, legacy.Error)))
		p.onboardingNote.Send(sel("setStringValue:"), nsString(i18n.T(i18n.OnboardingLegacyErrorNote)))
		if showOptions {
			p.onboardingLegacyNormal.Send(sel("setHidden:"), false)
			p.onboardingLegacyFavorite.Send(sel("setHidden:"), false)
			p.onboardingExtra.Send(sel("setTitle:"), nsString(i18n.T(i18n.OnboardingLegacySkip)))
			p.onboardingExtra.Send(sel("setHidden:"), false)
			p.onboardingNext.Send(sel("setTitle:"), nsString(i18n.T(i18n.OnboardingLegacyRetry)))
		}
	case showOptions:
		p.onboardingBody.Send(sel("setStringValue:"),
			nsString(i18n.Tf(i18n.OnboardingLegacyFoundBody, legacy.Path, legacy.ImportableItemCount)))
		p.onboardingNote.Send(sel("setStringValue:"), nsString(i18n.T(i18n.OnboardingLegacyFoundNote)))
		p.onboardingLegacyNormal.Send(sel("setHidden:"), false)
		p.onboardingLegacyFavorite.Send(sel("setHidden:"), false)
		p.onboardingExtra.Send(sel("setTitle:"), nsString(i18n.T(i18n.OnboardingLegacySkip)))
		p.onboardingExtra.Send(sel("setHidden:"), false)
		p.onboardingNext.Send(sel("setTitle:"), nsString(i18n.T(i18n.OnboardingLegacyImport)))
		p.updateLegacyNext()
	case legacy.Found:
		p.onboardingBody.Send(sel("setStringValue:"), nsString(i18n.T(i18n.OnboardingLegacyEmptyBody)))
		p.onboardingNote.Send(sel("setStringValue:"), nsString(i18n.T(i18n.OnboardingLegacyEmptyNote)))
	default:
		p.onboardingBody.Send(sel("setStringValue:"), nsString(i18n.T(i18n.OnboardingLegacyMissingBody)))
		p.onboardingNote.Send(sel("setStringValue:"), nsString(i18n.T(i18n.OnboardingLegacyMissingNote)))
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
		return i18n.T(i18n.OnboardingShortcutNotSet)
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
		nsString(i18n.T(i18n.OnboardingAccessibility)+"    "+onboardingPermissionText(accessibility)))
	p.onboardingFullDisk.Send(sel("setStringValue:"),
		nsString(i18n.T(i18n.OnboardingFullDisk)+"    "+onboardingPermissionText(fullDisk)))
	p.onboardingAccessibilityButton.Send(sel("setHidden:"), accessibility)
	p.onboardingFullDiskButton.Send(sel("setHidden:"), fullDisk)
}

func onboardingPermissionText(granted bool) string {
	if granted {
		return i18n.T(i18n.OnboardingGranted)
	}
	return i18n.T(i18n.OnboardingNotGranted)
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
	setFrame(p.onboardingIcon, pad, height-pad-52, 52, 52)
	setFrame(p.onboardingTitle, pad+64, height-pad-29, width-pad*2-64, 28)
	setFrame(p.onboardingDetail, pad+64, height-pad-67, width-pad*2-64, 34)
	setFrame(p.onboardingHeaderDivider, pad, height-pad-76, width-2*pad, 1)
	if p.onboardingStep == onboardingLegacyStep {
		setFrame(p.onboardingBody, pad, 330, width-2*pad, 90)
	} else {
		setFrame(p.onboardingBody, pad, 330, width-2*pad, 115)
	}

	rowWidth := width - 2*pad - 116
	setFrame(p.onboardingAccessibility, pad, 392, rowWidth, 22)
	setFrame(p.onboardingFullDisk, pad, 350, rowWidth, 22)
	setFrame(p.onboardingAccessibilityButton, width-pad-104, 386, 104, 28)
	setFrame(p.onboardingFullDiskButton, width-pad-104, 344, 104, 28)
	setFrame(p.onboardingNote, pad, 266, width-2*pad, 68)

	setFrame(p.onboardingShortcutOpen, pad, 392, width-2*pad, 22)
	setFrame(p.onboardingShortcutPreference, pad, 358, width-2*pad, 22)
	setFrame(p.onboardingLegacyNormal, pad+2, 282, width-2*pad, 24)
	setFrame(p.onboardingLegacyFavorite, pad+2, 252, width-2*pad, 24)

	setFrame(p.onboardingProgressBar, pad, 88, 112, 8)
	setFrame(p.onboardingProgress, pad+120, 82, 140, 18)
	setFrame(p.onboardingFooterDivider, pad, 76, width-2*pad, 1)
	setFrame(p.onboardingNext, width-pad-110, 24, 110, buttonHeight)
	setFrame(p.onboardingBack, width-pad-110-8-90, 24, 90, buttonHeight)
	setFrame(p.onboardingExtra, width-pad-110-8-90-8-110, 24, 110, buttonHeight)
}
