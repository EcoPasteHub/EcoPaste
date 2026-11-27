//go:build darwin

package macos

import "testing"

func TestSettingsAppsDiff(t *testing.T) {
	base := []SettingsApp{{ID: "a", Name: "A", Mode: "allowed"}, {ID: "b", Name: "B", Mode: "allowed"}}
	toggled := []SettingsApp{{ID: "a", Name: "A", Mode: "blockCapture"}, {ID: "b", Name: "B", Mode: "allowed"}}
	added := append(append([]SettingsApp(nil), base...), SettingsApp{ID: "c", Name: "C"})

	if settingsAppsEqual(base, toggled) {
		t.Fatal("mode change must not compare equal")
	}
	if !settingsAppsSameIdentity(base, toggled) {
		t.Fatal("mode change should keep the same identity")
	}
	if settingsAppsSameIdentity(base, added) {
		t.Fatal("added app should change identity")
	}
	rows := settingsAppChangedRows(base, toggled)
	if len(rows) != 1 || rows[0] != 0 {
		t.Fatalf("changed rows = %v; want [0]", rows)
	}
}

func TestStringHelpers(t *testing.T) {
	if !stringSlicesEqual([]string{"a", "b"}, []string{"a", "b"}) {
		t.Fatal("equal slices reported unequal")
	}
	if stringSlicesEqual([]string{"a"}, []string{"a", "b"}) {
		t.Fatal("different slices reported equal")
	}
	if got := stringIndex([]string{"x", "y"}, "y"); got != 1 {
		t.Fatalf("stringIndex y = %d; want 1", got)
	}
	if got := stringIndex([]string{"x"}, "z"); got != -1 {
		t.Fatalf("stringIndex z = %d; want -1", got)
	}
}
