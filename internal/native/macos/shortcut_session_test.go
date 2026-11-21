//go:build darwin

package macos

import "testing"

func TestFormatShortcutGlyphs(t *testing.T) {
	cases := map[string]string{
		"Command+C":           "⌘ C",
		"Control+Alt+Shift+K": "⌃⌥⇧ K",
		"Alt+ArrowUp":         "⌥ ↑",
		"F5":                  "F5",
		"":                    "",
	}
	for value, want := range cases {
		if got := formatShortcutGlyphs(value); got != want {
			t.Errorf("formatShortcutGlyphs(%q) = %q, want %q", value, got, want)
		}
	}
}

func TestShortcutRecordingShowsModifiersBeforeKey(t *testing.T) {
	recording := shortcutRecording{}
	recording.updateModifiers([]string{"command"})
	if got := recording.display(); got != "⌘" {
		t.Fatalf("modifier display = %q", got)
	}
	recording.press("C")
	if got := recording.display(); got != "⌘ C" {
		t.Fatalf("combo display = %q", got)
	}
	if value := recording.complete(); value != "Command+C" {
		t.Fatalf("complete = %q", value)
	}
}

func TestShortcutRecordingAcceptsKeyBeforeModifier(t *testing.T) {
	recording := shortcutRecording{}
	recording.press("C")
	if got := recording.display(); got != "C" {
		t.Fatalf("key display = %q", got)
	}
	if value := recording.complete(); value != "" {
		t.Fatalf("single key should not complete: %q", value)
	}
	recording.updateModifiers([]string{"alt"})
	if got := recording.display(); got != "⌥ C" {
		t.Fatalf("key-first combo display = %q", got)
	}
	if value := recording.complete(); value != "Alt+C" {
		t.Fatalf("complete = %q", value)
	}
}

func TestShortcutRecordingReleaseClearsKeyDisplay(t *testing.T) {
	recording := shortcutRecording{}
	recording.press("C")
	recording.release("C")
	if got := recording.display(); got != "" {
		t.Fatalf("display after release = %q", got)
	}
}

func TestShortcutRecordingKeepsKeyWhileAddingModifiers(t *testing.T) {
	recording := shortcutRecording{}
	recording.updateModifiers([]string{"shift"})
	recording.press("C")
	if value := recording.complete(); value != "" {
		t.Fatalf("shift+C should be invalid: %q", value)
	}
	recording.updateModifiers([]string{"command", "shift"})
	if got := recording.display(); got != "⌘⇧ C" {
		t.Fatalf("display after adding command = %q", got)
	}
	if value := recording.complete(); value != "Command+Shift+C" {
		t.Fatalf("complete = %q", value)
	}
}

func TestShortcutRecordingFunctionKeyAlone(t *testing.T) {
	recording := shortcutRecording{}
	recording.press("F5")
	if value := recording.complete(); value != "F5" {
		t.Fatalf("complete = %q", value)
	}

	recording = shortcutRecording{}
	recording.press("C")
	if value := recording.complete(); value != "" {
		t.Fatalf("single letter should be rejected: %q", value)
	}
}

func TestShortcutRecordingStorageNames(t *testing.T) {
	recording := shortcutRecording{}
	recording.updateModifiers([]string{"alt"})
	recording.press("ArrowUp")
	if value := recording.complete(); value != "Alt+ArrowUp" {
		t.Fatalf("complete = %q", value)
	}
}

func TestShortcutRecordingRepeatDoesNotToggleKey(t *testing.T) {
	recording := shortcutRecording{}
	recording.press("C")
	recording.press("C")
	if got := recording.display(); got != "C" {
		t.Fatalf("display after key repeat = %q", got)
	}
}

func TestShortcutModifiersReduced(t *testing.T) {
	if !shortcutModifiersReduced([]string{"alt", "shift"}, []string{"alt"}) {
		t.Fatal("removing shift should count as reduced")
	}
	if shortcutModifiersReduced([]string{"alt"}, []string{"alt", "shift"}) {
		t.Fatal("adding shift should not count as reduced")
	}
	if shortcutModifiersReduced([]string{"alt"}, []string{"alt"}) {
		t.Fatal("unchanged modifiers should not count as reduced")
	}
}
