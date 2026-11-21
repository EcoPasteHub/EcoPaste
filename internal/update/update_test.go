package update

import (
	"errors"
	"strings"
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/nsgo-dev/nsgo/core"
)

func TestDescribeUpdateResult(t *testing.T) {
	i18n.SetLanguage("zh-CN")
	t.Cleanup(func() { i18n.SetLanguage("zh-CN") })

	tests := []struct {
		name        string
		result      core.UpdateCheck
		wantTitle   string
		wantMessage string
	}{
		{
			name:        "failure",
			result:      core.UpdateCheck{Err: errors.New("offline")},
			wantTitle:   i18n.T(i18n.UpdateCheckFailedTitle),
			wantMessage: i18n.T(i18n.UpdateCheckFailedBody),
		},
		{
			name:        "up to date",
			result:      core.UpdateCheck{Current: "2.0.0"},
			wantTitle:   i18n.T(i18n.UpdateUpToDateTitle),
			wantMessage: "2.0.0",
		},
		{
			name:        "available",
			result:      core.UpdateCheck{Current: "2.0.0", Version: "2.1.0"},
			wantTitle:   i18n.T(i18n.UpdateAvailableTitle),
			wantMessage: "2.1.0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title, message := Describe(tt.result, "2.0.0")
			if title != tt.wantTitle {
				t.Errorf("title = %q, want %q", title, tt.wantTitle)
			}
			if !strings.Contains(message, tt.wantMessage) {
				t.Errorf("message = %q, want it to contain %q", message, tt.wantMessage)
			}
		})
	}
}
