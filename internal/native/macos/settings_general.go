//go:build darwin

package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

func (v *SettingsView) buildGeneral() {
	content := v.newTab("general", settingsTabTitle("general"))
	v.addSectionLabel(content, 438, i18n.T(i18n.SettingsBehavior))
	v.addSwitch(content, 408, i18n.T(i18n.SettingsDockIcon), "general.dockIcon",
		func(s SettingsValues) bool { return s.DockIcon })
	v.addSwitch(content, 374, i18n.T(i18n.SettingsTrayIcon), "general.trayIcon",
		func(s SettingsValues) bool { return s.TrayIcon })
	v.addSwitch(content, 340, i18n.T(i18n.SettingsAutoStart), "general.autoStart",
		func(s SettingsValues) bool { return s.AutoStart })
	v.addSwitch(content, 306, i18n.T(i18n.SettingsCopyThenHide), "copyThenHideWindow",
		func(s SettingsValues) bool { return s.CopyThenHideWindow })

	v.addSectionLabel(content, 264, i18n.T(i18n.SettingsClipboard))
	v.addSwitch(content, 234, i18n.T(i18n.SettingsCopySound), "feedback.copySound",
		func(s SettingsValues) bool { return s.CopySound })
	v.addSwitch(content, 200, i18n.T(i18n.SettingsUpdateOnReuse), "updateOnReuse",
		func(s SettingsValues) bool { return s.UpdateOnReuse })
	v.addPopup(content, 162, i18n.T(i18n.SettingsSort), "sort",
		[]settingsOption{
			{Title: i18n.T(i18n.SettingsSortRecent), Value: "updatedAtDesc"},
			{Title: i18n.T(i18n.SettingsSortCreated), Value: "createdAtDesc"},
			{Title: i18n.T(i18n.SettingsSortUseCount), Value: "useCountDesc"},
		},
		func(s SettingsValues) string { return s.Sort })

	v.addSectionLabel(content, 118, i18n.T(i18n.SettingsWindowAppearance))
	v.addPopup(content, 88, i18n.T(i18n.SettingsPosition), "window.position",
		[]settingsOption{
			{Title: i18n.T(i18n.SettingsPositionFollow), Value: "followCursor"},
			{Title: i18n.T(i18n.SettingsPositionCenter), Value: "center"},
			{Title: i18n.T(i18n.SettingsPositionKeep), Value: "remember"},
		},
		func(s SettingsValues) string { return s.WindowPosition })
	v.addPopup(content, 50, i18n.T(i18n.SettingsTheme), "appearance.theme",
		[]settingsOption{
			{Title: i18n.T(i18n.SettingsThemeAuto), Value: "auto"},
			{Title: i18n.T(i18n.SettingsThemeLight), Value: "light"},
			{Title: i18n.T(i18n.SettingsThemeDark), Value: "dark"},
		},
		func(s SettingsValues) string { return s.Theme })
	v.addPopup(content, 12, i18n.T(i18n.SettingsLanguage), "appearance.language",
		[]settingsOption{
			{Title: i18n.T(i18n.SettingsLanguageZh), Value: "zh-CN"},
			{Title: i18n.T(i18n.SettingsLanguageEn), Value: "en-US"},
		},
		func(s SettingsValues) string { return s.Language })
}

func (v *SettingsView) addSectionLabel(content objc.ID, y float64, title string) {
	label := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(title))
	label.Send(sel("setFont:"), objc.ID(classNSFont).Send(sel("systemFontOfSize:weight:"), 11.0, 0.3))
	label.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	label.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: y}, Size: CGSize{Width: tabContentW - 48, Height: 16}})
	content.Send(sel("addSubview:"), label)
}
