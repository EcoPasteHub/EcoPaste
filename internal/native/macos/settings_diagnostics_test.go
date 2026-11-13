//go:build darwin

package macos

import "testing"

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
