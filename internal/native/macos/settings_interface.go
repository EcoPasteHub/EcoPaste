//go:build darwin

package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

func (v *SettingsView) buildInterfacePage() {
	document := v.newPage("interface")
	top := 18.0

	_, cardTop := v.addSectionCard(document, top, i18n.SettingsDisplayDensity, 3)
	v.addStepperRange(document, settingsRowTop(cardTop, 0), i18n.SettingsTextMaxLines, "display.textMaxLines",
		1, 5, 1, i18n.SettingsUnitLines,
		func(s SettingsValues) uint32 { return s.TextMaxLines })
	v.addStepperRange(document, settingsRowTop(cardTop, 1), i18n.SettingsImageMaxHeight, "display.imageMaxHeight",
		20, 100, 4, " px",
		func(s SettingsValues) uint32 { return s.ImageMaxHeight })
	v.addStepperRange(document, settingsRowTop(cardTop, 2), i18n.SettingsFileMaxCount, "display.fileMaxCount",
		1, 5, 1, i18n.SettingsUnitItems,
		func(s SettingsValues) uint32 { return s.FileMaxCount })
	top = cardTop + 3*settingsRowHeight + settingsGroupGap

	_, cardTop = v.addSectionCard(document, top, i18n.SettingsWindowOpen, 4)
	v.addSwitch(document, settingsRowTop(cardTop, 0), i18n.SettingsScrollTop, "window.scrollToTopOnOpen",
		func(s SettingsValues) bool { return s.ScrollToTopOnOpen })
	v.addPopup(document, settingsRowTop(cardTop, 1), i18n.SettingsSelectRange, "window.selectRangeOnOpen",
		[]settingsOption{
			{Key: i18n.SettingsPreserve, Value: "preserve"},
			{Key: i18n.SettingsAll, Value: "all"},
			{Key: i18n.PanelFavorites, Value: "favorite"},
		},
		func(s SettingsValues) string { return s.RangeOnOpen })
	v.addPopup(document, settingsRowTop(cardTop, 2), i18n.SettingsSelectCategory, "window.selectCategoryOnOpen",
		[]settingsOption{
			{Key: i18n.SettingsPreserve, Value: "preserve"},
			{Key: i18n.SettingsAll, Value: "all"},
			{Key: i18n.PanelTextCategory, Value: "text"},
			{Key: i18n.PanelImageCategory, Value: "image"},
			{Key: i18n.PanelFilesCategory, Value: "files"},
		},
		func(s SettingsValues) string { return s.CategoryOnOpen })
	v.groupField = v.addPopup(document, settingsRowTop(cardTop, 3), i18n.SettingsSelectGroup, "window.selectGroupOnOpen",
		[]settingsOption{
			{Key: i18n.SettingsPreserve, Value: "preserve"},
			{Key: i18n.SettingsAllGroups, Value: "all"},
		},
		func(s SettingsValues) string { return s.GroupOnOpen })
	top = cardTop + 4*settingsRowHeight
	top = v.addPageHint(document, top+8, 30, i18n.SettingsWindowHint) + 18

	_, cardTop = v.addSectionCard(document, top, i18n.SettingsWindowSearch, 2)
	v.addSwitch(document, settingsRowTop(cardTop, 0), i18n.SettingsSearchFocus, "search.defaultFocus",
		func(s SettingsValues) bool { return s.SearchFocus })
	v.addSwitch(document, settingsRowTop(cardTop, 1), i18n.SettingsSearchClear, "search.clearOnHide",
		func(s SettingsValues) bool { return s.SearchClear })
	top = cardTop + 2*settingsRowHeight + settingsGroupGap

	_, cardTop = v.addSectionCard(document, top, i18n.SettingsInteractionMouse, 7)
	v.addPopup(document, settingsRowTop(cardTop, 0), i18n.SettingsMouseLeft, "content.autoPaste",
		[]settingsOption{
			{Key: i18n.SettingsDisabled, Value: "disabled"},
			{Key: i18n.SettingsSingleClickPaste, Value: "singleClickPaste"},
			{Key: i18n.SettingsDoubleClickPaste, Value: "doubleClickPaste"},
			{Key: i18n.SettingsSingleClickCopy, Value: "singleClickCopy"},
			{Key: i18n.SettingsDoubleClickCopy, Value: "doubleClickCopy"},
		},
		func(s SettingsValues) string { return s.AutoPaste })
	v.addPopup(document, settingsRowTop(cardTop, 1), i18n.SettingsMouseMiddle, "content.middleClick",
		[]settingsOption{
			{Key: i18n.SettingsDisabled, Value: "disabled"},
			{Key: i18n.MenuPaste, Value: "singleClickPaste"},
			{Key: i18n.SettingsSinglePastePlain, Value: "singleClickPastePlain"},
			{Key: i18n.MenuCopy, Value: "singleClickCopy"},
			{Key: i18n.SettingsSingleCopyPlain, Value: "singleClickCopyPlain"},
		},
		func(s SettingsValues) string { return s.MiddleClick })
	v.addSwitch(document, settingsRowTop(cardTop, 2), i18n.SettingsPasteFilesAsPath, "content.pasteFilesAsPath",
		func(s SettingsValues) bool { return s.PasteFilesAsPath })
	v.addSwitch(document, settingsRowTop(cardTop, 3), i18n.SettingsAutoFavorite, "content.autoFavorite",
		func(s SettingsValues) bool { return s.AutoFavorite })
	v.addSwitch(document, settingsRowTop(cardTop, 4), i18n.SettingsShowOriginal, "content.showOriginalPreview",
		func(s SettingsValues) bool { return s.ShowOriginalPreview })
	v.addSwitch(document, settingsRowTop(cardTop, 5), i18n.SettingsCopyTextPlain, "copyPlain",
		func(s SettingsValues) bool { return s.CopyPlain })
	v.addSwitch(document, settingsRowTop(cardTop, 6), i18n.SettingsPasteTextPlain, "pastePlain",
		func(s SettingsValues) bool { return s.PastePlain })
	top = cardTop + 7*settingsRowHeight + settingsGroupGap

	previewCard, cardTop := v.addSectionCard(document, top, i18n.SettingsDisplayPreview, 1)
	v.addSwitch(document, settingsRowTop(cardTop, 0), i18n.SettingsSpacePreview, "preview.spaceEnabled",
		func(s SettingsValues) bool { return s.SpacePreview })
	v.displayPreviewTop = cardTop + settingsRowHeight + 8
	v.displayPreviewCard = previewCard
	v.buildDisplayPreview(document, v.displayPreviewTop)
	v.finishPage(document, v.displayPreviewTop+120)
}

func (v *SettingsView) buildDisplayPreview(document objc.ID, top float64) {
	box := objc.ID(class("NSVisualEffectView")).Send(sel("new"))
	box.Send(sel("setMaterial:"), int64(14))
	box.Send(sel("setBlendingMode:"), int64(1))
	box.Send(sel("setState:"), int64(1))
	box.Send(sel("setWantsLayer:"), true)
	if layer := box.Send(sel("layer")); layer != 0 {
		layer.Send(sel("setCornerRadius:"), 8.0)
		layer.Send(sel("setMasksToBounds:"), true)
	}
	document.Send(sel("addSubview:"), box)
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
