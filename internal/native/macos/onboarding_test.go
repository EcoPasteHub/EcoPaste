//go:build darwin

package macos

import "testing"

func TestOnboardingOpensSettingsTabForActionStep(t *testing.T) {
	cases := map[int]string{
		0: "general",
		1: "general",
		2: "shortcuts",
		3: "apps",
		4: "general",
	}
	for step, want := range cases {
		if got := onboardingSettingsTab(step); got != want {
			t.Errorf("step %d tab = %q, want %q", step, got, want)
		}
	}
}

func TestSelectSettingsTabRejectsUnknownName(t *testing.T) {
	if SelectSettingsTab("missing") {
		t.Error("unknown tab accepted")
	}
	for name := range settingsTabIdentifiers {
		if !SelectSettingsTab(name) {
			t.Errorf("tab %q rejected", name)
		}
	}
}
