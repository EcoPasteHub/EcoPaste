//go:build darwin

package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

const settingsPermissionRowHeight = 60.0

func (v *SettingsView) addPermission(content objc.ID, y float64, title, detail, action string) objc.ID {
	const (
		buttonWidth = 132.0
		statusWidth = 60.0
		controlGap  = 12.0
	)
	buttonX := settingsContentX + settingsContentWidth - settingsRowInset - buttonWidth
	statusX := buttonX - controlGap - statusWidth
	detailWidth := statusX - (settingsContentX + settingsRowInset) - controlGap

	titleLabel := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(title))
	titleLabel.Send(sel("setFont:"), systemFont(13))
	titleLabel.Send(sel("setLineBreakMode:"), int64(4))
	titleLabel.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsContentX + settingsRowInset, Y: y + 7},
		Size:   CGSize{Width: detailWidth, Height: 18},
	})
	content.Send(sel("addSubview:"), titleLabel)
	path := map[string]string{
		"accessibilityPermissionPressed:":  "permissions.accessibility",
		"fullDiskAccessPermissionPressed:": "permissions.fullDiskAccess",
	}[action]
	v.addSearchEntry(content, title, path, detail, i18n.T(i18n.SettingsPermissionsKeywords),
		CGRect{
			Origin: CGPoint{X: settingsContentX + 1, Y: y},
			Size:   CGSize{Width: settingsContentWidth - 2, Height: settingsPermissionRowHeight},
		})
	detailLabel := objc.ID(classNSTextField).Send(sel("wrappingLabelWithString:"), nsString(detail))
	detailLabel.Send(sel("setFont:"), systemFont(11))
	detailLabel.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	detailLabel.Send(sel("setMaximumNumberOfLines:"), int64(2))
	detailLabel.Send(sel("setPreferredMaxLayoutWidth:"), detailWidth)
	detailLabel.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsContentX + settingsRowInset, Y: y + 29},
		Size:   CGSize{Width: detailWidth, Height: 28},
	})
	content.Send(sel("addSubview:"), detailLabel)

	status := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(i18n.T(i18n.SettingsChecking)))
	status.Send(sel("setFont:"), systemFont(11))
	status.Send(sel("setAlignment:"), int64(2))
	status.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: statusX, Y: y + 22},
		Size:   CGSize{Width: statusWidth, Height: 18},
	})
	content.Send(sel("addSubview:"), status)

	button := objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(i18n.T(i18n.SettingsOpenSystemSettings)), settingsTarget, sel(action))
	button.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: buttonX, Y: y + 16},
		Size:   CGSize{Width: buttonWidth, Height: 28},
	})
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
	text, color := i18n.T(i18n.PermissionNotGranted), "systemRedColor"
	if granted {
		text, color = i18n.T(i18n.PermissionGranted), "systemGreenColor"
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
