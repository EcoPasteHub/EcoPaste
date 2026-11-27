//go:build darwin

package macos

import "testing"

func TestSettingsTabsHaveMetadata(t *testing.T) {
	want := []string{"general", "appearance", "clipboard", "interaction", "data", "system", "about"}
	if len(settingsTabs) != len(want) {
		t.Fatalf("settings tabs = %d; want %d", len(settingsTabs), len(want))
	}
	for _, name := range want {
		tab, ok := settingsTabs[name]
		if !ok {
			t.Fatalf("settings tab %q is missing", name)
		}
		if tab == "" {
			t.Fatalf("tab %q has no title", name)
		}
	}
}

func TestSettingsPagesMatchNavigation(t *testing.T) {
	want := []string{"general", "appearance", "clipboard", "interaction", "data", "system", "about"}
	onMain(func() {
		view := newSettings(settingsEmitterFunc(func(string, any) {}))
		defer view.Close()
		for _, name := range want {
			if view.tabItems[name] == 0 {
				t.Errorf("settings tab %q has no page", name)
			}
		}
		if len(view.tabItems) != len(want) {
			t.Errorf("settings pages = %d; want %d", len(view.tabItems), len(want))
		}
		if view.pageHeights["interaction"] <= view.pageHeights["appearance"] {
			t.Errorf("measured page heights do not reflect content: interaction=%v appearance=%v",
				view.pageHeights["interaction"], view.pageHeights["appearance"])
		}
	})
}

func TestEveryScalarSettingBelongsToOneCategory(t *testing.T) {
	want := map[string]string{
		"general.autoStart":           "general",
		"general.trayIcon":            "general",
		"shortcuts.openClipboard":     "general",
		"shortcuts.openPreference":    "general",
		"appearance.theme":            "appearance",
		"appearance.language":         "appearance",
		"display.textMaxLines":        "appearance",
		"display.imageMaxHeight":      "appearance",
		"display.fileMaxCount":        "appearance",
		"capture.text":                "clipboard",
		"capture.html":                "clipboard",
		"capture.rtf":                 "clipboard",
		"capture.image":               "clipboard",
		"capture.files":               "clipboard",
		"capture.maxTextMb":           "clipboard",
		"capture.maxImageMb":          "clipboard",
		"sort":                        "data",
		"history.retention":           "data",
		"history.maxCount":            "data",
		"sensitive.collectSecrets":    "data",
		"sensitive.redactSecrets":     "data",
		"content.deleteNormal":        "data",
		"content.deleteFavorite":      "data",
		"content.deletePinned":        "data",
		"copyThenHideWindow":          "interaction",
		"feedback.copySound":          "interaction",
		"window.position":             "interaction",
		"window.scrollToTopOnOpen":    "interaction",
		"window.selectRangeOnOpen":    "interaction",
		"window.selectCategoryOnOpen": "interaction",
		"window.selectGroupOnOpen":    "interaction",
		"search.clearOnHide":          "interaction",
		"content.autoPaste":           "interaction",
		"content.autoFavorite":        "interaction",
		"content.showOriginalPreview": "interaction",
		"copyPlain":                   "interaction",
		"preview.spaceEnabled":        "interaction",
		"logging.level":               "system",
	}

	onMain(func() {
		view := newSettings(settingsEmitterFunc(func(string, any) {}))
		defer view.Close()
		seen := make(map[string]string, len(want))
		mark := func(path, tab string) {
			t.Helper()
			if expected, ok := want[path]; !ok {
				t.Errorf("unexpected setting %q in category %q", path, tab)
			} else if expected != tab {
				t.Errorf("setting %q belongs to %q, got %q", path, expected, tab)
			}
			if previous, ok := seen[path]; ok {
				t.Errorf("setting %q appears in both %q and %q", path, previous, tab)
			}
			seen[path] = tab
		}
		for _, field := range view.fields {
			mark(field.path, field.tab)
		}
		for _, input := range view.numberInputs {
			mark(input.path, input.tab)
		}
		for _, recorder := range view.shortcutRecorders {
			mark(recorder.path, recorder.tab)
		}
		if view.captureTable == 0 || view.appsHost == 0 {
			t.Error("clipboard category is missing capture order or source-app controls")
		}
		if view.dataDir == 0 || view.change == 0 || view.restore == 0 {
			t.Error("data category is missing storage controls")
		}
		if len(seen) != len(want) {
			for path := range want {
				if _, ok := seen[path]; !ok {
					t.Errorf("setting %q is not assigned to a category", path)
				}
			}
		}
	})
}
