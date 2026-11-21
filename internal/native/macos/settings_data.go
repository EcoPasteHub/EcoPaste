//go:build darwin

package macos

import "github.com/EcoPasteHub/EcoPaste/internal/i18n"

func (v *SettingsView) buildDataPage() {
	document := v.newPage("data")
	top := 18.0

	_, cardTop := v.addSectionCard(document, top, i18n.SettingsHistoryTitle, 1)
	v.addPopup(document, settingsRowTop(cardTop, 0), i18n.SettingsSort, "sort",
		[]settingsOption{
			{Key: i18n.SettingsSortRecent, Value: "updatedAtDesc"},
			{Key: i18n.SettingsSortCreated, Value: "createdAtDesc"},
			{Key: i18n.SettingsSortUseCount, Value: "useCountDesc"},
		}, func(s SettingsValues) string { return s.Sort })
	top = cardTop + settingsRowHeight + settingsGroupGap

	_, cardTop = v.addSectionCard(document, top, i18n.SettingsHistoryCleanup, 2)
	v.addPopup(document, settingsRowTop(cardTop, 0), i18n.SettingsRetention, "history.retention",
		[]settingsOption{
			{Key: i18n.SettingsRetentionForever, Value: "forever"},
			{Key: i18n.SettingsRetention1Hour, Value: "1h"},
			{Key: i18n.SettingsRetention6Hours, Value: "6h"},
			{Key: i18n.SettingsRetention1Day, Value: "1d"},
			{Key: i18n.SettingsRetention3Days, Value: "3d"},
			{Key: i18n.SettingsRetention7Days, Value: "7d"},
			{Key: i18n.SettingsRetention2Weeks, Value: "2w"},
			{Key: i18n.SettingsRetention1Month, Value: "1mo"},
			{Key: i18n.SettingsRetention3Months, Value: "3mo"},
			{Key: i18n.SettingsRetention1Year, Value: "365d"},
		}, func(s SettingsValues) string { return s.Retention })
	v.addNumberInput(document, settingsRowTop(cardTop, 1), i18n.SettingsMaxCount, "history.maxCount",
		0, 100000, 100, i18n.SettingsUnitCount,
		func(s SettingsValues) uint32 { return s.MaxCount })
	top = cardTop + 2*settingsRowHeight + settingsGroupGap

	_, cardTop = v.addSectionCardInfo(document, top, i18n.SettingsHistorySensitive, i18n.SettingsHistoryHint, 2)
	v.addSwitch(document, settingsRowTop(cardTop, 0), i18n.SettingsCollectSecrets, "sensitive.collectSecrets",
		func(s SettingsValues) bool { return s.CollectSecrets })
	v.addSwitch(document, settingsRowTop(cardTop, 1), i18n.SettingsRedactSecrets, "sensitive.redactSecrets",
		func(s SettingsValues) bool { return s.RedactSecrets })
	top = cardTop + 2*settingsRowHeight + settingsGroupGap

	_, cardTop = v.addSectionCardInfo(document, top, i18n.SettingsDeletionConfirmations, i18n.SettingsDeletionIntro, 6)
	v.addSwitch(document, settingsRowTop(cardTop, 0), i18n.SettingsDeleteConfirm, "content.deleteConfirm",
		func(s SettingsValues) bool { return s.DeleteConfirm })
	v.addSwitch(document, settingsRowTop(cardTop, 1), i18n.SettingsDeleteFavoriteConfirm, "content.deleteFavoriteConfirm",
		func(s SettingsValues) bool { return s.DeleteFavoriteConfirm })
	v.addSwitch(document, settingsRowTop(cardTop, 2), i18n.SettingsDeletePinnedConfirm, "content.deletePinnedConfirm",
		func(s SettingsValues) bool { return s.DeletePinnedConfirm })
	v.addSwitch(document, settingsRowTop(cardTop, 3), i18n.SettingsDeleteFavoriteItems, "content.deleteFavoriteItems",
		func(s SettingsValues) bool { return s.DeleteFavoriteItems })
	v.addSwitch(document, settingsRowTop(cardTop, 4), i18n.SettingsDeletePinnedItems, "content.deletePinnedItems",
		func(s SettingsValues) bool { return s.DeletePinnedItems })
	v.addSwitch(document, settingsRowTop(cardTop, 5), i18n.SettingsDeleteFavoriteOnlyFav, "content.deleteFavoriteOnlyFav",
		func(s SettingsValues) bool { return s.DeleteFavoriteOnlyFav })
	top = cardTop + 6*settingsRowHeight + settingsGroupGap
	v.buildStorageSection(document, top)

	v.finishPage(document)
}
