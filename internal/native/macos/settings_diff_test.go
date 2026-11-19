//go:build darwin

package macos

import "testing"

func TestSettingsActionsSameShape(t *testing.T) {
	a := []settingsAction{{key: "copy", title: "Copy", enabled: true}, {key: "delete", title: "Delete"}}
	sameShapeDifferentState := []settingsAction{{key: "copy", title: "复制", enabled: false}, {key: "delete", title: "删除", enabled: true}}
	reordered := []settingsAction{{key: "delete", title: "Delete"}, {key: "copy", title: "Copy"}}

	if !settingsActionsSameShape(a, sameShapeDifferentState) {
		t.Fatal("expected same shape for identical key order")
	}
	if settingsActionsSameShape(a, reordered) {
		t.Fatal("expected reordered rows to differ in shape")
	}
	if settingsActionsSameShape(a, a[:1]) {
		t.Fatal("expected shorter list to differ in shape")
	}
}

func TestSettingsActionChangedRows(t *testing.T) {
	current := []settingsAction{{key: "copy", title: "Copy", enabled: true}, {key: "delete", title: "Delete"}}
	next := []settingsAction{{key: "copy", title: "Copy", enabled: false}, {key: "delete", title: "Delete"}}

	rows := settingsActionChangedRows(current, next)
	if len(rows) != 1 || rows[0] != 0 {
		t.Fatalf("changed rows = %v; want [0]", rows)
	}

	if got := settingsActionChangedRows(current, current); len(got) != 0 {
		t.Fatalf("unchanged rows = %v; want none", got)
	}
}

func TestSettingsAppsDiff(t *testing.T) {
	base := []SettingsApp{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}}
	toggled := []SettingsApp{{ID: "a", Name: "A", Excluded: true}, {ID: "b", Name: "B"}}
	added := append(append([]SettingsApp(nil), base...), SettingsApp{ID: "c", Name: "C"})

	if settingsAppsEqual(base, toggled) {
		t.Fatal("excluded change must not compare equal")
	}
	if !settingsAppsSameIdentity(base, toggled) {
		t.Fatal("excluded change should keep the same identity")
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
