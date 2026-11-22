//go:build darwin

package macos

import (
	"testing"

	"github.com/ebitengine/purego/objc"
)

func newTestSettingsView(t *testing.T) (*SettingsView, *[]SettingsChange) {
	t.Helper()
	var changes []SettingsChange
	view := newSettings(settingsEmitterFunc(func(_ string, data any) {
		if change, ok := data.(SettingsChange); ok {
			changes = append(changes, change)
		}
	}))
	t.Cleanup(func() { onMain(view.Close) })
	return view, &changes
}

type settingsEmitterFunc func(name string, data any)

func (f settingsEmitterFunc) Emit(name string, data any) { f(name, data) }

func settingsControlPaths() map[string]bool {
	paths := []string{
		"logging.level",
		"general.autoStart", "general.trayIcon",
		"shortcuts.openClipboard", "shortcuts.openPreference",
		"appearance.theme", "appearance.language",
		"copyThenHideWindow", "feedback.copySound", "sort", "window.position",
		"display.textMaxLines", "display.imageMaxHeight", "display.fileMaxCount",
		"window.scrollToTopOnOpen", "window.selectRangeOnOpen",
		"window.selectCategoryOnOpen", "window.selectGroupOnOpen",
		"search.clearOnHide",
		"content.autoPaste",
		"content.autoFavorite", "content.showOriginalPreview", "copyPlain",
		"preview.spaceEnabled",
		"capture.text", "capture.html", "capture.rtf", "capture.image", "capture.files",
		"capture.maxTextMb", "capture.maxImageMb",
		"history.retention", "history.maxCount",
		"sensitive.collectSecrets", "sensitive.redactSecrets",
		"content.deleteConfirm", "content.deleteFavoriteConfirm", "content.deletePinnedConfirm",
		"content.deleteFavoriteItems", "content.deletePinnedItems", "content.deleteFavoriteOnlyFav",
	}
	want := make(map[string]bool, len(paths))
	for _, path := range paths {
		want[path] = true
	}
	return want
}

func TestSettingsControlsEmitTheirOwnPath(t *testing.T) {
	onMain(func() { testSettingsControlsEmitTheirOwnPath(t) })
}

func testSettingsControlsEmitTheirOwnPath(t *testing.T) {
	view, changes := newTestSettingsView(t)
	if view == nil {
		return
	}
	want := settingsControlPaths()
	seen := map[string]bool{}
	record := func(path string) {
		t.Helper()
		if !want[path] {
			t.Errorf("unexpected control path %q", path)
		}
		if seen[path] {
			t.Errorf("duplicate control path %q", path)
		}
		seen[path] = true
	}

	for _, field := range view.fields {
		*changes = nil
		switch field.kind {
		case "bool":
			field.control.Send(sel("setState:"), int64(1))
			settingsToggleChanged(0, 0, field.control)
		case "popup":
			field.control.Send(sel("selectItemAtIndex:"), int64(len(field.options)-1))
			settingsPopupChanged(0, 0, field.control)
		}
		if len(*changes) != 1 {
			t.Errorf("%s emitted %d changes", field.path, len(*changes))
			continue
		}
		if change := (*changes)[0]; change.Path != field.path {
			t.Errorf("control for %q emitted %q", field.path, change.Path)
		}
		record(field.path)
	}
	for _, n := range view.numberInputs {
		*changes = nil
		n.step(n.stepValue)
		if len(*changes) != 1 || (*changes)[0].Path != n.path {
			t.Errorf("number input %q emitted %+v", n.path, *changes)
		}
		record(n.path)
	}
	for _, recorder := range view.shortcutRecorders {
		*changes = nil
		recorder.recording = true
		recorder.submitRecording("Alt+J")
		if len(*changes) != 1 || (*changes)[0].Path != recorder.path {
			t.Errorf("shortcut recorder %q emitted %+v", recorder.path, *changes)
		}
		recorder.recording = false
		record(recorder.path)
	}
	for path := range want {
		if !seen[path] {
			t.Errorf("no control for %q", path)
		}
	}
}

func TestSettingsControlsShowLoadedValues(t *testing.T) {
	onMain(func() { testSettingsControlsShowLoadedValues(t) })
}

func testSettingsControlsShowLoadedValues(t *testing.T) {
	view, _ := newTestSettingsView(t)
	if view == nil {
		return
	}
	view.SetProp("values", SettingsValues{
		LogLevel:              "all",
		SearchClear:           false,
		Sort:                  "useCountDesc",
		TextMaxLines:          5,
		OpenClipboardShortcut: "Alt+J",
		CopyThenHideWindow:    true,
		DeleteFavoriteItems:   true,
		ShowOriginalPreview:   true,
		SpacePreview:          true,
		CaptureFiles:          true,
		CollectSecrets:        true,
		GroupOnOpen:           "group:work",
		Groups:                []SettingsGroup{{ID: "work", Name: "Work"}},
	})
	for _, field := range view.fields {
		switch field.path {
		case "logging.level":
			if index := int(objc.Send[int64](field.control, sel("indexOfSelectedItem"))); index < 0 ||
				index >= len(field.options) || field.options[index] != "all" {
				t.Errorf("log level selection = %d of %v", index, field.options)
			}
		case "search.clearOnHide", "content.deleteConfirm":
			if got := objcState(field.control); got != 0 {
				t.Errorf("%s = %d", field.path, got)
			}
		case "sort":
			if got := field.options[int(field.control.Send(sel("indexOfSelectedItem")))]; got != "useCountDesc" {
				t.Errorf("sort = %q", got)
			}
		case "content.showOriginalPreview", "preview.spaceEnabled",
			"content.deleteFavoriteItems", "capture.files", "sensitive.collectSecrets":
			if got := objcState(field.control); got != 1 {
				t.Errorf("%s = %d", field.path, got)
			}
		}
	}
	if view.groupField == nil {
		t.Error("group popup missing")
	} else if index := int(objc.Send[int64](view.groupField.control, sel("indexOfSelectedItem"))); index < 0 ||
		index >= len(view.groupField.options) || view.groupField.options[index] != "group:work" {
		t.Errorf("group selection = %d of %v", index, view.groupField.options)
	}
	for _, n := range view.numberInputs {
		if n.path == "display.textMaxLines" && n.value != 5 {
			t.Errorf("text max lines = %d", n.value)
		}
	}
	for _, recorder := range view.shortcutRecorders {
		if recorder.path == "shortcuts.openClipboard" && recorder.value != "Alt+J" {
			t.Errorf("open clipboard shortcut = %q", recorder.value)
		}
	}
}

func objcState(control objc.ID) int64 {
	return objc.Send[int64](control, sel("state"))
}
