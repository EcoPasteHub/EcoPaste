package app

import (
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/keystroke"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
)

func TestAutoPasteModes(t *testing.T) {
	cases := map[settings.AutoPaste]bool{
		settings.AutoPasteDisabled:         false,
		settings.AutoPasteSingleClickCopy:  false,
		settings.AutoPasteDoubleClickCopy:  false,
		settings.AutoPasteSingleClickPaste: true,
		settings.AutoPasteDoubleClickPaste: true,
	}
	for mode, want := range cases {
		if got := autoPaste(mode); got != want {
			t.Errorf("autoPaste(%q) = %v, want %v", mode, got, want)
		}
	}
}

func TestNeedsAccessibility(t *testing.T) {
	model := &historyModel{}
	if !model.NeedsAccessibility(keystroke.ErrPermission) {
		t.Error("permission error was not recognized")
	}
	if model.NeedsAccessibility(nil) {
		t.Error("nil error asked for permission")
	}
}
