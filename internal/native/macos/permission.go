//go:build darwin

package macos

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

const (
	accessibilitySettingsURL  = "x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility"
	fullDiskAccessSettingsURL = "x-apple.systempreferences:com.apple.preference.security?Privacy_AllFiles"
)

var (
	accessibilityCheckOnce sync.Once
	axIsProcessTrusted     func() bool
)

// AccessibilityGranted reports whether macOS allows the app to post input
// events, which automatic paste needs.
func AccessibilityGranted() bool {
	accessibilityCheckOnce.Do(func() {
		services, err := purego.Dlopen("/System/Library/Frameworks/ApplicationServices.framework/ApplicationServices",
			purego.RTLD_GLOBAL|purego.RTLD_NOW)
		if err != nil {
			return
		}
		purego.RegisterLibFunc(&axIsProcessTrusted, services, "AXIsProcessTrusted")
	})
	return axIsProcessTrusted != nil && axIsProcessTrusted()
}

// FullDiskAccessGranted probes a TCC-protected database; reading it only
// succeeds after the user grants full disk access.
func FullDiskAccessGranted() bool {
	paths := []string{"/Library/Application Support/com.apple.TCC/TCC.db"}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, "Library/Application Support/com.apple.TCC/TCC.db"))
	}
	return fullDiskAccessGranted(paths, probeReadable)
}

func fullDiskAccessGranted(paths []string, probe func(string) error) bool {
	for _, path := range paths {
		if probe(path) == nil {
			return true
		}
	}
	return false
}

func probeReadable(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	return file.Close()
}

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
		OpenAccessibilitySettings()
	}
}

// OpenAccessibilitySettings opens the privacy pane for accessibility.
func OpenAccessibilitySettings() {
	openSettingsURL(accessibilitySettingsURL)
}

// OpenFullDiskAccessSettings opens the privacy pane for full disk access.
func OpenFullDiskAccessSettings() {
	openSettingsURL(fullDiskAccessSettingsURL)
}

func openSettingsURL(raw string) {
	workspace := objc.ID(class("NSWorkspace")).Send(sel("sharedWorkspace"))
	url := objc.ID(class("NSURL")).Send(sel("URLWithString:"), nsString(raw))
	workspace.Send(sel("openURL:"), url)
}
