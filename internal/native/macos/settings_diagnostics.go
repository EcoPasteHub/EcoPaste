//go:build darwin

package macos

import (
	"strings"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

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
		return i18n.T(i18n.PermissionGranted)
	}
	return i18n.T(i18n.PermissionNotGranted)
}

func writePlainText(text string) bool {
	pasteboard := generalPasteboard()
	pasteboard.Send(sel("clearContents"))
	return pasteboard.Send(sel("setString:forType:"), nsString(text), nsString(TypeString)) != 0
}
