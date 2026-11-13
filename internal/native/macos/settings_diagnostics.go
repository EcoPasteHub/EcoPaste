//go:build darwin

package macos

import (
	"fmt"
	"strings"

	"github.com/ebitengine/purego/objc"
)

func (v *SettingsView) buildDiagnostics(content objc.ID) {
	y := 232.0
	v.addLabel(content, y, "诊断信息")
	v.addSearchEntry(content, "诊断信息", "diagnostics.report",
		"复制版本、系统、数据目录与权限状态", "诊断 排查 支持",
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
		nsString("复制诊断信息"), settingsTarget, sel("copyDiagnosticsPressed:"))
	button.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: y - 98}, Size: CGSize{Width: 128, Height: 28}})
	button.Send(sel("setBezelStyle:"), int64(1))
	button.Send(sel("setAccessibilityLabel:"), nsString("复制诊断信息"))
	button.Send(sel("setAccessibilityHelp:"), nsString("把应用版本、系统版本、数据目录和权限状态复制到剪贴板"))
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
	text := "数据目录: " + values.DataDir
	v.diagnosticsDir.Send(sel("setStringValue:"), nsString(text))
	v.diagnosticsDir.Send(sel("setToolTip:"), nsString(values.DataDir))
}

func (v *SettingsView) copyDiagnostics() {
	report := formatDiagnosticReport(diagnosticVersion(), diagnosticSystemVersion(),
		v.lastValues.DataDir, AccessibilityGranted(), FullDiskAccessGranted())
	if writePlainText(report) {
		v.diagnosticsHint.Send(sel("setStringValue:"), nsString("已复制到剪贴板"))
	} else {
		v.diagnosticsHint.Send(sel("setStringValue:"), nsString("复制失败"))
	}
}

func diagnosticVersion() string {
	version := AppVersion()
	if version == "" {
		return "EcoPaste 开发版"
	}
	return "EcoPaste " + version
}

func diagnosticSystemVersion() string {
	info := objc.ID(class("NSProcessInfo")).Send(sel("processInfo"))
	text := goString(info.Send(sel("operatingSystemVersionString")))
	if text == "" {
		return "macOS 版本未知"
	}
	return "macOS " + strings.TrimPrefix(text, "Version ")
}

func formatDiagnosticReport(version, system, dataDir string, accessibility, fullDisk bool) string {
	return fmt.Sprintf("%s\n%s\n数据目录: %s\n辅助功能: %s\n完全磁盘访问: %s",
		version, system, dataDir,
		permissionText(accessibility), permissionText(fullDisk))
}

func permissionText(granted bool) string {
	if granted {
		return "已授权"
	}
	return "未授权"
}

func writePlainText(text string) bool {
	pasteboard := generalPasteboard()
	pasteboard.Send(sel("clearContents"))
	return pasteboard.Send(sel("setString:forType:"), nsString(text), nsString(TypeString)) != 0
}
