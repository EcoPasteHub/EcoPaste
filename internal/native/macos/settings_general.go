//go:build darwin

package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

func (v *SettingsView) buildGeneralPage() {
	document := v.newPage("general")
	top := 18.0

	_, cardTop := v.addSectionCard(document, top, i18n.SettingsStartup, 3)
	v.addSwitch(document, settingsRowTop(cardTop, 0), i18n.SettingsAutoStart, "general.autoStart",
		func(s SettingsValues) bool { return s.AutoStart })
	v.addShortcutField(document, settingsRowTop(cardTop, 1), i18n.SettingsOpenClipboard, "shortcuts.openClipboard",
		func(s SettingsValues) string { return s.OpenClipboardShortcut })
	v.addShortcutField(document, settingsRowTop(cardTop, 2), i18n.SettingsOpenPreference, "shortcuts.openPreference",
		func(s SettingsValues) string { return s.OpenPreferenceShortcut })
	top = cardTop + 3*settingsRowHeight + settingsGroupGap

	_, cardTop = v.addSectionCard(document, top, i18n.SettingsAppearanceGroup, 4)
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
	v.addSwitch(document, settingsRowTop(cardTop, 2), i18n.SettingsTrayIcon, "general.trayIcon",
		func(s SettingsValues) bool { return s.TrayIcon })
	v.addSwitch(document, settingsRowTop(cardTop, 3), i18n.SettingsDockIcon, "general.dockIcon",
		func(s SettingsValues) bool { return s.DockIcon })
	top = cardTop + 4*settingsRowHeight + settingsGroupGap

	_, cardTop = v.addSectionCard(document, top, i18n.SettingsBehavior, 5)
	v.addSwitch(document, settingsRowTop(cardTop, 0), i18n.SettingsCopyThenHide, "copyThenHideWindow",
		func(s SettingsValues) bool { return s.CopyThenHideWindow })
	v.addSwitch(document, settingsRowTop(cardTop, 1), i18n.SettingsCopySound, "feedback.copySound",
		func(s SettingsValues) bool { return s.CopySound })
	v.addSwitch(document, settingsRowTop(cardTop, 2), i18n.SettingsUpdateOnReuse, "updateOnReuse",
		func(s SettingsValues) bool { return s.UpdateOnReuse })
	v.addPopup(document, settingsRowTop(cardTop, 3), i18n.SettingsSort, "sort",
		[]settingsOption{
			{Key: i18n.SettingsSortRecent, Value: "updatedAtDesc"},
			{Key: i18n.SettingsSortCreated, Value: "createdAtDesc"},
			{Key: i18n.SettingsSortUseCount, Value: "useCountDesc"},
		},
		func(s SettingsValues) string { return s.Sort })
	v.addPopup(document, settingsRowTop(cardTop, 4), i18n.SettingsPosition, "window.position",
		[]settingsOption{
			{Key: i18n.SettingsPositionFollow, Value: "followCursor"},
			{Key: i18n.SettingsPositionCenter, Value: "center"},
			{Key: i18n.SettingsPositionKeep, Value: "remember"},
		},
		func(s SettingsValues) string { return s.WindowPosition })
	top = cardTop + 5*settingsRowHeight + settingsGroupGap

	top = v.addPageHint(document, top, 32, i18n.SettingsShortcutRule)
	v.finishPage(document, top)
}

func (v *SettingsView) addSectionLabel(content objc.ID, y float64, title i18n.Key) {
	label := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(i18n.T(title)))
	label.Send(sel("setFont:"), objc.ID(classNSFont).Send(sel("systemFontOfSize:weight:"), 11.0, 0.3))
	label.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	label.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsContentX + 4, Y: y},
		Size:   CGSize{Width: settingsContentWidth - 8, Height: 16},
	})
	content.Send(sel("addSubview:"), label)
	v.bindString(label, title)
}
