//go:build darwin

package macos

import "testing"

func TestSettingsTabsCoverSidebarOrder(t *testing.T) {
	seen := make(map[string]struct{}, len(settingsTabOrder))
	for _, name := range settingsTabOrder {
		tab, ok := settingsTabs[name]
		if !ok {
			t.Fatalf("sidebar tab %q has no metadata", name)
		}
		if tab.title == "" || tab.symbol == "" {
			t.Fatalf("sidebar tab %q has incomplete metadata: %+v", name, tab)
		}
		seen[name] = struct{}{}
	}
	if len(seen) != len(settingsTabs) {
		t.Fatalf("metadata has %d tabs, sidebar order has %d", len(settingsTabs), len(seen))
	}
}
