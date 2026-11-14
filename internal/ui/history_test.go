package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
)

func TestRelativeTimeBuckets(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.Local)
	cases := []struct {
		then time.Time
		want string
	}{
		{now.Add(-20 * time.Second), "刚刚"},
		{now.Add(-8 * time.Minute), "8 分钟前"},
		{now.Add(-3 * time.Hour), "3 小时前"},
		{now.Add(-30 * time.Hour), "昨天"},
		{now.Add(-4 * 24 * time.Hour), "4 天前"},
		{time.Date(2026, 10, 1, 9, 0, 0, 0, time.Local), "10-01"},
	}
	for _, tc := range cases {
		if got := relativeTime(now, tc.then); got != tc.want {
			t.Errorf("relativeTime(%v) = %q, want %q", tc.then, got, tc.want)
		}
	}
}

func TestHistoryRowIncludesMetadata(t *testing.T) {
	row := historyRow(HistoryItem{
		ID:            "1",
		Kind:          "text",
		Title:         "hello",
		SourceApp:     "Safari",
		SourceAppIcon: "/tmp/safari.png",
		CreatedAt:     time.Now().Add(-2 * time.Minute),
		Note:          "greeting",
		Favorite:      true,
		Pinned:        true,
	})
	if row.Title != "hello" || row.Detail != "文本 · Safari · 2 分钟前 · 已固定 · 收藏" {
		t.Errorf("row = %+v", row)
	}
	if row.SourceIconPath != "/tmp/safari.png" {
		t.Errorf("source icon = %q", row.SourceIconPath)
	}
	if row.Note != "greeting" {
		t.Errorf("note = %q", row.Note)
	}
}

func TestHistoryRowSplitsTextPreview(t *testing.T) {
	row := historyRow(HistoryItem{
		ID:    "1",
		Kind:  "text",
		Title: "first line\nsecond line\nthird line",
	})
	if row.Title != "first line" || row.Preview != "second line third line" {
		t.Errorf("title = %q, preview = %q", row.Title, row.Preview)
	}
	if row.Detail != "文本" {
		t.Errorf("detail = %q", row.Detail)
	}

	single := historyRow(HistoryItem{ID: "2", Kind: "text", Title: "single line"})
	if single.Title != "single line" || single.Preview != "" {
		t.Errorf("single title = %q, preview = %q", single.Title, single.Preview)
	}
}

func TestFileTitleLimitsNames(t *testing.T) {
	got := FileTitle([]string{"/tmp/a.txt", "/tmp/b.txt", "/tmp/c.txt"}, 2)
	if got != "a.txt、b.txt 等 3 个" {
		t.Errorf("title = %q", got)
	}
}

func TestHistoryRowIncludesFileSummary(t *testing.T) {
	row := historyRow(HistoryItem{
		ID:          "1",
		Kind:        "files",
		Title:       "report.pdf",
		FileSummary: "PDF · 2.0 KB",
		CreatedAt:   time.Now(),
	})
	if !strings.HasPrefix(row.Detail, "文件 · PDF · 2.0 KB · 刚刚") {
		t.Errorf("detail = %q", row.Detail)
	}
}

func TestHistoryRowTypeLabels(t *testing.T) {
	cases := []struct {
		kind, subKind, want string
	}{
		{"text", "", "文本"},
		{"text", "html", "HTML"},
		{"text", "rtf", "RTF"},
		{"text", "url", "链接"},
		{"text", "email", "邮件"},
		{"text", "color", "颜色"},
		{"text", "path", "路径"},
		{"image", "", "图片"},
		{"files", "", "文件"},
	}
	for _, tc := range cases {
		row := historyRow(HistoryItem{Kind: tc.kind, SubKind: tc.subKind, Title: "content"})
		if !strings.HasPrefix(row.Detail, tc.want) {
			t.Errorf("type label for %s/%s = %q, want prefix %q",
				tc.kind, tc.subKind, row.Detail, tc.want)
		}
	}
}

func TestHistoryPresentationFollowsLanguage(t *testing.T) {
	i18n.SetLanguage("en-US")
	t.Cleanup(func() { i18n.SetLanguage("zh-CN") })

	row := historyRow(HistoryItem{
		ID:        "1",
		Kind:      "text",
		Title:     "hello",
		CreatedAt: time.Now(),
		Pinned:    true,
	})
	if row.Detail != "Text · Just now · Pinned" {
		t.Errorf("detail = %q", row.Detail)
	}
	if got := FileTitle([]string{"/tmp/a.txt", "/tmp/b.txt", "/tmp/c.txt"}, 2); got != "a.txt, b.txt and 3 more" {
		t.Errorf("file title = %q", got)
	}
}

func TestHideAfterActionRespectsPinnedPanel(t *testing.T) {
	if hideAfterAction(false, true) != true {
		t.Error("unpinned panel ignored hide request")
	}
	if hideAfterAction(true, true) {
		t.Error("pinned panel hid after action")
	}
	if hideAfterAction(false, false) {
		t.Error("hide request was invented")
	}
}

func TestOpenSelectionAppliesCategoryAndGroup(t *testing.T) {
	cases := []struct {
		name    string
		opening HistoryOpenBehavior
		filter  string
		group   string
		want    [2]string
	}{
		{"category only", HistoryOpenBehavior{Filter: "image", GroupID: "preserve"}, "", "", [2]string{"image", ""}},
		{"all groups keeps category", HistoryOpenBehavior{Filter: "image", GroupID: ""}, "", "work", [2]string{"image", ""}},
		{"group wins over category", HistoryOpenBehavior{Filter: "image", GroupID: "work"}, "", "", [2]string{"", "work"}},
		{"all groups clears group", HistoryOpenBehavior{Filter: "preserve", GroupID: ""}, "text", "work", [2]string{"text", ""}},
		{"preserve keeps both", HistoryOpenBehavior{Filter: "preserve", GroupID: "preserve"}, "files", "work", [2]string{"files", "work"}},
		{"all category clears filter", HistoryOpenBehavior{Filter: "", GroupID: "preserve"}, "image", "", [2]string{"", ""}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			filter, group := openSelection(tc.opening, tc.filter, tc.group)
			if [2]string{filter, group} != tc.want {
				t.Errorf("open selection = %q, %q; want %q, %q", filter, group, tc.want[0], tc.want[1])
			}
		})
	}
}
