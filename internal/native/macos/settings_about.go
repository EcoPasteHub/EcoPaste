//go:build darwin

package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

const (
	aboutIconSize        = 84.0
	aboutInfoRowHeight   = 42.0
	aboutActionRowHeight = 58.0
	aboutVersionPillW    = 118.0
	aboutVersionPillH    = 22.0
)

func (v *SettingsView) buildAboutPage() {
	document := v.newPage("about")
	top := 20.0
	top = v.addAboutHero(document, top)
	top = v.addAboutInfo(document, top+22)
	top = v.addAboutMaintenance(document, top+22)
	v.addAboutUpdateHint(document, top+10)
	v.finishPage(document, top+28)
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
	top += aboutIconSize + 6

	name := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString("EcoPaste"))
	name.Send(sel("setFont:"), objc.ID(classNSFont).Send(
		sel("systemFontOfSize:weight:"), 22.0, 0.3))
	name.Send(sel("setAlignment:"), int64(2))
	name.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsContentX, Y: top},
		Size:   CGSize{Width: settingsContentWidth, Height: 28},
	})
	document.Send(sel("addSubview:"), name)
	top += 32

	tagline := objc.ID(classNSTextField).Send(sel("labelWithString:"),
		nsString(i18n.T(i18n.SettingsAboutTagline)))
	tagline.Send(sel("setFont:"), systemFont(12))
	tagline.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	tagline.Send(sel("setAlignment:"), int64(2))
	tagline.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsContentX + 12, Y: top},
		Size:   CGSize{Width: settingsContentWidth - 24, Height: 18},
	})
	document.Send(sel("addSubview:"), tagline)
	top += 24

	versionPillX := settingsContentX + (settingsContentWidth-aboutVersionPillW)/2
	pill := objc.ID(classNSBox).Send(sel("new"))
	pill.Send(sel("setBoxType:"), int64(4))
	pill.Send(sel("setTitlePosition:"), int64(0))
	pill.Send(sel("setBorderWidth:"), 0.0)
	pill.Send(sel("setFillColor:"), settingsCardFillColor())
	pill.Send(sel("setCornerRadius:"), aboutVersionPillH/2)
	pill.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: versionPillX, Y: top},
		Size:   CGSize{Width: aboutVersionPillW, Height: aboutVersionPillH},
	})
	document.Send(sel("addSubview:"), pill)
	pill.Send(sel("release"))

	version := objc.ID(classNSTextField).Send(sel("labelWithString:"),
		nsString(i18n.Tf(i18n.SettingsUpdateVersion, AppVersion())))
	version.Send(sel("setFont:"), objc.ID(classNSFont).Send(
		sel("systemFontOfSize:weight:"), 11.0, 0.2))
	version.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	version.Send(sel("setAlignment:"), int64(2))
	version.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: versionPillX, Y: top + 3},
		Size:   CGSize{Width: aboutVersionPillW, Height: 16},
	})
	document.Send(sel("addSubview:"), version)
	return top + aboutVersionPillH
}

func (v *SettingsView) addAboutInfo(document objc.ID, top float64) float64 {
	v.addSectionLabel(document, top, i18n.T(i18n.SettingsAboutInfo))
	cardTop := top + 24
	addSettingsListCard(document, CGRect{
		Origin: CGPoint{X: settingsContentX, Y: cardTop},
		Size:   CGSize{Width: settingsContentWidth, Height: 2 * aboutInfoRowHeight},
	})
	v.addCardSeparatorOffset(document, cardTop, aboutInfoRowHeight)
	v.addAboutInfoRow(document, cardTop, i18n.T(i18n.SettingsAboutSystem), diagnosticSystemVersion())
	v.addAboutSourceRow(document, cardTop+aboutInfoRowHeight)
	return cardTop + 2*aboutInfoRowHeight
}

func (v *SettingsView) addAboutInfoRow(document objc.ID, rowTop float64, title, value string) {
	label := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(title))
	label.Send(sel("setFont:"), systemFont(13))
	label.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsContentX + settingsRowInset, Y: rowTop + 12},
		Size:   CGSize{Width: settingsContentWidth/2 - settingsRowInset, Height: 18},
	})
	document.Send(sel("addSubview:"), label)

	valueLabel := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(value))
	valueLabel.Send(sel("setFont:"), systemFont(12))
	valueLabel.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	valueLabel.Send(sel("setAlignment:"), int64(1))
	valueLabel.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{
			X: settingsContentX + settingsContentWidth - settingsRowInset - 220,
			Y: rowTop + 11,
		},
		Size: CGSize{Width: 220, Height: 18},
	})
	document.Send(sel("addSubview:"), valueLabel)
}

func (v *SettingsView) addAboutSourceRow(document objc.ID, rowTop float64) {
	v.addAboutInfoRow(document, rowTop, i18n.T(i18n.SettingsAboutSource), "")

	button := objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString("GitHub ↗"), settingsTarget, sel("openSourcePressed:"))
	button.Send(sel("setBordered:"), false)
	button.Send(sel("setFont:"), systemFont(13))
	button.Send(sel("setContentTintColor:"), objc.ID(classNSColor).Send(sel("controlAccentColor")))
	button.Send(sel("setAlignment:"), int64(1))
	button.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{
			X: settingsContentX + settingsContentWidth - settingsRowInset - 220,
			Y: rowTop + 8,
		},
		Size: CGSize{Width: 220, Height: 26},
	})
	button.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.SettingsAboutSource)))
	button.Send(sel("setAccessibilityHelp:"), nsString(i18n.T(i18n.SettingsAboutSourceA11y)))
	document.Send(sel("addSubview:"), button)
	settingsViews[button] = v
}

func (v *SettingsView) addAboutMaintenance(document objc.ID, top float64) float64 {
	v.addSectionLabel(document, top, i18n.T(i18n.SettingsAboutMaintenance))
	cardTop := top + 24
	addSettingsListCard(document, CGRect{
		Origin: CGPoint{X: settingsContentX, Y: cardTop},
		Size:   CGSize{Width: settingsContentWidth, Height: 2 * aboutActionRowHeight},
	})
	v.addCardSeparatorOffset(document, cardTop, aboutActionRowHeight)
	v.addAboutActionRow(document, cardTop,
		i18n.T(i18n.SettingsAboutUpdateTitle), i18n.T(i18n.SettingsAboutUpdateDetail),
		i18n.SettingsUpdateCheck, "updateCheckPressed:", i18n.SettingsUpdateIntro)
	v.addAboutActionRow(document, cardTop+aboutActionRowHeight,
		i18n.T(i18n.SettingsAboutResetTitle), i18n.T(i18n.SettingsAboutResetDetail),
		i18n.SettingsReset, "resetSettingsPressed:", i18n.SettingsResetBody)
	return cardTop + 2*aboutActionRowHeight
}

func (v *SettingsView) addAboutActionRow(document objc.ID, rowTop float64, title, detail string, buttonTitle i18n.Key, action string, help i18n.Key) {
	const buttonWidth = 144.0
	label := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(title))
	label.Send(sel("setFont:"), systemFont(13))
	label.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsContentX + settingsRowInset, Y: rowTop + 9},
		Size:   CGSize{Width: settingsContentWidth - 2*settingsRowInset - buttonWidth - 16, Height: 18},
	})
	document.Send(sel("addSubview:"), label)

	detailLabel := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(detail))
	detailLabel.Send(sel("setFont:"), systemFont(11))
	detailLabel.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	detailLabel.Send(sel("setLineBreakMode:"), int64(4))
	detailLabel.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsContentX + settingsRowInset, Y: rowTop + 29},
		Size: CGSize{
			Width:  settingsContentWidth - 2*settingsRowInset - buttonWidth - 16,
			Height: 16,
		},
	})
	document.Send(sel("addSubview:"), detailLabel)

	button := v.addPageButton(document,
		settingsContentX+settingsContentWidth-settingsRowInset-buttonWidth,
		rowTop+(aboutActionRowHeight-30)/2, buttonWidth, 30,
		buttonTitle, action, help)
	button.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(buttonTitle)))
	if action == "resetSettingsPressed:" {
		v.reset = button
	}
}

func (v *SettingsView) addAboutUpdateHint(document objc.ID, top float64) {
	hint := objc.ID(classNSTextField).Send(sel("labelWithString:"),
		nsString(i18n.T(i18n.SettingsUpdateManaged)))
	hint.Send(sel("setFont:"), systemFont(11))
	hint.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("tertiaryLabelColor")))
	hint.Send(sel("setAlignment:"), int64(2))
	hint.Send(sel("setLineBreakMode:"), int64(4))
	hint.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsContentX + settingsRowInset, Y: top},
		Size:   CGSize{Width: settingsContentWidth - 2*settingsRowInset, Height: 16},
	})
	document.Send(sel("addSubview:"), hint)
}
