//go:build darwin

package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

// buildInteraction adds mouse click behavior and paste defaults.
func (v *SettingsView) buildInteraction() {
	content := v.newTab("interaction", settingsTabTitle("interaction"))

	intro := objc.ID(classNSTextField).Send(sel("labelWithString:"),
		nsString(i18n.T(i18n.SettingsInteractionIntro)))
	intro.Send(sel("setFont:"), systemFont(11))
	intro.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	intro.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: 438}, Size: CGSize{Width: tabContentW - 48, Height: 18}})
	content.Send(sel("addSubview:"), intro)

	y := 396.0
	v.addPopup(content, y, i18n.T(i18n.SettingsMouseLeft), "content.autoPaste",
		[]settingsOption{
			{Title: i18n.T(i18n.SettingsDisabled), Value: "disabled"},
			{Title: i18n.T(i18n.SettingsSingleClickPaste), Value: "singleClickPaste"},
			{Title: i18n.T(i18n.SettingsDoubleClickPaste), Value: "doubleClickPaste"},
			{Title: i18n.T(i18n.SettingsSingleClickCopy), Value: "singleClickCopy"},
			{Title: i18n.T(i18n.SettingsDoubleClickCopy), Value: "doubleClickCopy"},
		},
		func(s SettingsValues) string { return s.AutoPaste })
	v.addPopup(content, y-52, i18n.T(i18n.SettingsMouseMiddle), "content.middleClick",
		[]settingsOption{
			{Title: i18n.T(i18n.SettingsDisabled), Value: "disabled"},
			{Title: i18n.T(i18n.MenuPaste), Value: "singleClickPaste"},
			{Title: i18n.T(i18n.SettingsSinglePastePlain), Value: "singleClickPastePlain"},
			{Title: i18n.T(i18n.MenuCopy), Value: "singleClickCopy"},
			{Title: i18n.T(i18n.SettingsSingleCopyPlain), Value: "singleClickCopyPlain"},
		},
		func(s SettingsValues) string { return s.MiddleClick })
	v.addSwitch(content, y-104, i18n.T(i18n.SettingsPasteFilesAsPath), "content.pasteFilesAsPath",
		func(s SettingsValues) bool { return s.PasteFilesAsPath })
	v.addSwitch(content, y-142, i18n.T(i18n.SettingsAutoFavorite), "content.autoFavorite",
		func(s SettingsValues) bool { return s.AutoFavorite })
	v.addSwitch(content, y-180, i18n.T(i18n.SettingsShowOriginal), "content.showOriginalPreview",
		func(s SettingsValues) bool { return s.ShowOriginalPreview })

	note := objc.ID(classNSTextField).Send(sel("labelWithString:"),
		nsString(i18n.T(i18n.SettingsInteractionNote)))
	note.Send(sel("setFont:"), systemFont(11))
	note.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("tertiaryLabelColor")))
	note.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: y - 234}, Size: CGSize{Width: tabContentW - 48, Height: 32}})
	content.Send(sel("addSubview:"), note)
}

// buildDeletion adds the per-category delete protection rules.
func (v *SettingsView) buildDeletion() {
	content := v.newTab("deletion", settingsTabTitle("deletion"))

	intro := objc.ID(classNSTextField).Send(sel("labelWithString:"),
		nsString(i18n.T(i18n.SettingsDeletionIntro)))
	intro.Send(sel("setFont:"), systemFont(11))
	intro.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	intro.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: 438}, Size: CGSize{Width: tabContentW - 48, Height: 18}})
	content.Send(sel("addSubview:"), intro)

	y := 396.0
	v.addSwitch(content, y, i18n.T(i18n.SettingsDeleteConfirm), "content.deleteConfirm",
		func(s SettingsValues) bool { return s.DeleteConfirm })
	v.addSwitch(content, y-38, i18n.T(i18n.SettingsDeleteFavoriteConfirm), "content.deleteFavoriteConfirm",
		func(s SettingsValues) bool { return s.DeleteFavoriteConfirm })
	v.addSwitch(content, y-76, i18n.T(i18n.SettingsDeletePinnedConfirm), "content.deletePinnedConfirm",
		func(s SettingsValues) bool { return s.DeletePinnedConfirm })
	v.addSwitch(content, y-120, i18n.T(i18n.SettingsDeleteFavoriteItems), "content.deleteFavoriteItems",
		func(s SettingsValues) bool { return s.DeleteFavoriteItems })
	v.addSwitch(content, y-158, i18n.T(i18n.SettingsDeletePinnedItems), "content.deletePinnedItems",
		func(s SettingsValues) bool { return s.DeletePinnedItems })
	v.addSwitch(content, y-196, i18n.T(i18n.SettingsDeleteFavoriteOnlyFav), "content.deleteFavoriteOnlyFav",
		func(s SettingsValues) bool { return s.DeleteFavoriteOnlyFav })
}
