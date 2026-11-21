//go:build darwin

package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

func (v *SettingsView) buildGeneralPage() {
	document := v.newPage("general")
	top := 18.0

	_, cardTop := v.addSectionCard(document, top, i18n.SettingsStartup, 1)
	v.addSwitch(document, settingsRowTop(cardTop, 0), i18n.SettingsAutoStart, "general.autoStart",
		func(s SettingsValues) bool { return s.AutoStart })
	top = cardTop + settingsRowHeight + settingsGroupGap

	_, cardTop = v.addSectionCard(document, top, i18n.SettingsMenuBarGroup, 1)
	v.addSwitch(document, settingsRowTop(cardTop, 0), i18n.SettingsTrayIcon, "general.trayIcon",
		func(s SettingsValues) bool { return s.TrayIcon })
	top = cardTop + settingsRowHeight + settingsGroupGap

	_, cardTop = v.addSectionCard(document, top, i18n.SettingsShortcutsGroup, 2)
	v.addShortcutField(document, settingsRowTop(cardTop, 0), i18n.SettingsOpenClipboard, "shortcuts.openClipboard",
		func(s SettingsValues) string { return s.OpenClipboardShortcut })
	v.addInfoAfterText(document, settingsContentX+settingsRowInset,
		settingsRowTop(cardTop, 0)+settingsRowHeight/2, i18n.SettingsOpenClipboard,
		systemFont(13), i18n.SettingsShortcutRule)
	v.addShortcutField(document, settingsRowTop(cardTop, 1), i18n.SettingsOpenPreference, "shortcuts.openPreference",
		func(s SettingsValues) string { return s.OpenPreferenceShortcut })
	v.addInfoAfterText(document, settingsContentX+settingsRowInset,
		settingsRowTop(cardTop, 1)+settingsRowHeight/2, i18n.SettingsOpenPreference,
		systemFont(13), i18n.SettingsShortcutRule)

	v.finishPage(document)
}

func settingsSectionFont() objc.ID {
	return objc.ID(classNSFont).Send(sel("systemFontOfSize:weight:"), 11.0, 0.3)
}

func (v *SettingsView) addSectionLabel(content objc.ID, y float64, title i18n.Key) {
	label := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(i18n.T(title)))
	label.Send(sel("setFont:"), settingsSectionFont())
	label.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	label.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsContentX + 4, Y: y},
		Size:   CGSize{Width: settingsContentWidth - 8, Height: 16},
	})
	content.Send(sel("addSubview:"), label)
	v.bindString(label, title)
}
