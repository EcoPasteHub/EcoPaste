//go:build darwin

package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
)

func (v *SettingsView) buildInterfacePage() {
	document := v.newPage("interface")
	top := 18.0

	_, cardTop := v.addSectionCard(document, top, i18n.SettingsDisplayDensity, 3)
	v.addNumberInput(document, settingsRowTop(cardTop, 0), i18n.SettingsTextMaxLines, "display.textMaxLines",
		1, 5, 1, i18n.SettingsUnitLines,
		func(s SettingsValues) uint32 { return s.TextMaxLines })
	v.addNumberInput(document, settingsRowTop(cardTop, 1), i18n.SettingsImageMaxHeight, "display.imageMaxHeight",
		20, 100, 4, " px",
		func(s SettingsValues) uint32 { return s.ImageMaxHeight })
	v.addNumberInput(document, settingsRowTop(cardTop, 2), i18n.SettingsFileMaxCount, "display.fileMaxCount",
		1, 5, 1, i18n.SettingsUnitItems,
		func(s SettingsValues) uint32 { return s.FileMaxCount })
	top = cardTop + 3*settingsRowHeight + settingsGroupGap

	_, cardTop = v.addSectionCardInfo(document, top, i18n.SettingsWindowOpen, i18n.SettingsWindowHint, 4)
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
	top = cardTop + 4*settingsRowHeight + settingsGroupGap

	_, cardTop = v.addSectionCard(document, top, i18n.SettingsWindowSearch, 2)
	v.addSwitch(document, settingsRowTop(cardTop, 0), i18n.SettingsSearchFocus, "search.defaultFocus",
		func(s SettingsValues) bool { return s.SearchFocus })
	v.addSwitch(document, settingsRowTop(cardTop, 1), i18n.SettingsSearchClear, "search.clearOnHide",
		func(s SettingsValues) bool { return s.SearchClear })
	top = cardTop + 2*settingsRowHeight + settingsGroupGap

	_, cardTop = v.addSectionCardInfo(document, top, i18n.SettingsInteractionMouse, i18n.SettingsInteractionIntro, 4)
	v.addPopup(document, settingsRowTop(cardTop, 0), i18n.SettingsMouseLeft, "content.autoPaste",
		[]settingsOption{
			{Key: i18n.SettingsDisabled, Value: "disabled"},
			{Key: i18n.SettingsSingleClickPaste, Value: "singleClickPaste"},
			{Key: i18n.SettingsDoubleClickPaste, Value: "doubleClickPaste"},
			{Key: i18n.SettingsSingleClickCopy, Value: "singleClickCopy"},
			{Key: i18n.SettingsDoubleClickCopy, Value: "doubleClickCopy"},
		},
		func(s SettingsValues) string { return s.AutoPaste })
	v.addSwitch(document, settingsRowTop(cardTop, 1), i18n.SettingsAutoFavorite, "content.autoFavorite",
		func(s SettingsValues) bool { return s.AutoFavorite })
	v.addSwitch(document, settingsRowTop(cardTop, 2), i18n.SettingsShowOriginal, "content.showOriginalPreview",
		func(s SettingsValues) bool { return s.ShowOriginalPreview })
	v.addSwitch(document, settingsRowTop(cardTop, 3), i18n.SettingsCopyTextPlain, "copyPlain",
		func(s SettingsValues) bool { return s.CopyPlain })
	top = cardTop + 4*settingsRowHeight + settingsGroupGap

	_, cardTop = v.addSectionCard(document, top, i18n.SettingsInteractionKeyboard, 1)
	v.addSwitch(document, settingsRowTop(cardTop, 0), i18n.SettingsSpacePreview, "preview.spaceEnabled",
		func(s SettingsValues) bool { return s.SpacePreview })
	v.finishPage(document, cardTop+settingsRowHeight)
}
