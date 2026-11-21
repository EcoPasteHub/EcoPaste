//go:build darwin

package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

func (v *SettingsView) buildStorageSection(document objc.ID, top float64) float64 {
	v.addSectionLabel(document, top, i18n.SettingsStorage)
	cardTop := top + 24
	card := objc.ID(classNSBox).Send(sel("new"))
	card.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: settingsContentX, Y: cardTop},
		Size: CGSize{Width: settingsContentWidth, Height: 88}})
	card.Send(sel("setBoxType:"), int64(4))
	card.Send(sel("setTitlePosition:"), int64(0))
	card.Send(sel("setBorderWidth:"), 0.0)
	card.Send(sel("setFillColor:"), settingsCardFillColor())
	card.Send(sel("setCornerRadius:"), settingsCardRadius)
	document.Send(sel("addSubview:"), card)
	card.Send(sel("release"))

	rowTop := cardTop + 10
	v.addLabel(document, rowTop, i18n.SettingsDataDir)
	v.dataDir = objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
	v.dataDir.Send(sel("setFont:"), systemFont(11))
	v.dataDir.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	v.dataDir.Send(sel("setLineBreakMode:"), int64(4))
	v.dataDir.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: settingsContentX + settingsRowInset, Y: rowTop + 27},
		Size: CGSize{Width: settingsContentWidth - 2*settingsRowInset - 214, Height: 15}})
	document.Send(sel("addSubview:"), v.dataDir)

	v.change = objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(i18n.T(i18n.SettingsChange)), settingsTarget, sel("dataDirPressed:"))
	v.change.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: settingsContentX + settingsContentWidth - 166, Y: rowTop + 8},
		Size: CGSize{Width: 70, Height: 28}})
	document.Send(sel("addSubview:"), v.change)
	settingsViews[v.change] = v
	v.bindText(func() {
		v.change.Send(sel("setTitle:"), nsString(i18n.T(i18n.SettingsChange)))
	})

	v.restore = objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(i18n.T(i18n.SettingsRestoreDefault)), settingsTarget, sel("defaultDataDirPressed:"))
	v.restore.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: settingsContentX + settingsContentWidth - 166, Y: rowTop + 40},
		Size: CGSize{Width: 150, Height: 28}})
	document.Send(sel("addSubview:"), v.restore)
	settingsViews[v.restore] = v
	v.bindText(func() {
		v.restore.Send(sel("setTitle:"), nsString(i18n.T(i18n.SettingsRestoreDefault)))
	})
	top = cardTop + 88 + settingsGroupGap

	v.addSectionLabel(document, top, i18n.SettingsBackupIntro)
	cardTop = top + 24
	addSettingsListCard(document, CGRect{Origin: CGPoint{X: settingsContentX, Y: cardTop},
		Size: CGSize{Width: settingsContentWidth, Height: 56}})
	export := v.addPageButton(document, settingsContentX+settingsRowInset, cardTop+14, 136, 28,
		i18n.SettingsExportBackup, "exportBackupPressed:", i18n.SettingsBackupHint)
	export.Send(sel("setAccessibilityHelp:"), nsString(i18n.T(i18n.SettingsBackupHint)))
	importButton := v.addPageButton(document, settingsContentX+settingsRowInset+144, cardTop+14, 136, 28,
		i18n.SettingsImportBackup, "importBackupPressed:", i18n.SettingsBackupHint)
	importButton.Send(sel("setAccessibilityHelp:"), nsString(i18n.T(i18n.SettingsBackupHint)))
	return cardTop + 56 + settingsGroupGap
}
