//go:build darwin

package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

func registerSettingsTargets() {
	target, err := objc.RegisterClass("EcoPasteSettingsTarget", class("NSObject"), nil, nil, []objc.MethodDef{
		{Cmd: sel("settingsSearchChanged:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			if v := settingsViews[sender]; v != nil {
				v.searchChanged(sender)
			}
		}},
		{Cmd: sel("settingsSearchPicked:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			if v := settingsViews[sender]; v != nil {
				v.searchResultPicked(sender)
			}
		}},
		{Cmd: sel("settingsSearchHighlightExpired:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			if v := settingsTimers[sender]; v != nil {
				delete(settingsTimers, sender)
				v.searchHighlightTimer = 0
				if v.searchHighlight != 0 {
					v.searchHighlight.Send(sel("setHidden:"), true)
				}
			}
		}},
		{Cmd: sel("toggleChanged:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
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
		}},
		{Cmd: sel("popupChanged:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
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
			v.cx.Emit("change", SettingsChange{Path: field.path, Value: field.options[index]})
		}},
		{Cmd: sel("stepperChanged:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			v := settingsViews[sender]
			if v == nil {
				return
			}
			field := v.fieldFor(sender)
			if field == nil {
				return
			}
			value := uint32(objc.Send[float64](sender, sel("doubleValue")))
			v.setStepperLabel(field, value)
			v.cx.Emit("change", SettingsChange{Path: field.path, Value: value})
		}},
		{Cmd: sel("dataDirPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
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
		}},
		{Cmd: sel("defaultDataDirPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			v := settingsViews[sender]
			if v == nil {
				return
			}
			if !Confirm(i18n.T(i18n.SettingsRestoreDirTitle),
				i18n.T(i18n.SettingsRestoreDirBody), i18n.T(i18n.SettingsMigrateRestart)) {
				return
			}
			v.cx.Emit("change", SettingsChange{Path: "general.defaultDataDir", Value: true})
		}},
		{Cmd: sel("accessibilityPermissionPressed:"), Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) {
			OpenAccessibilitySettings()
		}},
		{Cmd: sel("fullDiskAccessPermissionPressed:"), Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) {
			OpenFullDiskAccessSettings()
		}},
		{Cmd: sel("permissionTimerFired:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			if v := settingsTimers[sender]; v != nil {
				v.refreshPermissionLabels()
			}
		}},
		{Cmd: sel("copyDiagnosticsPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			if v := settingsViews[sender]; v != nil {
				v.copyDiagnostics()
			}
		}},
		{Cmd: sel("addAppPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			v := settingsViews[sender]
			if v == nil {
				return
			}
			path, ok := OpenApplicationPanel(i18n.T(i18n.SettingsChooseExcludedApp))
			if !ok {
				return
			}
			v.cx.Emit("change", SettingsChange{Path: "apps.add", Value: path})
		}},
		{Cmd: sel("refreshAppsPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			v := settingsViews[sender]
			if v == nil {
				return
			}
			v.cx.Emit("change", SettingsChange{Path: "apps.refresh", Value: true})
		}},
		{Cmd: sel("exportBackupPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			v := settingsViews[sender]
			if v == nil {
				return
			}
			v.cx.Emit("change", SettingsChange{Path: "backup.export", Value: true})
		}},
		{Cmd: sel("importBackupPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			v := settingsViews[sender]
			if v == nil {
				return
			}
			v.cx.Emit("change", SettingsChange{Path: "backup.import", Value: true})
		}},
		{Cmd: sel("updateCheckPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			v := settingsViews[sender]
			if v == nil {
				return
			}
			v.cx.Emit("change", SettingsChange{Path: "update.check", Value: true})
		}},
		{Cmd: sel("resetSettingsPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			v := settingsViews[sender]
			if v == nil {
				return
			}
			if !Confirm(i18n.T(i18n.SettingsResetTitle),
				i18n.T(i18n.SettingsResetBody), i18n.T(i18n.SettingsResetConfirm)) {
				return
			}
			v.cx.Emit("change", SettingsChange{Path: "settings.reset", Value: true})
		}},
	})
	if err != nil {
		panic(err)
	}
	settingsTarget = objc.ID(target).Send(sel("new"))
}
