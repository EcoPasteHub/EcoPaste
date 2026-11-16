//go:build darwin

package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

const (
	aboutIconSize      = 108.0
	aboutInfoRowHeight = 52.0
	aboutInfoRows      = 4.0
	aboutCardRadius    = 14.0
	aboutPillWidth     = 104.0
	aboutPillHeight    = 24.0
	aboutButtonWidth   = 188.0
	aboutButtonHeight  = 34.0
)

func (v *SettingsView) buildAboutPage() {
	document := v.newPage("about")
	top := 18.0
	top = v.addAboutHero(document, top)
	top = v.addAboutInfoCard(document, top+22)
	v.addAboutActions(document, top+30)
	v.finishPage(document, settingsHeight-settingsSearchHeaderH)
}

func (v *SettingsView) addAboutHero(document objc.ID, top float64) float64 {
	iconImage := objc.ID(class("NSApplication")).
		Send(sel("sharedApplication")).
		Send(sel("applicationIconImage"))
	icon := objc.ID(class("NSImageView")).Send(sel("new"))
	icon.Send(sel("setImage:"), iconImage)
	icon.Send(sel("setImageScaling:"), int64(3))
	icon.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{
			X: settingsContentX + settingsContentWidth/2 - aboutIconSize/2,
			Y: top,
		},
		Size: CGSize{Width: aboutIconSize, Height: aboutIconSize},
	})
	document.Send(sel("addSubview:"), icon)
	icon.Send(sel("release"))
	top += aboutIconSize + 10

	name := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString("EcoPaste"))
	name.Send(sel("setFont:"), objc.ID(classNSFont).Send(
		sel("systemFontOfSize:weight:"), 30.0, 0.4))
	name.Send(sel("setAlignment:"), int64(2))
	name.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsContentX, Y: top},
		Size:   CGSize{Width: settingsContentWidth, Height: 34},
	})
	document.Send(sel("addSubview:"), name)
	top += 42

	versionText := AppVersion()
	if versionText == "" {
		versionText = i18n.T(i18n.SettingsDevBuild)
	}
	pillX := settingsContentX + (settingsContentWidth-aboutPillWidth)/2
	pill := objc.ID(classNSBox).Send(sel("new"))
	pill.Send(sel("setBoxType:"), int64(4))
	pill.Send(sel("setTitlePosition:"), int64(0))
	pill.Send(sel("setBorderWidth:"), 0.0)
	pill.Send(sel("setFillColor:"), settingsCardFillColor())
	pill.Send(sel("setCornerRadius:"), aboutPillHeight/2)
	pill.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: pillX, Y: top},
		Size:   CGSize{Width: aboutPillWidth, Height: aboutPillHeight},
	})
	document.Send(sel("addSubview:"), pill)
	pill.Send(sel("release"))

	version := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(versionText))
	version.Send(sel("setFont:"), objc.ID(classNSFont).Send(
		sel("systemFontOfSize:weight:"), 12.0, 0.2))
	version.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	version.Send(sel("setAlignment:"), int64(2))
	version.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: pillX, Y: top + 4},
		Size:   CGSize{Width: aboutPillWidth, Height: 16},
	})
	document.Send(sel("addSubview:"), version)
	return top + aboutPillHeight
}

func (v *SettingsView) addAboutInfoCard(document objc.ID, top float64) float64 {
	card := objc.ID(classNSBox).Send(sel("new"))
	card.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsContentX, Y: top},
		Size: CGSize{
			Width:  settingsContentWidth,
			Height: aboutInfoRows * aboutInfoRowHeight,
		},
	})
	card.Send(sel("setBoxType:"), int64(4))
	card.Send(sel("setTitlePosition:"), int64(0))
	card.Send(sel("setBorderWidth:"), 0.0)
	card.Send(sel("setFillColor:"), settingsCardFillColor())
	card.Send(sel("setCornerRadius:"), aboutCardRadius)
	document.Send(sel("addSubview:"), card)
	card.Send(sel("release"))

	for row := 1; row < int(aboutInfoRows); row++ {
		v.addCardSeparatorOffset(document, top, float64(row)*aboutInfoRowHeight)
	}
	v.addAboutLinkRow(document, top, i18n.T(i18n.SettingsAboutSource),
		"GitHub ↗", "openSourcePressed:", i18n.SettingsAboutSourceA11y)
	v.addAboutLinkRow(document, top+aboutInfoRowHeight, i18n.T(i18n.SettingsAboutFeedback),
		"GitHub Issues ↗", "openIssuesPressed:", i18n.SettingsAboutFeedbackA11y)
	v.addAboutValueRow(document, top+2*aboutInfoRowHeight,
		i18n.T(i18n.SettingsAboutLicense), "Apache-2.0")
	v.addAboutValueRow(document, top+3*aboutInfoRowHeight,
		i18n.T(i18n.SettingsAboutSystem), diagnosticSystemVersion())
	return top + aboutInfoRows*aboutInfoRowHeight
}

func (v *SettingsView) addAboutValueRow(document objc.ID, rowTop float64, title, value string) {
	label := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(title))
	label.Send(sel("setFont:"), systemFont(13))
	label.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsContentX + settingsRowInset + 2, Y: rowTop + 17},
		Size:   CGSize{Width: settingsContentWidth/2 - settingsRowInset, Height: 18},
	})
	document.Send(sel("addSubview:"), label)

	valueLabel := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(value))
	valueLabel.Send(sel("setFont:"), systemFont(13))
	valueLabel.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	valueLabel.Send(sel("setAlignment:"), int64(1))
	valueLabel.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{
			X: settingsContentX + settingsContentWidth - settingsRowInset - 220,
			Y: rowTop + 17,
		},
		Size: CGSize{Width: 220, Height: 18},
	})
	document.Send(sel("addSubview:"), valueLabel)
}

func (v *SettingsView) addAboutLinkRow(document objc.ID, rowTop float64, title, value, action string, help i18n.Key) {
	label := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(title))
	label.Send(sel("setFont:"), systemFont(13))
	label.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsContentX + settingsRowInset + 2, Y: rowTop + 17},
		Size:   CGSize{Width: settingsContentWidth/2 - settingsRowInset, Height: 18},
	})
	document.Send(sel("addSubview:"), label)

	button := objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(value), settingsTarget, sel(action))
	button.Send(sel("setBordered:"), false)
	button.Send(sel("setFont:"), systemFont(13))
	button.Send(sel("setContentTintColor:"), objc.ID(classNSColor).Send(sel("controlAccentColor")))
	button.Send(sel("setAlignment:"), int64(1))
	button.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{
			X: settingsContentX + settingsContentWidth - settingsRowInset - 220,
			Y: rowTop + 13,
		},
		Size: CGSize{Width: 220, Height: 26},
	})
	button.Send(sel("setAccessibilityLabel:"), nsString(title))
	button.Send(sel("setAccessibilityHelp:"), nsString(i18n.T(help)))
	document.Send(sel("addSubview:"), button)
	settingsViews[button] = v
}

func (v *SettingsView) addAboutActions(document objc.ID, top float64) {
	const gap = 48.0
	v.addPageButton(document, settingsContentX, top, aboutButtonWidth, aboutButtonHeight,
		i18n.SettingsReset, "resetSettingsPressed:", i18n.SettingsResetBody)
	v.addPageButton(document, settingsContentX+aboutButtonWidth+gap, top,
		aboutButtonWidth, aboutButtonHeight,
		i18n.SettingsUpdateCheck, "updateCheckPressed:", i18n.SettingsUpdateIntro)
}
