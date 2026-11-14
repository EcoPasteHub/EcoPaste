//go:build darwin

package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

const (
	settingsDisplayPreviewTop     = 250.0
	settingsDisplayPreviewPadding = 4.0
)

func (v *SettingsView) buildDisplay() {
	content := v.newTab("display", settingsTabTitle("display"))
	v.addSectionLabel(content, 438, i18n.T(i18n.SettingsDisplayDensity))

	y := 408.0
	v.addStepperRange(content, y, i18n.T(i18n.SettingsTextMaxLines), "display.textMaxLines",
		1, 5, 1, i18n.T(i18n.SettingsUnitLines),
		func(s SettingsValues) uint32 { return s.TextMaxLines })
	v.addStepperRange(content, y-42, i18n.T(i18n.SettingsImageMaxHeight), "display.imageMaxHeight", 20, 100, 4, " px",
		func(s SettingsValues) uint32 { return s.ImageMaxHeight })
	v.addStepperRange(content, y-84, i18n.T(i18n.SettingsFileMaxCount), "display.fileMaxCount",
		1, 5, 1, i18n.T(i18n.SettingsUnitItems),
		func(s SettingsValues) uint32 { return s.FileMaxCount })

	v.addSectionLabel(content, 286, i18n.T(i18n.SettingsDisplayPreview))
	hint := objc.ID(classNSTextField).Send(sel("labelWithString:"),
		nsString(i18n.T(i18n.SettingsDisplayPreviewHint)))
	hint.Send(sel("setFont:"), systemFont(11))
	hint.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("tertiaryLabelColor")))
	hint.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: 262}, Size: CGSize{Width: tabContentW - 48, Height: 16}})
	content.Send(sel("addSubview:"), hint)

	box := objc.ID(class("NSVisualEffectView")).Send(sel("new"))
	box.Send(sel("setMaterial:"), int64(14))
	box.Send(sel("setBlendingMode:"), int64(1))
	box.Send(sel("setState:"), int64(1))
	box.Send(sel("setWantsLayer:"), true)
	if layer := box.Send(sel("layer")); layer != 0 {
		layer.Send(sel("setCornerRadius:"), 8.0)
		layer.Send(sel("setMasksToBounds:"), true)
	}
	content.Send(sel("addSubview:"), box)
	v.displayPreviewBox = box

	divider := objc.ID(classNSBox).Send(sel("new"))
	divider.Send(sel("setBoxType:"), 2)
	divider.Send(sel("setTitlePosition:"), 0)
	box.Send(sel("addSubview:"), divider)
	v.displayPreviewDivider = divider

	if err := registerHistoryRowViews(); err != nil {
		return
	}
	for range 2 {
		row := newHistoryRow()
		box.Send(sel("addSubview:"), row)
		v.displayPreviewRows = append(v.displayPreviewRows, row)
	}
}

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
	boxWidth := float64(tabContentW - 48)
	boxHeight := rowHeight + 2*settingsDisplayPreviewPadding
	setFrame(v.displayPreviewBox, 24, settingsDisplayPreviewTop-boxHeight, boxWidth, boxHeight)
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
