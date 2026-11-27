//go:build darwin

package macos

import (
	"strings"
)

// shortcutModifierGlyph is the menu-style symbol shown for one modifier.
func shortcutModifierGlyph(name string) string {
	switch name {
	case "command":
		return "⌘"
	case "control":
		return "⌃"
	case "alt":
		return "⌥"
	case "shift":
		return "⇧"
	}
	return ""
}

// shortcutKeyGlyph returns the symbol shown for one recorded key. Keys
// without a single-character symbol render their stored name.
func shortcutKeyGlyph(name string) string {
	switch strings.ToLower(name) {
	case "enter", "return":
		return "↩"
	case "tab":
		return "⇥"
	case "space":
		return "Space"
	case "backspace":
		return "⌫"
	case "delete":
		return "⌦"
	case "esc", "escape":
		return "⎋"
	case "arrowup", "up":
		return "↑"
	case "arrowdown", "down":
		return "↓"
	case "arrowleft", "left":
		return "←"
	case "arrowright", "right":
		return "→"
	case "home":
		return "↖"
	case "end":
		return "↘"
	case "pageup":
		return "⇞"
	case "pagedown":
		return "⇟"
	}
	if name == "" {
		return ""
	}
	return strings.ToUpper(name)
}

// formatShortcutGlyphs renders a stored shortcut like "Alt+Shift+C" as
// "⌥ ⇧ C". Unknown fragments are shown verbatim so a hand-edited value
// never hides what is stored.
func formatShortcutGlyphs(value string) string {
	parts := strings.Split(strings.TrimSpace(value), "+")
	modifiers := make([]string, 0, len(parts))
	key := ""
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if modifier := shortcutModifierGlyph(strings.ToLower(part)); modifier != "" {
			modifiers = append(modifiers, modifier)
			continue
		}
		key = shortcutKeyGlyph(part)
	}
	text := strings.Join(modifiers, "")
	if key == "" {
		return text
	}
	if text == "" {
		return key
	}
	return text + " " + key
}

// shortcutRecording is one capture in progress. It is independent of
// AppKit so the key sequence and its display can be tested with events
// replayed as plain values.
type shortcutRecording struct {
	modifiers []string
	key       string
}

var shortcutCaptureModifiers = []string{"command", "control", "alt", "shift"}

// modifierGlyph builds the display text for the held modifiers.
func (r *shortcutRecording) modifierGlyph() string {
	glyphs := make([]string, 0, len(r.modifiers))
	for _, name := range r.modifiers {
		glyphs = append(glyphs, shortcutModifierGlyph(name))
	}
	return strings.Join(glyphs, "")
}

// display returns what the recorder shows while capturing. Modifiers are
// displayed as soon as they go down; a key appends its symbol.
func (r *shortcutRecording) display() string {
	text := r.modifierGlyph()
	if r.key == "" {
		return text
	}
	key := shortcutKeyGlyph(r.key)
	if text == "" {
		return key
	}
	return text + " " + key
}

// updateModifiers replaces the held modifier set. The non-modifier key is
// kept, so modifiers can be pressed after the key and still form the same
// combination.
func (r *shortcutRecording) updateModifiers(held []string) {
	next := make([]string, 0, len(held))
	set := map[string]bool{}
	for _, name := range held {
		set[strings.ToLower(name)] = true
	}
	for _, name := range shortcutCaptureModifiers {
		if set[name] {
			next = append(next, name)
		}
	}
	r.modifiers = next
}

// press records a key that was pressed down. Key repeat does not toggle the
// key off; release removes it.
func (r *shortcutRecording) press(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	if strings.EqualFold(r.key, name) {
		return
	}
	r.key = name
}

// holdsKey reports whether name is the non-modifier key currently down.
func (r *shortcutRecording) holdsKey(name string) bool {
	return name != "" && strings.EqualFold(r.key, strings.TrimSpace(name))
}

// release removes the non-modifier key when it is the one being held.
func (r *shortcutRecording) release(name string) {
	if r.holdsKey(name) {
		r.key = ""
	}
}

// complete turns the held keys into a stored shortcut. Incomplete or
// disallowed combinations return an empty value and leave the capture
// running.
func (r *shortcutRecording) complete() string {
	if r.key == "" {
		return ""
	}
	hasPrimary := false
	for _, name := range r.modifiers {
		if name != "shift" {
			hasPrimary = true
			break
		}
	}
	if !hasPrimary && !standaloneShortcutKey(r.key) {
		return ""
	}
	modifiers := make([]string, 0, len(r.modifiers))
	for _, name := range r.modifiers {
		modifiers = append(modifiers, strings.ToUpper(name[:1])+name[1:])
	}
	value := strings.Join(append(modifiers, shortcutStorageKey(r.key)), "+")
	return value
}

// shortcutStorageKey maps recorder names to the key names used by stored
// shortcuts and Carbon registration.
func shortcutStorageKey(name string) string {
	switch strings.ToLower(name) {
	case "enter":
		return "Enter"
	case "tab":
		return "Tab"
	case "space":
		return "Space"
	case "backspace":
		return "Backspace"
	case "delete":
		return "Delete"
	case "esc":
		return "Esc"
	case "arrowup":
		return "ArrowUp"
	case "arrowdown":
		return "ArrowDown"
	case "arrowleft":
		return "ArrowLeft"
	case "arrowright":
		return "ArrowRight"
	case "home":
		return "Home"
	case "end":
		return "End"
	case "pageup":
		return "PageUp"
	case "pagedown":
		return "PageDown"
	}
	return strings.ToUpper(name)
}

func standaloneShortcutKey(name string) bool {
	key := strings.ToLower(name)
	if len(key) < 2 || key[0] != 'f' {
		return false
	}
	switch key {
	case "f1", "f2", "f3", "f4", "f5", "f6", "f7", "f8", "f9", "f10", "f11", "f12":
		return true
	}
	return false
}
