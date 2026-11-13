package ui

import (
	"testing"
	"time"
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
	if row.Title != "hello" || row.Detail != "Safari · 2 分钟前 · greeting · 已固定 · 收藏" {
		t.Errorf("row = %+v", row)
	}
	if row.SourceIconPath != "/tmp/safari.png" {
		t.Errorf("source icon = %q", row.SourceIconPath)
	}
	if row.Note != "greeting" {
		t.Errorf("note = %q", row.Note)
	}
}

func TestFileTitleLimitsNames(t *testing.T) {
	got := FileTitle([]string{"/tmp/a.txt", "/tmp/b.txt", "/tmp/c.txt"}, 2)
	if got != "a.txt、b.txt 等 3 个" {
		t.Errorf("title = %q", got)
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
