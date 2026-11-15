//go:build darwin

package macos

import "testing"

func TestSettingsTabsHaveMetadata(t *testing.T) {
	for name, tab := range settingsTabs {
		if tab.title == "" || tab.symbol == "" {
			t.Fatalf("tab %q has incomplete metadata: %+v", name, tab)
		}
		if canonical, ok := canonicalSettingsTab(name); !ok || canonical != name {
			t.Fatalf("tab %q does not resolve to itself", name)
		}
	}
}
