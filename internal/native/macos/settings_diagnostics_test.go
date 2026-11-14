//go:build darwin

package macos

import (
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
)

func TestFormatDiagnosticReport(t *testing.T) {
	got := formatDiagnosticReport("EcoPaste 0.1.0", "macOS Version 26.1 (Build 25B78)",
		"/tmp/ecopaste", true, false)
	want := "EcoPaste 0.1.0\nmacOS Version 26.1 (Build 25B78)\n数据目录: /tmp/ecopaste\n辅助功能: 已授权\n完全磁盘访问: 未授权"
	if got != want {
		t.Fatalf("report = %q, want %q", got, want)
	}
}

func TestPermissionText(t *testing.T) {
	if permissionText(true) != "已授权" || permissionText(false) != "未授权" {
		t.Fatalf("unexpected permission labels")
	}
}

func TestFormatDiagnosticReportInEnglish(t *testing.T) {
	i18n.SetLanguage("en-US")
	t.Cleanup(func() { i18n.SetLanguage("zh-CN") })

	got := formatDiagnosticReport("EcoPaste 0.1.0", "macOS Version 26.1",
		"/tmp/ecopaste", true, false)
	want := "EcoPaste 0.1.0\nmacOS Version 26.1\nData Folder: /tmp/ecopaste\nAccessibility: Granted\nFull Disk Access: Not Granted"
	if got != want {
		t.Fatalf("report = %q, want %q", got, want)
	}
}
