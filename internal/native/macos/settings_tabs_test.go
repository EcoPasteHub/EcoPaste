//go:build darwin

package macos

import "testing"

func TestSettingsTabsHaveMetadata(t *testing.T) {
	for name, tab := range settingsTabs {
		if tab.title == "" || tab.symbol == "" {
			t.Fatalf("tab %q has incomplete metadata: %+v", name, tab)
		}
	}
}
