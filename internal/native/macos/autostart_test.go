//go:build darwin

package macos

import (
	"strings"
	"testing"
)

func TestAutostartNeedsChange(t *testing.T) {
	cases := []struct {
		name    string
		status  int64
		enabled bool
		want    bool
	}{
		{"disable not registered", appServiceNotRegistered, false, false},
		{"disable not found", appServiceNotFound, false, false},
		{"disable enabled", appServiceEnabled, false, true},
		{"disable needs approval", appServiceNeedsApproval, false, true},
		{"enable not registered", appServiceNotRegistered, true, true},
		{"enable not found", appServiceNotFound, true, true},
		{"enable enabled", appServiceEnabled, true, false},
		{"enable needs approval", appServiceNeedsApproval, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := autostartNeedsChange(tc.status, tc.enabled); got != tc.want {
				t.Errorf("autostartNeedsChange(%d, %t) = %t, want %t", tc.status, tc.enabled, got, tc.want)
			}
		})
	}
}

func TestAutostartErrorSuggestsApplicationsFolder(t *testing.T) {
	err := autostartError(appServiceNotFound, true)
	if err == nil || !strings.Contains(err.Error(), "应用程序") {
		t.Errorf("autostartError(NotFound, true) = %v", err)
	}
}
