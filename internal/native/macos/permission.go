//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

const accessibilitySettingsURL = "x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility"

// showAccessibilityAlert explains why automatic paste is unavailable and
// offers to open the system setting that grants it.
func showAccessibilityAlert() {
	alert := objc.ID(class("NSAlert")).Send(sel("new"))
	defer alert.Send(sel("release"))
	alert.Send(sel("setMessageText:"), nsString("需要辅助功能权限"))
	alert.Send(sel("setInformativeText:"),
		nsString("EcoPaste 需要辅助功能权限才能自动粘贴。请在“系统设置 > 隐私与安全性 > 辅助功能”中允许 EcoPaste。"))
	alert.Send(sel("addButtonWithTitle:"), nsString("打开系统设置"))
	alert.Send(sel("addButtonWithTitle:"), nsString("稍后"))
	if alert.Send(sel("runModal")) == 1000 {
		openAccessibilitySettings()
	}
}

func openAccessibilitySettings() {
	workspace := objc.ID(class("NSWorkspace")).Send(sel("sharedWorkspace"))
	url := objc.ID(class("NSURL")).Send(sel("URLWithString:"), nsString(accessibilitySettingsURL))
	workspace.Send(sel("openURL:"), url)
}
