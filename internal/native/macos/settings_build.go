//go:build darwin

package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

func (v *SettingsView) buildData() {
	content := v.newTab("data", settingsTabTitle("data"))
	v.addSectionLabel(content, 438, i18n.T(i18n.SettingsStorage))
	v.buildDataDir(content, 408)

	v.addSectionLabel(content, 320, i18n.T(i18n.SettingsBackupIntro))
	v.addSearchEntry(content, i18n.T(i18n.SettingsBackupIntro), "data.backup",
		i18n.T(i18n.SettingsBackupHint), i18n.T(i18n.SettingsBackupKeywords),
		CGRect{Origin: CGPoint{X: 12, Y: 232}, Size: CGSize{Width: tabContentW - 24, Height: 86}})
	hint := objc.ID(classNSTextField).Send(sel("wrappingLabelWithString:"), nsString(i18n.T(i18n.SettingsBackupHint)))
	hint.Send(sel("setFont:"), systemFont(11))
	hint.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	hint.Send(sel("setPreferredMaxLayoutWidth:"), tabContentW-48)
	hint.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: 288}, Size: CGSize{Width: tabContentW - 48, Height: 30}})
	content.Send(sel("addSubview:"), hint)

	export := objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(i18n.T(i18n.SettingsExportBackup)), settingsTarget, sel("exportBackupPressed:"))
	export.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: 240}, Size: CGSize{Width: 140, Height: 28}})
	export.Send(sel("setBezelStyle:"), int64(1))
	export.Send(sel("setAccessibilityHelp:"), nsString(i18n.T(i18n.SettingsBackupHint)))
	content.Send(sel("addSubview:"), export)
	settingsViews[export] = v

	importButton := objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(i18n.T(i18n.SettingsImportBackup)), settingsTarget, sel("importBackupPressed:"))
	importButton.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 176, Y: 240}, Size: CGSize{Width: 140, Height: 28}})
	importButton.Send(sel("setBezelStyle:"), int64(1))
	importButton.Send(sel("setAccessibilityHelp:"), nsString(i18n.T(i18n.SettingsBackupHint)))
	content.Send(sel("addSubview:"), importButton)
	settingsViews[importButton] = v
}

func (v *SettingsView) buildDataDir(content objc.ID, y float64) {
	v.addLabel(content, y, i18n.T(i18n.SettingsDataDir))
	v.addSearchEntry(content, i18n.T(i18n.SettingsDataDir), "general.dataDir",
		i18n.T(i18n.SettingsDataDirKeywords), "",
		CGRect{Origin: CGPoint{X: 12, Y: y - 26}, Size: CGSize{Width: tabContentW - 24, Height: 48}})
	v.dataDir = objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
	v.dataDir.Send(sel("setFont:"), systemFont(11))
	v.dataDir.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("tertiaryLabelColor")))
	v.dataDir.Send(sel("setLineBreakMode:"), int64(4))
	v.dataDir.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: y - 22}, Size: CGSize{Width: tabContentW - 48, Height: 16}})
	content.Send(sel("addSubview:"), v.dataDir)

	v.restore = objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(i18n.T(i18n.SettingsRestoreDefault)), settingsTarget, sel("defaultDataDirPressed:"))
	v.restore.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: tabContentW - 224, Y: y - 60}, Size: CGSize{Width: 120, Height: 28}})
	content.Send(sel("addSubview:"), v.restore)
	settingsViews[v.restore] = v

	v.change = objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(i18n.T(i18n.SettingsChange)), settingsTarget, sel("dataDirPressed:"))
	v.change.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: tabContentW - 94, Y: y - 60}, Size: CGSize{Width: 70, Height: 28}})
	content.Send(sel("addSubview:"), v.change)
	settingsViews[v.change] = v
}

func (v *SettingsView) buildCapture() {
	content := v.newTab("capture", settingsTabTitle("capture"))
	v.addSectionLabel(content, 438, i18n.T(i18n.SettingsCaptureTypes))
	y := 408.0
	v.addSwitch(content, y, i18n.T(i18n.SettingsCaptureText), "capture.text",
		func(s SettingsValues) bool { return s.CaptureText })
	v.addSwitch(content, y-36, i18n.T(i18n.SettingsCaptureHTML), "capture.html",
		func(s SettingsValues) bool { return s.CaptureHTML })
	v.addSwitch(content, y-72, i18n.T(i18n.SettingsCaptureRTF), "capture.rtf",
		func(s SettingsValues) bool { return s.CaptureRTF })
	v.addSwitch(content, y-108, i18n.T(i18n.KindImage), "capture.image",
		func(s SettingsValues) bool { return s.CaptureImage })
	v.addSwitch(content, y-144, i18n.T(i18n.KindFiles), "capture.files",
		func(s SettingsValues) bool { return s.CaptureFiles })
	v.addSectionLabel(content, 232, i18n.T(i18n.SettingsCaptureLimits))
	v.addStepper(content, y-202, i18n.T(i18n.SettingsMaxText), "capture.maxTextMb", 1, 1024, " MB",
		func(s SettingsValues) uint32 { return s.MaxTextMB })
	v.addStepper(content, y-242, i18n.T(i18n.SettingsMaxImage), "capture.maxImageMb", 10, 4096, " MB",
		func(s SettingsValues) uint32 { return s.MaxImageMB })
	v.buildCaptureOrder(content)
}

func (v *SettingsView) buildWindow() {
	content := v.newTab("window", settingsTabTitle("window"))
	v.addSectionLabel(content, 438, i18n.T(i18n.SettingsWindowOpen))
	y := 402.0
	v.addSwitch(content, y, i18n.T(i18n.SettingsScrollTop), "window.scrollToTopOnOpen",
		func(s SettingsValues) bool { return s.ScrollToTopOnOpen })
	v.addPopup(content, y-48, i18n.T(i18n.SettingsSelectRange), "window.selectRangeOnOpen",
		[]settingsOption{
			{Title: i18n.T(i18n.SettingsPreserve), Value: "preserve"},
			{Title: i18n.T(i18n.SettingsAll), Value: "all"},
			{Title: i18n.T(i18n.PanelFavorites), Value: "favorite"},
		},
		func(s SettingsValues) string { return s.RangeOnOpen })
	v.addPopup(content, y-90, i18n.T(i18n.SettingsSelectCategory), "window.selectCategoryOnOpen",
		[]settingsOption{
			{Title: i18n.T(i18n.SettingsPreserve), Value: "preserve"},
			{Title: i18n.T(i18n.SettingsAll), Value: "all"},
			{Title: i18n.T(i18n.PanelTextCategory), Value: "text"},
			{Title: i18n.T(i18n.PanelImageCategory), Value: "image"},
			{Title: i18n.T(i18n.PanelFilesCategory), Value: "files"},
		},
		func(s SettingsValues) string { return s.CategoryOnOpen })
	v.groupField = v.addPopup(content, y-132, i18n.T(i18n.SettingsSelectGroup), "window.selectGroupOnOpen",
		[]settingsOption{
			{Title: i18n.T(i18n.SettingsPreserve), Value: "preserve"},
			{Title: i18n.T(i18n.SettingsAllGroups), Value: "all"},
		},
		func(s SettingsValues) string { return s.GroupOnOpen })
	v.addHint(content, 226, 32, i18n.T(i18n.SettingsWindowHint))
	v.addSectionLabel(content, 190, i18n.T(i18n.SettingsWindowSearch))
	v.addSwitch(content, 158, i18n.T(i18n.SettingsSearchFocus), "search.defaultFocus",
		func(s SettingsValues) bool { return s.SearchFocus })
	v.addSwitch(content, 122, i18n.T(i18n.SettingsSearchClear), "search.clearOnHide",
		func(s SettingsValues) bool { return s.SearchClear })
}

func (v *SettingsView) buildHistory() {
	content := v.newTab("history", settingsTabTitle("history"))
	v.addSectionLabel(content, 438, i18n.T(i18n.SettingsHistoryCleanup))
	y := 402.0
	v.addPopup(content, y, i18n.T(i18n.SettingsRetention), "history.retention",
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
	v.addStepper(content, y-48, i18n.T(i18n.SettingsMaxCount), "history.maxCount",
		100, 100000, i18n.T(i18n.SettingsUnitCount),
		func(s SettingsValues) uint32 { return s.MaxCount })
	v.addStepper(content, y-90, i18n.T(i18n.SettingsCleanupInterval), "history.cleanupIntervalHours",
		1, 168, i18n.T(i18n.SettingsUnitHours),
		func(s SettingsValues) uint32 { return s.CleanupIntervalHours })

	v.addSectionLabel(content, 274, i18n.T(i18n.SettingsHistorySensitive))
	v.addSwitch(content, 240, i18n.T(i18n.SettingsCollectSecrets), "sensitive.collectSecrets",
		func(s SettingsValues) bool { return s.CollectSecrets })
	v.addSwitch(content, 204, i18n.T(i18n.SettingsRedactSecrets), "sensitive.redactSecrets",
		func(s SettingsValues) bool { return s.RedactSecrets })
	v.addHint(content, 146, 32, i18n.T(i18n.SettingsHistoryHint))
}

func (v *SettingsView) buildPreview() {
	content := v.newTab("preview", settingsTabTitle("preview"))
	v.addSectionLabel(content, 438, i18n.T(i18n.SettingsPreviewSection))
	intro := objc.ID(classNSTextField).Send(sel("wrappingLabelWithString:"),
		nsString(i18n.T(i18n.SettingsPreviewIntro)))
	intro.Send(sel("setFont:"), systemFont(11))
	intro.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	intro.Send(sel("setPreferredMaxLayoutWidth:"), tabContentW-48)
	intro.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: 408}, Size: CGSize{Width: tabContentW - 48, Height: 28}})
	content.Send(sel("addSubview:"), intro)
	v.addSwitch(content, 372, i18n.T(i18n.SettingsSpacePreview), "preview.spaceEnabled",
		func(s SettingsValues) bool { return s.SpacePreview })
}

func (v *SettingsView) buildUpdate() {
	content := v.newTab("update", settingsTabTitle("update"))
	iconImage := objc.Send[objc.ID](
		objc.ID(class("NSApplication")).Send(sel("sharedApplication")), sel("applicationIconImage"))
	icon := objc.ID(class("NSImageView")).Send(sel("new"))
	icon.Send(sel("setImage:"), iconImage)
	icon.Send(sel("setImageScaling:"), int64(3))
	icon.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: 346}, Size: CGSize{Width: 64, Height: 64}})
	content.Send(sel("addSubview:"), icon)
	icon.Send(sel("release"))

	name := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString("EcoPaste"))
	name.Send(sel("setFont:"), objc.ID(classNSFont).Send(sel("boldSystemFontOfSize:"), 17.0))
	name.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 108, Y: 384}, Size: CGSize{Width: 300, Height: 22}})
	content.Send(sel("addSubview:"), name)

	version := objc.ID(classNSTextField).Send(sel("labelWithString:"),
		nsString(i18n.Tf(i18n.SettingsUpdateVersion, AppVersion())))
	version.Send(sel("setFont:"), systemFont(12))
	version.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	version.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 108, Y: 362}, Size: CGSize{Width: 300, Height: 18}})
	content.Send(sel("addSubview:"), version)

	intro := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(i18n.T(i18n.SettingsUpdateIntro)))
	intro.Send(sel("setFont:"), systemFont(12))
	intro.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	intro.Send(sel("setLineBreakMode:"), int64(4))
	intro.Send(sel("setMaximumNumberOfLines:"), int64(2))
	intro.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: 312}, Size: CGSize{Width: tabContentW - 48, Height: 36}})
	content.Send(sel("addSubview:"), intro)

	check := objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(i18n.T(i18n.SettingsUpdateCheck)), settingsTarget, sel("updateCheckPressed:"))
	check.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: 260}, Size: CGSize{Width: 150, Height: 30}})
	check.Send(sel("setBezelStyle:"), int64(1))
	content.Send(sel("addSubview:"), check)
	settingsViews[check] = v

	managed := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(i18n.T(i18n.SettingsUpdateManaged)))
	managed.Send(sel("setFont:"), systemFont(11))
	managed.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("tertiaryLabelColor")))
	managed.Send(sel("setLineBreakMode:"), int64(4))
	managed.Send(sel("setMaximumNumberOfLines:"), int64(2))
	managed.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: 212}, Size: CGSize{Width: tabContentW - 48, Height: 34}})
	content.Send(sel("addSubview:"), managed)

	v.addSearchEntry(content, i18n.T(i18n.SettingsTabUpdate), "update.check",
		i18n.T(i18n.SettingsUpdateIntro), "update sparkle version",
		CGRect{Origin: CGPoint{X: 12, Y: 254}, Size: CGSize{Width: tabContentW - 24, Height: 84}})
}

func (v *SettingsView) buildPermissions() {
	content := v.newTab("permissions", settingsTabTitle("permissions"))
	v.addSectionLabel(content, 438, i18n.T(i18n.SettingsPermissionsSystem))
	intro := objc.ID(classNSTextField).Send(sel("labelWithString:"),
		nsString(i18n.T(i18n.SettingsPermissionsIntro)))
	intro.Send(sel("setFont:"), systemFont(11))
	intro.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	intro.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: 414}, Size: CGSize{Width: tabContentW - 48, Height: 18}})
	content.Send(sel("addSubview:"), intro)

	y := 388.0
	v.accessibilityStatus = v.addPermission(content, y, i18n.T(i18n.SettingsAccessibility),
		i18n.T(i18n.SettingsAccessibilityDetail), "accessibilityPermissionPressed:")
	v.fullDiskStatus = v.addPermission(content, y-80, i18n.T(i18n.SettingsFullDisk),
		i18n.T(i18n.SettingsFullDiskDetail), "fullDiskAccessPermissionPressed:")
	v.buildDiagnostics(content)
	v.refreshPermissionLabels()
	v.startPermissionTimer()
}

func (v *SettingsView) addPermission(content objc.ID, y float64, title, detail, action string) objc.ID {
	v.addLabel(content, y, title)
	path := map[string]string{
		"accessibilityPermissionPressed:":  "permissions.accessibility",
		"fullDiskAccessPermissionPressed:": "permissions.fullDiskAccess",
	}[action]
	v.addSearchEntry(content, title, path, detail, i18n.T(i18n.SettingsPermissionsKeywords),
		CGRect{Origin: CGPoint{X: 12, Y: y - 32}, Size: CGSize{Width: tabContentW - 24, Height: 64}})
	detailLabel := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(detail))
	detailLabel.Send(sel("setFont:"), systemFont(11))
	detailLabel.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	detailLabel.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: y - 20}, Size: CGSize{Width: tabContentW - 200, Height: 16}})
	content.Send(sel("addSubview:"), detailLabel)

	status := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(i18n.T(i18n.SettingsChecking)))
	status.Send(sel("setFont:"), systemFont(11))
	status.Send(sel("setAlignment:"), int64(2))
	status.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: tabContentW - 300, Y: y}, Size: CGSize{Width: 102, Height: 18}})
	content.Send(sel("addSubview:"), status)

	button := objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(i18n.T(i18n.SettingsOpenSystemSettings)), settingsTarget, sel(action))
	button.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: tabContentW - 188, Y: y - 5}, Size: CGSize{Width: 164, Height: 28}})
	button.Send(sel("setAccessibilityLabel:"), nsString(i18n.Tf(i18n.SettingsOpenSystemSettingsA11y, title)))
	content.Send(sel("addSubview:"), button)
	return status
}

func (v *SettingsView) refreshPermissionLabels() {
	v.setPermissionStatus(v.accessibilityStatus, AccessibilityGranted())
	v.setPermissionStatus(v.fullDiskStatus, FullDiskAccessGranted())
}

func (v *SettingsView) setPermissionStatus(label objc.ID, granted bool) {
	if label == 0 {
		return
	}
	text, color := i18n.T(i18n.OnboardingNotGranted), "systemRedColor"
	if granted {
		text, color = i18n.T(i18n.OnboardingGranted), "systemGreenColor"
	}
	label.Send(sel("setStringValue:"), nsString(text))
	label.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel(color)))
}

func (v *SettingsView) startPermissionTimer() {
	timer := objc.ID(class("NSTimer")).Send(sel("scheduledTimerWithTimeInterval:target:selector:userInfo:repeats:"),
		1.5, settingsTarget, sel("permissionTimerFired:"), objc.ID(0), true)
	settingsTimers[timer] = v
	v.permissionTimer = timer
}
