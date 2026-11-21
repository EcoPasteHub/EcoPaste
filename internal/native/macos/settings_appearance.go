//go:build darwin

package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
)

func (v *SettingsView) buildAppearancePage() {
	document := v.newPage("appearance")
	top := 18.0

	_, cardTop := v.addSectionCard(document, top, i18n.SettingsAppearanceGroup, 2)
	v.addPopup(document, settingsRowTop(cardTop, 0), i18n.SettingsTheme, "appearance.theme",
		[]settingsOption{
			{Key: i18n.SettingsThemeAuto, Value: "auto"},
			{Key: i18n.SettingsThemeLight, Value: "light"},
			{Key: i18n.SettingsThemeDark, Value: "dark"},
		},
		func(s SettingsValues) string { return s.Theme })
	v.addPopup(document, settingsRowTop(cardTop, 1), i18n.SettingsLanguage, "appearance.language",
		[]settingsOption{
			{Key: i18n.SettingsLanguageAuto, Value: "auto"},
			{Key: i18n.SettingsLanguageZh, Value: "zh-CN"},
			{Key: i18n.SettingsLanguageEn, Value: "en-US"},
		},
		func(s SettingsValues) string { return s.Language })
	top = cardTop + 2*settingsRowHeight + settingsGroupGap

	_, cardTop = v.addSectionCard(document, top, i18n.SettingsDisplayDensity, 3)
	v.addNumberInput(document, settingsRowTop(cardTop, 0), i18n.SettingsTextMaxLines, "display.textMaxLines",
		1, 5, 1, i18n.SettingsUnitLines,
		func(s SettingsValues) uint32 { return s.TextMaxLines })
	v.addNumberInput(document, settingsRowTop(cardTop, 1), i18n.SettingsImageMaxHeight, "display.imageMaxHeight",
		20, 100, 4, " px",
		func(s SettingsValues) uint32 { return s.ImageMaxHeight })
	v.addNumberInput(document, settingsRowTop(cardTop, 2), i18n.SettingsFileMaxCount, "display.fileMaxCount",
		1, 5, 1, i18n.SettingsUnitItems,
		func(s SettingsValues) uint32 { return s.FileMaxCount })
	top = cardTop + 3*settingsRowHeight + settingsGroupGap
	v.finishPage(document, top)
}
