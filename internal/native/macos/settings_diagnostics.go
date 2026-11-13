//go:build darwin

package macos

import (
	"strings"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

func (v *SettingsView) buildDiagnostics(content objc.ID) {
	y := 232.0
	v.addLabel(content, y, i18n.T(i18n.SettingsDiagnostics))
	v.addSearchEntry(content, i18n.T(i18n.SettingsDiagnostics), "diagnostics.report",
		i18n.T(i18n.SettingsDiagnosticsDetail), i18n.T(i18n.SettingsDiagnosticsKeywords),
		rowHighlightFrame(y))

	version := diagnosticVersion()
	versionLabel := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(version))
	versionLabel.Send(sel("setFont:"), systemFont(11))
	versionLabel.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	versionLabel.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: y - 24}, Size: CGSize{Width: tabContentW - 48, Height: 16}})
	content.Send(sel("addSubview:"), versionLabel)

	systemLabel := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(diagnosticSystemVersion()))
	systemLabel.Send(sel("setFont:"), systemFont(11))
	systemLabel.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	systemLabel.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: y - 42}, Size: CGSize{Width: tabContentW - 48, Height: 16}})
	content.Send(sel("addSubview:"), systemLabel)

	v.diagnosticsDir = objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
	v.diagnosticsDir.Send(sel("setFont:"), systemFont(11))
	v.diagnosticsDir.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	v.diagnosticsDir.Send(sel("setLineBreakMode:"), int64(4))
	v.diagnosticsDir.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: y - 60}, Size: CGSize{Width: tabContentW - 48, Height: 16}})
	content.Send(sel("addSubview:"), v.diagnosticsDir)

	button := objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(i18n.T(i18n.SettingsCopyDiagnostics)), settingsTarget, sel("copyDiagnosticsPressed:"))
	button.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: y - 98}, Size: CGSize{Width: 128, Height: 28}})
	button.Send(sel("setBezelStyle:"), int64(1))
	button.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.SettingsCopyDiagnostics)))
	button.Send(sel("setAccessibilityHelp:"), nsString(i18n.T(i18n.SettingsCopyDiagnosticsHelp)))
	content.Send(sel("addSubview:"), button)
	settingsViews[button] = v

	v.diagnosticsHint = objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
	v.diagnosticsHint.Send(sel("setFont:"), systemFont(11))
	v.diagnosticsHint.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	v.diagnosticsHint.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 160, Y: y - 93}, Size: CGSize{Width: tabContentW - 184, Height: 16}})
	content.Send(sel("addSubview:"), v.diagnosticsHint)
}

func (v *SettingsView) applyDiagnostics(values SettingsValues) {
	if v.diagnosticsDir == 0 {
		return
	}
	text := i18n.T(i18n.SettingsDataDirLabel) + values.DataDir
	v.diagnosticsDir.Send(sel("setStringValue:"), nsString(text))
	v.diagnosticsDir.Send(sel("setToolTip:"), nsString(values.DataDir))
}

func (v *SettingsView) copyDiagnostics() {
	report := formatDiagnosticReport(diagnosticVersion(), diagnosticSystemVersion(),
		v.lastValues.DataDir, AccessibilityGranted(), FullDiskAccessGranted())
	if writePlainText(report) {
		v.diagnosticsHint.Send(sel("setStringValue:"), nsString(i18n.T(i18n.SettingsCopied)))
	} else {
		v.diagnosticsHint.Send(sel("setStringValue:"), nsString(i18n.T(i18n.SettingsCopyFailed)))
	}
}

func diagnosticVersion() string {
	version := AppVersion()
	if version == "" {
		return i18n.T(i18n.SettingsDevBuild)
	}
	return "EcoPaste " + version
}

func diagnosticSystemVersion() string {
	info := objc.ID(class("NSProcessInfo")).Send(sel("processInfo"))
	text := goString(info.Send(sel("operatingSystemVersionString")))
	if text == "" {
		return i18n.T(i18n.SettingsMacOSUnknown)
	}
	return "macOS " + strings.TrimPrefix(text, "Version ")
}

func formatDiagnosticReport(version, system, dataDir string, accessibility, fullDisk bool) string {
	return i18n.Tf(i18n.SettingsReport, version, system, dataDir,
		permissionText(accessibility), permissionText(fullDisk))
}

func permissionText(granted bool) string {
	if granted {
		return i18n.T(i18n.OnboardingGranted)
	}
	return i18n.T(i18n.OnboardingNotGranted)
}

func writePlainText(text string) bool {
	pasteboard := generalPasteboard()
	pasteboard.Send(sel("clearContents"))
	return pasteboard.Send(sel("setString:forType:"), nsString(text), nsString(TypeString)) != 0
}
