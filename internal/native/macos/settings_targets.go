//go:build darwin

package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

func emitSettingsChange(sender objc.ID, path string, value any) {
	if v := settingsViews[sender]; v != nil {
		v.cx.Emit("change", SettingsChange{Path: path, Value: value})
	}
}

func registerSettingsTargets() {
	target, err := objc.RegisterClass("EcoPasteSettingsTarget", class("NSObject"), nil, nil, settingsTargetMethods())
	if err != nil {
		panic(err)
	}
	settingsTarget = objc.ID(target).Send(sel("new"))
	// Permission status can only change while the user is in System Settings,
	// so refresh on the way back instead of polling TCC on a timer.
	objc.ID(class("NSNotificationCenter")).Send(sel("defaultCenter")).
		Send(sel("addObserver:selector:name:object:"), settingsTarget, sel("appDidBecomeActive:"),
			nsString("NSApplicationDidBecomeActiveNotification"), objc.ID(0))
}

func settingsTargetMethods() []objc.MethodDef {
	return []objc.MethodDef{
		{Cmd: sel("toggleChanged:"), Fn: settingsToggleChanged},
		{Cmd: sel("popupChanged:"), Fn: settingsPopupChanged},
		{Cmd: sel("dataDirPressed:"), Fn: settingsDataDirPressed},
		{Cmd: sel("defaultDataDirPressed:"), Fn: settingsDefaultDataDirPressed},
		{Cmd: sel("accessibilityPermissionPressed:"), Fn: settingsAccessibilityPermissionPressed},
		{Cmd: sel("fullDiskAccessPermissionPressed:"), Fn: settingsFullDiskAccessPermissionPressed},
		{Cmd: sel("appDidBecomeActive:"), Fn: settingsAppDidBecomeActive},
		{Cmd: sel("copyDiagnosticsPressed:"), Fn: settingsCopyDiagnosticsPressed},
		{Cmd: sel("addAppPressed:"), Fn: settingsAddAppPressed},
		{Cmd: sel("refreshAppsPressed:"), Fn: settingsRefreshAppsPressed},
		{Cmd: sel("exportBackupPressed:"), Fn: settingsExportBackupPressed},
		{Cmd: sel("importBackupPressed:"), Fn: settingsImportBackupPressed},
		{Cmd: sel("updateCheckPressed:"), Fn: settingsUpdateCheckPressed},
		{Cmd: sel("openSourcePressed:"), Fn: settingsOpenSourcePressed},
		{Cmd: sel("openIssuesPressed:"), Fn: settingsOpenIssuesPressed},
		{Cmd: sel("resetSettingsPressed:"), Fn: settingsResetPressed},
		{Cmd: sel("shortcutDeletePressed:"), Fn: settingsShortcutDeletePressed},
	}
}

func settingsShortcutDeletePressed(_ objc.ID, _ objc.SEL, sender objc.ID) {
	if recorder := shortcutDeleteButtons[sender]; recorder != nil {
		recorder.clear()
	}
}

func settingsToggleChanged(_ objc.ID, _ objc.SEL, sender objc.ID) {
	v := settingsViews[sender]
	if v == nil {
		return
	}
	field := v.fieldFor(sender)
	if field == nil {
		return
	}
	on := objc.Send[int64](sender, sel("state")) == 1
	v.cx.Emit("change", SettingsChange{Path: field.path, Value: on})
}

func settingsPopupChanged(_ objc.ID, _ objc.SEL, sender objc.ID) {
	v := settingsViews[sender]
	if v == nil {
		return
	}
	field := v.fieldFor(sender)
	if field == nil {
		return
	}
	index := int(objc.Send[int64](sender, sel("indexOfSelectedItem")))
	if index < 0 || index >= len(field.options) {
		return
	}
	v.layoutPopup(field)
	suppressSettingsPopupHover(sender)
	v.cx.Emit("change", SettingsChange{Path: field.path, Value: field.options[index]})
}

func settingsDataDirPressed(_ objc.ID, _ objc.SEL, sender objc.ID) {
	v := settingsViews[sender]
	if v == nil {
		return
	}
	parent, ok := OpenDataDirectoryPanel(i18n.T(i18n.SettingsChooseDataDir))
	if !ok {
		return
	}
	if !Confirm(i18n.T(i18n.SettingsChangeDataDirTitle),
		i18n.T(i18n.SettingsChangeDataDirBody), i18n.T(i18n.SettingsMigrateRestart)) {
		return
	}
	v.cx.Emit("change", SettingsChange{Path: "general.dataDir", Value: parent})
}

func settingsDefaultDataDirPressed(_ objc.ID, _ objc.SEL, sender objc.ID) {
	v := settingsViews[sender]
	if v == nil {
		return
	}
	if !Confirm(i18n.T(i18n.SettingsRestoreDirTitle),
		i18n.T(i18n.SettingsRestoreDirBody), i18n.T(i18n.SettingsMigrateRestart)) {
		return
	}
	v.cx.Emit("change", SettingsChange{Path: "general.defaultDataDir", Value: true})
}

func settingsAccessibilityPermissionPressed(objc.ID, objc.SEL, objc.ID) {
	OpenAccessibilitySettings()
}

func settingsFullDiskAccessPermissionPressed(objc.ID, objc.SEL, objc.ID) {
	OpenFullDiskAccessSettings()
}

func settingsAppDidBecomeActive(_ objc.ID, _ objc.SEL, _ objc.ID) {
	settingsTabMu.Lock()
	views := make([]*SettingsView, 0, len(settingsOpenViews))
	for view := range settingsOpenViews {
		views = append(views, view)
	}
	settingsTabMu.Unlock()
	for _, view := range views {
		if view.currentTab == "advanced" {
			view.refreshPermissionLabels()
		}
	}
}

func settingsCopyDiagnosticsPressed(_ objc.ID, _ objc.SEL, sender objc.ID) {
	if v := settingsViews[sender]; v != nil {
		v.copyDiagnostics()
	}
}

func settingsAddAppPressed(_ objc.ID, _ objc.SEL, sender objc.ID) {
	path, ok := OpenApplicationPanel(i18n.T(i18n.SettingsChooseExcludedApp))
	if !ok {
		return
	}
	emitSettingsChange(sender, "apps.add", path)
}

func settingsRefreshAppsPressed(_ objc.ID, _ objc.SEL, sender objc.ID) {
	emitSettingsChange(sender, "apps.refresh", true)
}

func settingsExportBackupPressed(_ objc.ID, _ objc.SEL, sender objc.ID) {
	emitSettingsChange(sender, "backup.export", true)
}

func settingsImportBackupPressed(_ objc.ID, _ objc.SEL, sender objc.ID) {
	emitSettingsChange(sender, "backup.import", true)
}

func settingsUpdateCheckPressed(_ objc.ID, _ objc.SEL, sender objc.ID) {
	emitSettingsChange(sender, "update.check", true)
}

func settingsOpenSourcePressed(objc.ID, objc.SEL, objc.ID) {
	_ = OpenURL(SourceURL)
}

func settingsOpenIssuesPressed(objc.ID, objc.SEL, objc.ID) {
	_ = OpenURL(IssuesURL)
}

func settingsResetPressed(_ objc.ID, _ objc.SEL, sender objc.ID) {
	if settingsViews[sender] == nil {
		return
	}
	if !Confirm(i18n.T(i18n.SettingsResetTitle),
		i18n.T(i18n.SettingsResetBody), i18n.T(i18n.SettingsResetConfirm)) {
		return
	}
	emitSettingsChange(sender, "settings.reset", true)
}
