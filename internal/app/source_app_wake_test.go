package app

import (
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/settings"
)

func TestSourceAppWakeBlocking(t *testing.T) {
	cfg := settings.Default().Clipboard
	cfg.SourceApps.Rules = []settings.SourceAppRule{
		{ID: "com.example.wake", Mode: settings.SourceAppBlockWake},
		{ID: "com.example.both", Mode: settings.SourceAppBlockCaptureAndWake},
		{ID: "com.example.capture", Mode: settings.SourceAppBlockCapture},
	}
	for _, id := range []string{"com.example.wake", "com.example.both"} {
		if !sourceAppWakeBlocked(cfg, id) {
			t.Errorf("%s should block wake", id)
		}
	}
	for _, id := range []string{"com.example.capture", "com.example.other", ""} {
		if sourceAppWakeBlocked(cfg, id) {
			t.Errorf("%s should allow wake", id)
		}
	}
}
