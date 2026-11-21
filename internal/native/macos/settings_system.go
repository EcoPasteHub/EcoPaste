//go:build darwin

package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

func (v *SettingsView) buildSystemPage() {
	document := v.newPage("system")
	top := 18.0

	top = v.buildPermissionsSection(document, top)
	v.finishPage(document)
}

func (v *SettingsView) buildPermissionsSection(document objc.ID, top float64) float64 {
	v.addSectionLabel(document, top, i18n.SettingsPermissionsSystem)
	v.addInfoAfterText(document, settingsContentX+4, top+8, i18n.SettingsPermissionsSystem,
		settingsSectionFont(), i18n.SettingsPermissionsIntro)
	cardTop := top + 24
	addSettingsListCard(document, CGRect{Origin: CGPoint{X: settingsContentX, Y: cardTop},
		Size: CGSize{Width: settingsContentWidth, Height: 2 * settingsPermissionRowHeight}})
	v.addCardSeparatorOffset(document, cardTop, settingsPermissionRowHeight)
	v.accessibilityStatus = v.addPermission(document, cardTop,
		i18n.SettingsAccessibility, i18n.SettingsAccessibilityDetail,
		"accessibilityPermissionPressed:")
	v.fullDiskStatus = v.addPermission(document, cardTop+settingsPermissionRowHeight,
		i18n.SettingsFullDisk, i18n.SettingsFullDiskDetail,
		"fullDiskAccessPermissionPressed:")
	top = cardTop + 2*settingsPermissionRowHeight + settingsGroupGap

	v.addSectionLabel(document, top, i18n.SettingsDiagnostics)
	top += 24
	addSettingsListCard(document, CGRect{Origin: CGPoint{X: settingsContentX, Y: top},
		Size: CGSize{Width: settingsContentWidth, Height: 82}})
	versionLabel := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(diagnosticVersion()))
	versionLabel.Send(sel("setFont:"), systemFont(11))
	versionLabel.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	versionLabel.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: settingsContentX + settingsRowInset, Y: top + 12},
		Size: CGSize{Width: settingsContentWidth - 2*settingsRowInset, Height: 15}})
	document.Send(sel("addSubview:"), versionLabel)

	systemLabel := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(diagnosticSystemVersion()))
	systemLabel.Send(sel("setFont:"), systemFont(11))
	systemLabel.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	systemLabel.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: settingsContentX + settingsRowInset, Y: top + 30},
		Size: CGSize{Width: settingsContentWidth - 2*settingsRowInset, Height: 15}})
	document.Send(sel("addSubview:"), systemLabel)

	button := v.addPageButton(document, settingsContentX+settingsRowInset, top+49, 128, 25,
		i18n.SettingsCopyDiagnostics, "copyDiagnosticsPressed:", i18n.SettingsCopyDiagnosticsHelp)
	button.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.SettingsCopyDiagnostics)))
	v.bindText(func() {
		button.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.SettingsCopyDiagnostics)))
	})
	v.diagnosticsHint = objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
	v.diagnosticsHint.Send(sel("setFont:"), systemFont(11))
	v.diagnosticsHint.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	v.diagnosticsHint.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: settingsContentX + settingsRowInset + 140, Y: top + 54},
		Size: CGSize{Width: settingsContentWidth - 2*settingsRowInset - 140, Height: 15}})
	document.Send(sel("addSubview:"), v.diagnosticsHint)

	v.refreshPermissionLabels()
	top += 82 + settingsGroupGap

	_, cardTop = v.addSectionCard(document, top, i18n.SettingsLogging, 2)
	v.addPopup(document, settingsRowTop(cardTop, 0), i18n.SettingsLogLevel, "logging.level",
		[]settingsOption{
			{Key: i18n.SettingsLogLevelAll, Value: "all"},
			{Key: i18n.SettingsLogLevelError, Value: "error"},
			{Key: i18n.SettingsLogLevelWarn, Value: "warn"},
			{Key: i18n.SettingsLogLevelInfo, Value: "info"},
			{Key: i18n.SettingsLogLevelDebug, Value: "debug"},
		}, func(s SettingsValues) string { return s.LogLevel })
	logsButton := v.addPageButton(document, settingsContentX+settingsRowInset, settingsRowTop(cardTop, 1)+7,
		148, 28, i18n.SettingsOpenLogDirectory, "openLogsPressed:", i18n.SettingsOpenLogDirectoryHelp)
	logsButton.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.SettingsOpenLogDirectory)))
	v.bindText(func() {
		logsButton.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.SettingsOpenLogDirectory)))
	})
	return cardTop + 2*settingsRowHeight + settingsGroupGap
}
