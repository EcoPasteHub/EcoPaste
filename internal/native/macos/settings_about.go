//go:build darwin

package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

func (v *SettingsView) buildAboutPage() {
	document := v.newPage("about")
	top := 26.0

	iconImage := objc.Send[objc.ID](
		objc.ID(class("NSApplication")).Send(sel("sharedApplication")), sel("applicationIconImage"))
	icon := objc.ID(class("NSImageView")).Send(sel("new"))
	icon.Send(sel("setImage:"), iconImage)
	icon.Send(sel("setImageScaling:"), int64(3))
	icon.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsContentX + settingsContentWidth/2 - 34, Y: top},
		Size:   CGSize{Width: 68, Height: 68},
	})
	document.Send(sel("addSubview:"), icon)
	icon.Send(sel("release"))
	top += 84

	name := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString("EcoPaste"))
	name.Send(sel("setFont:"), objc.ID(classNSFont).Send(sel("systemFontOfSize:weight:"), 21.0, 0.3))
	name.Send(sel("setAlignment:"), int64(2))
	name.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: settingsContentX, Y: top},
		Size: CGSize{Width: settingsContentWidth, Height: 26}})
	document.Send(sel("addSubview:"), name)
	top += 30

	version := objc.ID(classNSTextField).Send(sel("labelWithString:"),
		nsString(i18n.Tf(i18n.SettingsUpdateVersion, AppVersion())))
	version.Send(sel("setFont:"), systemFont(12))
	version.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	version.Send(sel("setAlignment:"), int64(2))
	version.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: settingsContentX, Y: top},
		Size: CGSize{Width: settingsContentWidth, Height: 18}})
	document.Send(sel("addSubview:"), version)
	top += 46

	top = v.addPageParagraph(document, top, 34, i18n.T(i18n.SettingsUpdateIntro))

	_, cardTop := v.addSectionCard(document, top, i18n.T(i18n.SettingsTabUpdate), 1)
	rowTop := settingsRowTop(cardTop, 0)
	check := v.addPageButton(document, settingsContentX+settingsContentWidth-settingsRowInset-150,
		rowTop+8, 150, 28, i18n.SettingsUpdateCheck, "updateCheckPressed:", i18n.SettingsUpdateIntro)
	check.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.SettingsUpdateCheck)))
	top = cardTop + settingsRowHeight
	top = v.addPageHint(document, top+8, 30, i18n.T(i18n.SettingsUpdateManaged)) + 20

	_, cardTop = v.addSectionCard(document, top, i18n.T(i18n.SettingsResetSection), 1)
	rowTop = settingsRowTop(cardTop, 0)
	reset := v.addPageButton(document, settingsContentX+settingsContentWidth-settingsRowInset-150,
		rowTop+8, 150, 28, i18n.SettingsReset, "resetSettingsPressed:", i18n.SettingsResetBody)
	v.reset = reset
	top = cardTop + settingsRowHeight + 24
	v.finishPage(document, top)
}
