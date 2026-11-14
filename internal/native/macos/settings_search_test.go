//go:build darwin

package macos

import (
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
)

func TestMatchSettingsSearchUsesTitlePathAndKeywords(t *testing.T) {
	entries := []settingsSearchEntry{
		{
			title: "显示 Dock 图标", path: "general.dockIcon",
			tab: "general", tabTitle: "通用",
		},
		{
			title: "文本大小上限", path: "capture.maxTextMb",
			tab: "capture", tabTitle: "采集",
		},
		{
			title: "排序方式", path: "sort", keywords: "最近使用 创建时间 使用次数",
			tab: "general", tabTitle: "通用",
		},
	}

	cases := map[string]string{
		"dock":     "general.dockIcon",
		"max text": "capture.maxTextMb",
		"创建时间":     "sort",
	}
	for query, want := range cases {
		got := matchSettingsSearch(entries, query)
		if len(got) != 1 || got[0].path != want {
			t.Errorf("matchSettingsSearch(%q) = %v, want %q", query, got, want)
		}
	}
}

func TestMatchSettingsSearchRequiresEveryTermAndDeduplicates(t *testing.T) {
	entries := []settingsSearchEntry{
		{
			title: "打开时选中分类", path: "window.selectCategoryOnOpen",
			tab: "window", tabTitle: "窗口",
		},
		{
			title: "面板打开位置", path: "window.position",
			tab: "general", tabTitle: "通用",
		},
		{
			title: "面板打开位置", path: "window.position",
			tab: "window", tabTitle: "窗口",
		},
	}
	got := matchSettingsSearch(entries, "window position")
	if len(got) != 1 || got[0].tab != "general" {
		t.Fatalf("deduplicated matches = %v", got)
	}
	if got := matchSettingsSearch(entries, "window missing"); len(got) != 0 {
		t.Errorf("partial term match = %v", got)
	}
}

func TestMatchSettingsSearchLimitsResults(t *testing.T) {
	entries := make([]settingsSearchEntry, 0, 12)
	for index := 0; index < 12; index++ {
		entries = append(entries, settingsSearchEntry{
			title: "粘贴设置", path: "settings." + string(rune('a'+index)),
			tab: "general", tabTitle: "通用",
		})
	}
	if got := matchSettingsSearch(entries, "粘贴"); len(got) != settingsSearchLimit {
		t.Errorf("result count = %d, want %d", len(got), settingsSearchLimit)
	}
	if got := matchSettingsSearch(entries, "  "); len(got) != 0 {
		t.Errorf("empty query returned %v", got)
	}
}

func TestSettingsPathWords(t *testing.T) {
	got := settingsPathWords("clipboard.maxTextMB")
	if got != "clipboard max text m b" {
		t.Errorf("settingsPathWords = %q", got)
	}
}

func TestSettingsSearchMatchesEnglishLabels(t *testing.T) {
	i18n.SetLanguage("en-US")
	t.Cleanup(func() { i18n.SetLanguage("zh-CN") })

	entries := []settingsSearchEntry{
		{
			title: i18n.T(i18n.SettingsDockIcon), path: "general.dockIcon",
			tab: "general", tabTitle: settingsTabTitle("general"),
		},
		{
			title: i18n.T(i18n.SettingsMaxText), path: "capture.maxTextMb",
			tab: "capture", tabTitle: settingsTabTitle("capture"),
		},
	}
	if got := matchSettingsSearch(entries, "dock icon"); len(got) != 1 || got[0].path != "general.dockIcon" {
		t.Errorf("dock icon matches = %+v", got)
	}
	if got := matchSettingsSearch(entries, "capture maximum text"); len(got) != 1 || got[0].path != "capture.maxTextMb" {
		t.Errorf("capture maximum text matches = %+v", got)
	}
}
