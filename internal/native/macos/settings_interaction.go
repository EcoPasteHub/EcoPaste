//go:build darwin

package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

func (v *SettingsView) buildInteractionPage() {
	document := v.newPage("interaction")
	top := 18.0

	_, cardTop := v.addSectionCardInfo(document, top, i18n.SettingsWindowOpen, i18n.SettingsWindowHint, 5)
	v.addPopup(document, settingsRowTop(cardTop, 0), i18n.SettingsPosition, "window.position",
		[]settingsOption{
			{Key: i18n.SettingsPositionFollow, Value: "followCursor"},
			{Key: i18n.SettingsPositionCenter, Value: "center"},
			{Key: i18n.SettingsPositionKeep, Value: "remember"},
		}, func(s SettingsValues) string { return s.WindowPosition })
	v.addSwitch(document, settingsRowTop(cardTop, 1), i18n.SettingsScrollTop, "window.scrollToTopOnOpen",
		func(s SettingsValues) bool { return s.ScrollToTopOnOpen })
	v.addPopup(document, settingsRowTop(cardTop, 2), i18n.SettingsSelectRange, "window.selectRangeOnOpen",
		[]settingsOption{
			{Key: i18n.SettingsPreserve, Value: "preserve"},
			{Key: i18n.SettingsAll, Value: "all"},
			{Key: i18n.PanelFavorites, Value: "favorite"},
		}, func(s SettingsValues) string { return s.RangeOnOpen })
	v.addPopup(document, settingsRowTop(cardTop, 3), i18n.SettingsSelectCategory, "window.selectCategoryOnOpen",
		[]settingsOption{
			{Key: i18n.SettingsPreserve, Value: "preserve"},
			{Key: i18n.SettingsAll, Value: "all"},
			{Key: i18n.PanelTextCategory, Value: "text"},
			{Key: i18n.PanelImageCategory, Value: "image"},
			{Key: i18n.PanelFilesCategory, Value: "files"},
		}, func(s SettingsValues) string { return s.CategoryOnOpen })
	v.groupField = v.addPopup(document, settingsRowTop(cardTop, 4), i18n.SettingsSelectGroup, "window.selectGroupOnOpen",
		[]settingsOption{
			{Key: i18n.SettingsPreserve, Value: "preserve"},
			{Key: i18n.SettingsAllGroups, Value: "all"},
		}, func(s SettingsValues) string { return s.GroupOnOpen })
	top = cardTop + 5*settingsRowHeight + settingsGroupGap

	_, cardTop = v.addSectionCard(document, top, i18n.SettingsWindowSearch, 2)
	v.addSwitch(document, settingsRowTop(cardTop, 0), i18n.SettingsSearchFocus, "search.defaultFocus",
		func(s SettingsValues) bool { return s.SearchFocus })
	v.addSwitch(document, settingsRowTop(cardTop, 1), i18n.SettingsSearchClear, "search.clearOnHide",
		func(s SettingsValues) bool { return s.SearchClear })
	top = cardTop + 2*settingsRowHeight + settingsGroupGap

	_, cardTop = v.addSectionCardInfo(document, top, i18n.SettingsInteractionContent, i18n.SettingsInteractionIntro, 6)
	v.addPopup(document, settingsRowTop(cardTop, 0), i18n.SettingsMouseLeft, "content.autoPaste",
		[]settingsOption{
			{Key: i18n.SettingsDisabled, Value: "disabled"},
			{Key: i18n.SettingsSingleClickPaste, Value: "singleClickPaste"},
			{Key: i18n.SettingsDoubleClickPaste, Value: "doubleClickPaste"},
			{Key: i18n.SettingsSingleClickCopy, Value: "singleClickCopy"},
			{Key: i18n.SettingsDoubleClickCopy, Value: "doubleClickCopy"},
		}, func(s SettingsValues) string { return s.AutoPaste })
	v.addSwitch(document, settingsRowTop(cardTop, 1), i18n.SettingsAutoFavorite, "content.autoFavorite",
		func(s SettingsValues) bool { return s.AutoFavorite })
	v.addSwitch(document, settingsRowTop(cardTop, 2), i18n.SettingsShowOriginal, "content.showOriginalPreview",
		func(s SettingsValues) bool { return s.ShowOriginalPreview })
	v.addSwitch(document, settingsRowTop(cardTop, 3), i18n.SettingsCopyTextPlain, "copyPlain",
		func(s SettingsValues) bool { return s.CopyPlain })
	v.addSwitch(document, settingsRowTop(cardTop, 4), i18n.SettingsCopyThenHide, "copyThenHideWindow",
		func(s SettingsValues) bool { return s.CopyThenHideWindow })
	v.addSwitch(document, settingsRowTop(cardTop, 5), i18n.SettingsCopySound, "feedback.copySound",
		func(s SettingsValues) bool { return s.CopySound })
	top = cardTop + 6*settingsRowHeight + settingsGroupGap

	_, cardTop = v.addSectionCard(document, top, i18n.SettingsInteractionKeyboard, 1)
	v.addSwitch(document, settingsRowTop(cardTop, 0), i18n.SettingsSpacePreview, "preview.spaceEnabled",
		func(s SettingsValues) bool { return s.SpacePreview })
	top = cardTop + settingsRowHeight + settingsGroupGap
	top = v.buildHoverActions(document, top)

	v.finishPage(document, top)
}

func (v *SettingsView) buildHoverActions(document objc.ID, top float64) float64 {
	registerSettingsActionsOnce.Do(registerSettingsActionsTargets)
	v.addSectionLabel(document, top, i18n.SettingsHoverActions)
	v.addInfoAfterText(document, settingsContentX+4, top+8, i18n.SettingsHoverActions,
		settingsSectionFont(), i18n.SettingsHoverIntro)
	top += 24

	const listHeight = 300.0
	card := CGRect{Origin: CGPoint{X: settingsContentX, Y: top},
		Size: CGSize{Width: settingsContentWidth, Height: listHeight}}
	table := v.addSettingsListTable(document, card, 30.0, settingsContentWidth-4,
		"actions", i18n.SettingsHoverTable, settingsActionsTarget)
	v.actionsTable = table
	settingsActionsByTable[table] = v

	buttonX := settingsContentX + settingsContentWidth - 90
	v.actionUp = v.addSettingsMoveButton(document,
		CGRect{Origin: CGPoint{X: buttonX - 98, Y: top + listHeight + 12}, Size: CGSize{Width: 90, Height: 28}},
		i18n.SettingsMoveUp, i18n.SettingsActionMoveUpA11y, settingsActionsTarget, "actionUpPressed:")
	v.actionDown = v.addSettingsMoveButton(document,
		CGRect{Origin: CGPoint{X: buttonX, Y: top + listHeight + 12}, Size: CGSize{Width: 90, Height: 28}},
		i18n.SettingsMoveDown, i18n.SettingsActionMoveDownA11y, settingsActionsTarget, "actionDownPressed:")
	return top + listHeight + 52
}
