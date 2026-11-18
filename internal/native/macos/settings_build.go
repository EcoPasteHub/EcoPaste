//go:build darwin

package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

const settingsPermissionRowHeight = 44.0

func (v *SettingsView) addPermission(content objc.ID, y float64, titleKey, detailKey i18n.Key, action string) objc.ID {
	title := i18n.T(titleKey)
	const (
		buttonWidth = 132.0
		statusWidth = 60.0
		controlGap  = 12.0
	)
	buttonX := settingsContentX + settingsContentWidth - settingsRowInset - buttonWidth
	statusX := buttonX - controlGap - statusWidth
	titleWidth := statusX - (settingsContentX + settingsRowInset) - controlGap
	centerY := y + settingsPermissionRowHeight/2

	titleLabel := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(title))
	titleLabel.Send(sel("setFont:"), systemFont(13))
	titleLabel.Send(sel("setLineBreakMode:"), int64(4))
	titleLabel.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsContentX + settingsRowInset, Y: centerY - 9},
		Size:   CGSize{Width: titleWidth, Height: 18},
	})
	content.Send(sel("addSubview:"), titleLabel)
	v.bindString(titleLabel, titleKey)
	v.addInfoAfterText(content, settingsContentX+settingsRowInset, centerY,
		titleKey, systemFont(13), detailKey)

	status := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(i18n.T(i18n.SettingsChecking)))
	status.Send(sel("setFont:"), systemFont(11))
	status.Send(sel("setAlignment:"), int64(2))
	status.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: statusX, Y: centerY - 9},
		Size:   CGSize{Width: statusWidth, Height: 18},
	})
	content.Send(sel("addSubview:"), status)

	button := objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(i18n.T(i18n.SettingsOpenSystemSettings)), settingsTarget, sel(action))
	button.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: buttonX, Y: y + (settingsPermissionRowHeight-28)/2},
		Size:   CGSize{Width: buttonWidth, Height: 28},
	})
	button.Send(sel("setAccessibilityLabel:"), nsString(i18n.Tf(i18n.SettingsOpenSystemSettingsA11y, title)))
	content.Send(sel("addSubview:"), button)
	v.bindText(func() {
		title := i18n.T(titleKey)
		button.Send(sel("setTitle:"), nsString(i18n.T(i18n.SettingsOpenSystemSettings)))
		button.Send(sel("setAccessibilityLabel:"), nsString(i18n.Tf(i18n.SettingsOpenSystemSettingsA11y, title)))
	})
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
