//go:build darwin

package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

func (v *SettingsView) buildGeneralPage() {
	document := v.newPage("general")
	top := 18.0

	_, cardTop := v.addSectionCard(document, top, i18n.T(i18n.SettingsStartup), 3)
	v.addSwitch(document, settingsRowTop(cardTop, 0), i18n.T(i18n.SettingsAutoStart), "general.autoStart",
		func(s SettingsValues) bool { return s.AutoStart })
	v.addShortcutField(document, settingsRowTop(cardTop, 1), i18n.T(i18n.SettingsOpenClipboard), "shortcuts.openClipboard",
		func(s SettingsValues) string { return s.OpenClipboardShortcut })
	v.addShortcutField(document, settingsRowTop(cardTop, 2), i18n.T(i18n.SettingsOpenPreference), "shortcuts.openPreference",
		func(s SettingsValues) string { return s.OpenPreferenceShortcut })
	top = cardTop + 3*settingsRowHeight + settingsGroupGap

	_, cardTop = v.addSectionCard(document, top, i18n.T(i18n.SettingsAppearanceGroup), 4)
	v.addPopup(document, settingsRowTop(cardTop, 0), i18n.T(i18n.SettingsTheme), "appearance.theme",
		[]settingsOption{
			{Title: i18n.T(i18n.SettingsThemeAuto), Value: "auto"},
			{Title: i18n.T(i18n.SettingsThemeLight), Value: "light"},
			{Title: i18n.T(i18n.SettingsThemeDark), Value: "dark"},
		},
		func(s SettingsValues) string { return s.Theme })
	v.addPopup(document, settingsRowTop(cardTop, 1), i18n.T(i18n.SettingsLanguage), "appearance.language",
		[]settingsOption{
			{Title: i18n.T(i18n.SettingsLanguageZh), Value: "zh-CN"},
			{Title: i18n.T(i18n.SettingsLanguageEn), Value: "en-US"},
		},
		func(s SettingsValues) string { return s.Language })
	v.addSwitch(document, settingsRowTop(cardTop, 2), i18n.T(i18n.SettingsTrayIcon), "general.trayIcon",
		func(s SettingsValues) bool { return s.TrayIcon })
	v.addSwitch(document, settingsRowTop(cardTop, 3), i18n.T(i18n.SettingsDockIcon), "general.dockIcon",
		func(s SettingsValues) bool { return s.DockIcon })
	top = cardTop + 4*settingsRowHeight + settingsGroupGap

	_, cardTop = v.addSectionCard(document, top, i18n.T(i18n.SettingsBehavior), 5)
	v.addSwitch(document, settingsRowTop(cardTop, 0), i18n.T(i18n.SettingsCopyThenHide), "copyThenHideWindow",
		func(s SettingsValues) bool { return s.CopyThenHideWindow })
	v.addSwitch(document, settingsRowTop(cardTop, 1), i18n.T(i18n.SettingsCopySound), "feedback.copySound",
		func(s SettingsValues) bool { return s.CopySound })
	v.addSwitch(document, settingsRowTop(cardTop, 2), i18n.T(i18n.SettingsUpdateOnReuse), "updateOnReuse",
		func(s SettingsValues) bool { return s.UpdateOnReuse })
	v.addPopup(document, settingsRowTop(cardTop, 3), i18n.T(i18n.SettingsSort), "sort",
		[]settingsOption{
			{Title: i18n.T(i18n.SettingsSortRecent), Value: "updatedAtDesc"},
			{Title: i18n.T(i18n.SettingsSortCreated), Value: "createdAtDesc"},
			{Title: i18n.T(i18n.SettingsSortUseCount), Value: "useCountDesc"},
		},
		func(s SettingsValues) string { return s.Sort })
	v.addPopup(document, settingsRowTop(cardTop, 4), i18n.T(i18n.SettingsPosition), "window.position",
		[]settingsOption{
			{Title: i18n.T(i18n.SettingsPositionFollow), Value: "followCursor"},
			{Title: i18n.T(i18n.SettingsPositionCenter), Value: "center"},
			{Title: i18n.T(i18n.SettingsPositionKeep), Value: "remember"},
		},
		func(s SettingsValues) string { return s.WindowPosition })
	top = cardTop + 5*settingsRowHeight + settingsGroupGap

	top = v.addPageHint(document, top, 32, i18n.T(i18n.SettingsShortcutRule))
	v.finishPage(document, top)
}

func (v *SettingsView) addSectionLabel(content objc.ID, y float64, title string) {
	label := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(title))
	label.Send(sel("setFont:"), objc.ID(classNSFont).Send(sel("systemFontOfSize:weight:"), 11.0, 0.3))
	label.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	label.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsContentX + 4, Y: y},
		Size:   CGSize{Width: settingsContentWidth - 8, Height: 16},
	})
	content.Send(sel("addSubview:"), label)
}
