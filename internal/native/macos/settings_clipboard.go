//go:build darwin

package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

func (v *SettingsView) buildClipboardPage() {
	document := v.newPage("clipboard")
	top := 18.0

	_, cardTop := v.addSectionCard(document, top, i18n.SettingsCaptureTypes, 5)
	v.addSwitch(document, settingsRowTop(cardTop, 0), i18n.SettingsCaptureText, "capture.text",
		func(s SettingsValues) bool { return s.CaptureText })
	v.addSwitch(document, settingsRowTop(cardTop, 1), i18n.SettingsCaptureHTML, "capture.html",
		func(s SettingsValues) bool { return s.CaptureHTML })
	v.addSwitch(document, settingsRowTop(cardTop, 2), i18n.SettingsCaptureRTF, "capture.rtf",
		func(s SettingsValues) bool { return s.CaptureRTF })
	v.addSwitch(document, settingsRowTop(cardTop, 3), i18n.KindImage, "capture.image",
		func(s SettingsValues) bool { return s.CaptureImage })
	v.addSwitch(document, settingsRowTop(cardTop, 4), i18n.KindFiles, "capture.files",
		func(s SettingsValues) bool { return s.CaptureFiles })
	top = cardTop + 5*settingsRowHeight + settingsGroupGap

	_, cardTop = v.addSectionCard(document, top, i18n.SettingsCaptureLimits, 2)
	v.addNumberInput(document, settingsRowTop(cardTop, 0), i18n.SettingsMaxText, "capture.maxTextMb",
		1, 1024, 1, " MB", func(s SettingsValues) uint32 { return s.MaxTextMB })
	v.addNumberInput(document, settingsRowTop(cardTop, 1), i18n.SettingsMaxImage, "capture.maxImageMb",
		10, 4096, 10, " MB", func(s SettingsValues) uint32 { return s.MaxImageMB })
	top = cardTop + 2*settingsRowHeight + settingsGroupGap
	top = v.buildCapturePriorities(document, top)
	top = v.buildSourceAppsSection(document, top)
	v.finishPage(document)
}

func (v *SettingsView) buildCapturePriorities(document objc.ID, top float64) float64 {
	registerSettingsCaptureOnce.Do(registerSettingsCaptureTargets)
	v.addSectionLabel(document, top, i18n.SettingsCaptureOrder)
	v.addInfoAfterText(document, settingsContentX+4, top+8, i18n.SettingsCaptureOrder,
		settingsSectionFont(), i18n.SettingsCaptureHint)
	top += 24

	const listHeight = 112.0
	card := CGRect{Origin: CGPoint{X: settingsContentX, Y: top},
		Size: CGSize{Width: settingsContentWidth - 104, Height: listHeight}}
	table := v.addSettingsListTable(document, card, settingsCaptureRowHeight,
		float64(card.Size.Width-4), "capture", i18n.SettingsCaptureOrder, settingsCaptureTarget)
	v.captureTable = table
	settingsCaptureByTable[table] = v

	buttonX := settingsContentX + settingsContentWidth - 90
	v.captureUp = v.addSettingsMoveButton(document,
		CGRect{Origin: CGPoint{X: buttonX, Y: top + 12}, Size: CGSize{Width: 90, Height: 28}},
		i18n.SettingsMoveUp, i18n.SettingsMoveUpA11y, settingsCaptureTarget, "captureUpPressed:")
	v.captureDown = v.addSettingsMoveButton(document,
		CGRect{Origin: CGPoint{X: buttonX, Y: top + 48}, Size: CGSize{Width: 90, Height: 28}},
		i18n.SettingsMoveDown, i18n.SettingsMoveDownA11y, settingsCaptureTarget, "captureDownPressed:")
	return top + listHeight + settingsGroupGap
}
