//go:build darwin

package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
)

const (
	settingsDisplayPreviewTop     = 250.0
	settingsDisplayPreviewPadding = 4.0
)

func (v *SettingsView) applyDisplayPreview(values SettingsValues) {
	if v.displayPreviewBox == 0 {
		return
	}
	display := PanelDisplay{
		TextMaxLines:   int(values.TextMaxLines),
		ImageMaxHeight: int(values.ImageMaxHeight),
		FileMaxCount:   int(values.FileMaxCount),
		ShowOriginal:   true,
	}
	textHeight := historyTextRowHeight(display.TextMaxLines)
	imageHeight := historyImageRowHeight(display)
	rowHeight := max(textHeight, imageHeight)
	boxWidth := settingsContentWidth - 2*settingsRowInset
	boxHeight := rowHeight + 2*settingsDisplayPreviewPadding
	setFrame(v.displayPreviewBox, settingsContentX+settingsRowInset, v.displayPreviewTop, boxWidth, boxHeight)
	if v.displayPreviewCard != 0 {
		setFrame(v.displayPreviewCard, settingsContentX, v.displayPreviewTop-settingsRowHeight-8,
			settingsContentWidth, settingsRowHeight+boxHeight+16)
	}
	setFrame(v.displayPreviewDivider, boxWidth/2, settingsDisplayPreviewPadding, 1, rowHeight)

	if len(v.displayPreviewRows) < 2 {
		return
	}
	columnWidth := (boxWidth - 3*settingsDisplayPreviewPadding) / 2
	textRow := HistoryRow{
		Kind:   "text",
		Title:  i18n.T(i18n.SettingsDisplayPreviewText),
		Detail: i18n.T(i18n.SettingsDisplayPreviewTextDetail),
	}
	imageRow := HistoryRow{
		Kind:   "image",
		Title:  i18n.T(i18n.SettingsDisplayPreviewImage),
		Detail: i18n.T(i18n.SettingsDisplayPreviewImageDetail),
	}
	fillHistoryRow(v.displayPreviewRows[0], textRow, display, false, false)
	fillHistoryRow(v.displayPreviewRows[1], imageRow, display, false, false)
	setFrame(v.displayPreviewRows[0], settingsDisplayPreviewPadding,
		settingsDisplayPreviewPadding+(rowHeight-textHeight)/2, columnWidth, textHeight)
	setFrame(v.displayPreviewRows[1], 2*settingsDisplayPreviewPadding+columnWidth,
		settingsDisplayPreviewPadding+(rowHeight-imageHeight)/2, columnWidth, imageHeight)
}
