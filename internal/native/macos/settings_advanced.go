//go:build darwin

package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

func (v *SettingsView) buildAdvancedPage() {
	document := v.newPage("advanced")
	top := 18.0

	_, cardTop := v.addSectionCard(document, top, i18n.T(i18n.SettingsCaptureTypes), 5)
	v.addSwitch(document, settingsRowTop(cardTop, 0), i18n.T(i18n.SettingsCaptureText), "capture.text",
		func(s SettingsValues) bool { return s.CaptureText })
	v.addSwitch(document, settingsRowTop(cardTop, 1), i18n.T(i18n.SettingsCaptureHTML), "capture.html",
		func(s SettingsValues) bool { return s.CaptureHTML })
	v.addSwitch(document, settingsRowTop(cardTop, 2), i18n.T(i18n.SettingsCaptureRTF), "capture.rtf",
		func(s SettingsValues) bool { return s.CaptureRTF })
	v.addSwitch(document, settingsRowTop(cardTop, 3), i18n.T(i18n.KindImage), "capture.image",
		func(s SettingsValues) bool { return s.CaptureImage })
	v.addSwitch(document, settingsRowTop(cardTop, 4), i18n.T(i18n.KindFiles), "capture.files",
		func(s SettingsValues) bool { return s.CaptureFiles })
	top = cardTop + 5*settingsRowHeight + settingsGroupGap

	_, cardTop = v.addSectionCard(document, top, i18n.T(i18n.SettingsCaptureLimits), 2)
	v.addStepper(document, settingsRowTop(cardTop, 0), i18n.T(i18n.SettingsMaxText), "capture.maxTextMb", 1, 1024, " MB",
		func(s SettingsValues) uint32 { return s.MaxTextMB })
	v.addStepper(document, settingsRowTop(cardTop, 1), i18n.T(i18n.SettingsMaxImage), "capture.maxImageMb", 10, 4096, " MB",
		func(s SettingsValues) uint32 { return s.MaxImageMB })
	top = cardTop + 2*settingsRowHeight + settingsGroupGap
	top = v.buildCapturePriorities(document, top)

	_, cardTop = v.addSectionCard(document, top, i18n.T(i18n.SettingsHistoryCleanup), 3)
	v.addPopup(document, settingsRowTop(cardTop, 0), i18n.T(i18n.SettingsRetention), "history.retention",
		[]settingsOption{
			{Title: i18n.T(i18n.SettingsRetentionForever), Value: "forever"},
			{Title: i18n.T(i18n.SettingsRetention1Hour), Value: "1h"},
			{Title: i18n.T(i18n.SettingsRetention6Hours), Value: "6h"},
			{Title: i18n.T(i18n.SettingsRetention1Day), Value: "1d"},
			{Title: i18n.T(i18n.SettingsRetention3Days), Value: "3d"},
			{Title: i18n.T(i18n.SettingsRetention7Days), Value: "7d"},
			{Title: i18n.T(i18n.SettingsRetention2Weeks), Value: "2w"},
			{Title: i18n.T(i18n.SettingsRetention1Month), Value: "1mo"},
			{Title: i18n.T(i18n.SettingsRetention3Months), Value: "3mo"},
			{Title: i18n.T(i18n.SettingsRetention1Year), Value: "365d"},
		},
		func(s SettingsValues) string { return s.Retention })
	v.addStepper(document, settingsRowTop(cardTop, 1), i18n.T(i18n.SettingsMaxCount), "history.maxCount",
		100, 100000, i18n.T(i18n.SettingsUnitCount),
		func(s SettingsValues) uint32 { return s.MaxCount })
	v.addStepper(document, settingsRowTop(cardTop, 2), i18n.T(i18n.SettingsCleanupInterval), "history.cleanupIntervalHours",
		1, 168, i18n.T(i18n.SettingsUnitHours),
		func(s SettingsValues) uint32 { return s.CleanupIntervalHours })
	top = cardTop + 3*settingsRowHeight + settingsGroupGap

	_, cardTop = v.addSectionCard(document, top, i18n.T(i18n.SettingsHistorySensitive), 2)
	v.addSwitch(document, settingsRowTop(cardTop, 0), i18n.T(i18n.SettingsCollectSecrets), "sensitive.collectSecrets",
		func(s SettingsValues) bool { return s.CollectSecrets })
	v.addSwitch(document, settingsRowTop(cardTop, 1), i18n.T(i18n.SettingsRedactSecrets), "sensitive.redactSecrets",
		func(s SettingsValues) bool { return s.RedactSecrets })
	top = cardTop + 2*settingsRowHeight
	top = v.addPageHint(document, top+8, 32, i18n.T(i18n.SettingsHistoryHint)) + 18

	top = v.buildHoverActions(document, top)

	_, cardTop = v.addSectionCard(document, top, i18n.T(i18n.SettingsDeletionConfirmations), 6)
	v.addSwitch(document, settingsRowTop(cardTop, 0), i18n.T(i18n.SettingsDeleteConfirm), "content.deleteConfirm",
		func(s SettingsValues) bool { return s.DeleteConfirm })
	v.addSwitch(document, settingsRowTop(cardTop, 1), i18n.T(i18n.SettingsDeleteFavoriteConfirm), "content.deleteFavoriteConfirm",
		func(s SettingsValues) bool { return s.DeleteFavoriteConfirm })
	v.addSwitch(document, settingsRowTop(cardTop, 2), i18n.T(i18n.SettingsDeletePinnedConfirm), "content.deletePinnedConfirm",
		func(s SettingsValues) bool { return s.DeletePinnedConfirm })
	v.addSwitch(document, settingsRowTop(cardTop, 3), i18n.T(i18n.SettingsDeleteFavoriteItems), "content.deleteFavoriteItems",
		func(s SettingsValues) bool { return s.DeleteFavoriteItems })
	v.addSwitch(document, settingsRowTop(cardTop, 4), i18n.T(i18n.SettingsDeletePinnedItems), "content.deletePinnedItems",
		func(s SettingsValues) bool { return s.DeletePinnedItems })
	v.addSwitch(document, settingsRowTop(cardTop, 5), i18n.T(i18n.SettingsDeleteFavoriteOnlyFav), "content.deleteFavoriteOnlyFav",
		func(s SettingsValues) bool { return s.DeleteFavoriteOnlyFav })
	top = cardTop + 6*settingsRowHeight + settingsGroupGap

	top = v.buildStorageSection(document, top)
	top = v.buildPermissionsSection(document, top)
	v.finishPage(document, top)
}

func (v *SettingsView) buildCapturePriorities(document objc.ID, top float64) float64 {
	registerSettingsCaptureOnce.Do(registerSettingsCaptureTargets)
	v.addSectionLabel(document, top, i18n.T(i18n.SettingsCaptureOrder))
	top += 24
	top = v.addPageHint(document, top, 18, i18n.T(i18n.SettingsCaptureHint))

	const listHeight = 112.0
	card := CGRect{Origin: CGPoint{X: settingsContentX, Y: top},
		Size: CGSize{Width: settingsContentWidth - 104, Height: listHeight}}
	table := addSettingsListTable(document, card, settingsCaptureRowHeight,
		float64(card.Size.Width-4), "capture", i18n.T(i18n.SettingsCaptureOrder), settingsCaptureTarget)
	v.captureTable = table
	settingsCaptureByTable[table] = v

	buttonX := settingsContentX + settingsContentWidth - 90
	v.captureUp = v.addSettingsMoveButton(document,
		CGRect{Origin: CGPoint{X: buttonX, Y: top + 12}, Size: CGSize{Width: 90, Height: 28}},
		i18n.T(i18n.SettingsMoveUp), i18n.T(i18n.SettingsMoveUpA11y), settingsCaptureTarget, "captureUpPressed:")
	v.captureDown = v.addSettingsMoveButton(document,
		CGRect{Origin: CGPoint{X: buttonX, Y: top + 48}, Size: CGSize{Width: 90, Height: 28}},
		i18n.T(i18n.SettingsMoveDown), i18n.T(i18n.SettingsMoveDownA11y), settingsCaptureTarget, "captureDownPressed:")
	return top + listHeight + settingsGroupGap
}

func (v *SettingsView) buildHoverActions(document objc.ID, top float64) float64 {
	registerSettingsActionsOnce.Do(registerSettingsActionsTargets)
	v.addSectionLabel(document, top, i18n.T(i18n.SettingsHoverActions))
	top += 24
	top = v.addPageParagraph(document, top, 32, i18n.T(i18n.SettingsHoverIntro))

	const listHeight = 300.0
	card := CGRect{Origin: CGPoint{X: settingsContentX, Y: top},
		Size: CGSize{Width: settingsContentWidth, Height: listHeight}}
	table := addSettingsListTable(document, card, 30.0, settingsContentWidth-4,
		"actions", i18n.T(i18n.SettingsHoverTable), settingsActionsTarget)
	v.actionsTable = table
	settingsActionsByTable[table] = v

	buttonX := settingsContentX + settingsContentWidth - 90
	v.actionUp = v.addSettingsMoveButton(document,
		CGRect{Origin: CGPoint{X: buttonX - 98, Y: top + listHeight + 12}, Size: CGSize{Width: 90, Height: 28}},
		i18n.T(i18n.SettingsMoveUp), i18n.T(i18n.SettingsActionMoveUpA11y), settingsActionsTarget, "actionUpPressed:")
	v.actionDown = v.addSettingsMoveButton(document,
		CGRect{Origin: CGPoint{X: buttonX, Y: top + listHeight + 12}, Size: CGSize{Width: 90, Height: 28}},
		i18n.T(i18n.SettingsMoveDown), i18n.T(i18n.SettingsActionMoveDownA11y), settingsActionsTarget, "actionDownPressed:")
	return top + listHeight + 52
}

func (v *SettingsView) buildStorageSection(document objc.ID, top float64) float64 {
	v.addSectionLabel(document, top, i18n.T(i18n.SettingsStorage))
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
	v.addLabel(document, rowTop, i18n.T(i18n.SettingsDataDir))
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

	v.restore = objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(i18n.T(i18n.SettingsRestoreDefault)), settingsTarget, sel("defaultDataDirPressed:"))
	v.restore.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: settingsContentX + settingsContentWidth - 166, Y: rowTop + 40},
		Size: CGSize{Width: 150, Height: 28}})
	document.Send(sel("addSubview:"), v.restore)
	settingsViews[v.restore] = v
	top = cardTop + 88 + settingsGroupGap

	v.addSectionLabel(document, top, i18n.T(i18n.SettingsBackupIntro))
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

func (v *SettingsView) buildPermissionsSection(document objc.ID, top float64) float64 {
	v.addSectionLabel(document, top, i18n.T(i18n.SettingsPermissionsSystem))
	cardTop := top + 24
	addSettingsListCard(document, CGRect{Origin: CGPoint{X: settingsContentX, Y: cardTop},
		Size: CGSize{Width: settingsContentWidth, Height: 2 * settingsPermissionRowHeight}})
	v.addCardSeparatorOffset(document, cardTop, settingsPermissionRowHeight)
	v.accessibilityStatus = v.addPermission(document, cardTop,
		i18n.T(i18n.SettingsAccessibility), i18n.T(i18n.SettingsAccessibilityDetail),
		"accessibilityPermissionPressed:")
	v.fullDiskStatus = v.addPermission(document, cardTop+settingsPermissionRowHeight,
		i18n.T(i18n.SettingsFullDisk), i18n.T(i18n.SettingsFullDiskDetail),
		"fullDiskAccessPermissionPressed:")
	top = cardTop + 2*settingsPermissionRowHeight
	top = v.addPageHint(document, top+8, 24, i18n.T(i18n.SettingsPermissionsIntro)) + 18

	v.addSectionLabel(document, top, i18n.T(i18n.SettingsDiagnostics))
	top += 24
	addSettingsListCard(document, CGRect{Origin: CGPoint{X: settingsContentX, Y: top},
		Size: CGSize{Width: settingsContentWidth, Height: 82}})
	versionLabel := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(diagnosticVersion()))
	versionLabel.Send(sel("setFont:"), systemFont(11))
	versionLabel.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	versionLabel.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: settingsContentX + settingsRowInset, Y: top + 12},
		Size: CGSize{Width: settingsContentWidth - 2*settingsRowInset, Height: 15}})
	document.Send(sel("addSubview:"), versionLabel)

	systemLabel := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(diagnosticSystemVersion()))
	systemLabel.Send(sel("setFont:"), systemFont(11))
	systemLabel.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	systemLabel.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: settingsContentX + settingsRowInset, Y: top + 30},
		Size: CGSize{Width: settingsContentWidth - 2*settingsRowInset, Height: 15}})
	document.Send(sel("addSubview:"), systemLabel)

	button := v.addPageButton(document, settingsContentX+settingsRowInset, top+49, 128, 25,
		i18n.SettingsCopyDiagnostics, "copyDiagnosticsPressed:", i18n.SettingsCopyDiagnosticsHelp)
	button.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.SettingsCopyDiagnostics)))
	v.diagnosticsHint = objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
	v.diagnosticsHint.Send(sel("setFont:"), systemFont(11))
	v.diagnosticsHint.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	v.diagnosticsHint.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: settingsContentX + settingsRowInset + 140, Y: top + 54},
		Size: CGSize{Width: settingsContentWidth - 2*settingsRowInset - 140, Height: 15}})
	document.Send(sel("addSubview:"), v.diagnosticsHint)

	v.refreshPermissionLabels()
	v.startPermissionTimer()
	return top + 82 + settingsGroupGap
}
