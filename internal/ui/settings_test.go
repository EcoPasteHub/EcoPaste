package ui

import (
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
)

func TestSettingsNavigationHasLocalizedNativePages(t *testing.T) {
	want := []string{"general", "appearance", "clipboard", "interaction", "data", "system", "about"}
	tabs := settingsTabs()
	if len(tabs) != len(want) {
		t.Fatalf("settings navigation has %d tabs; want %d", len(tabs), len(want))
	}
	for i, name := range want {
		if tabs[i].id != name {
			t.Errorf("settings tab %d = %q; want %q", i, tabs[i].id, name)
		}
		if title := macos.SettingsTabTitle(name); title == name {
			t.Errorf("settings tab %q has no localized native title", name)
		}
		if tabs[i].icon == nil {
			t.Errorf("settings tab %q has no icon", name)
		}
	}
}
