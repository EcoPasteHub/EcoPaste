//go:build darwin

package macos

import "testing"

func TestHistoryRowTextUsesNoteUntilHover(t *testing.T) {
	row := HistoryRow{Title: "original", Detail: "Safari · 刚刚", Note: "note"}
	display := PanelDisplay{ShowOriginal: true}

	title, _, detail := historyRowText(row, display, false)
	if title != "备注：note" || detail != "Safari · 刚刚 · 备注" {
		t.Errorf("default title = %q, detail = %q", title, detail)
	}
	title, _, detail = historyRowText(row, display, true)
	if title != "original" || detail != "Safari · 刚刚 · 备注" {
		t.Errorf("hovered title = %q, detail = %q", title, detail)
	}
}

func TestHistoryRowTextKeepsNoteInlineWhenHoverDisabled(t *testing.T) {
	row := HistoryRow{Title: "original", Detail: "Safari · 刚刚", Note: "note"}
	title, _, detail := historyRowText(row, PanelDisplay{}, false)
	if title != "original" || detail != "Safari · 刚刚 · note" {
		t.Errorf("title = %q, detail = %q", title, detail)
	}
}
