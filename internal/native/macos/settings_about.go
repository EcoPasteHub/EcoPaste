//go:build darwin

package macos

import (
	"strings"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
	"github.com/nsgo-dev/nsgo"
	kubai "github.com/nsgo-dev/nsgo-icons-kubai"
	"github.com/nsgo-dev/nsgo/core"
)

const (
	aboutIconSize      = 110.0
	aboutRowIconSize   = 16.0
	aboutRowLabelInset = settingsRowInset + aboutRowIconSize + 6
	aboutInfoRowHeight = 50.0
	aboutInfoRows      = 4.0
	aboutCardRadius    = 16.0
	aboutPillHeight    = 24.0
	aboutButtonWidth   = 188.0
	aboutButtonHeight  = 34.0
)

func (v *SettingsView) buildAboutPage() {
	document := v.newPage("about")
	top := 20.0
	top = v.addAboutHero(document, top)
	top = v.addAboutInfoCard(document, top+24)
	v.addAboutActions(document, top+28)
	setFrame(document, 0, 0, settingsWidth, settingsHeight)
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
	top += aboutIconSize + 8

	name := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString("EcoPaste"))
	name.Send(sel("setFont:"), objc.ID(classNSFont).Send(
		sel("systemFontOfSize:weight:"), 24.0, 0.3))
	name.Send(sel("sizeToFit"))
	nameFrame := objc.Send[CGRect](name, sel("frame"))
	nameFrame.Origin = CGPoint{
		X: settingsContentX + (settingsContentWidth-nameFrame.Size.Width)/2,
		Y: top,
	}
	name.Send(sel("setFrame:"), nameFrame)
	document.Send(sel("addSubview:"), name)
	top += nameFrame.Size.Height + 8

	versionText := AppVersion()
	if versionText == "" {
		versionText = i18n.T(i18n.SettingsDevBuild)
	}
	version := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(versionText))
	version.Send(sel("setFont:"), objc.ID(classNSFont).Send(
		sel("systemFontOfSize:weight:"), 12.0, 0.2))
	version.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	version.Send(sel("sizeToFit"))
	versionFrame := objc.Send[CGRect](version, sel("frame"))
	pillWidth := versionFrame.Size.Width + 24
	if pillWidth < 44 {
		pillWidth = 44
	}
	pillX := settingsContentX + (settingsContentWidth-pillWidth)/2
	pill := objc.ID(classNSBox).Send(sel("new"))
	pill.Send(sel("setBoxType:"), int64(4))
	pill.Send(sel("setTitlePosition:"), int64(0))
	pill.Send(sel("setBorderWidth:"), 0.0)
	pill.Send(sel("setFillColor:"), settingsCardFillColor())
	pill.Send(sel("setCornerRadius:"), aboutPillHeight/2)
	pill.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: pillX, Y: top},
		Size:   CGSize{Width: pillWidth, Height: aboutPillHeight},
	})
	document.Send(sel("addSubview:"), pill)
	pill.Send(sel("release"))

	versionFrame.Origin = CGPoint{
		X: pillX + (pillWidth-versionFrame.Size.Width)/2,
		Y: top + (aboutPillHeight-versionFrame.Size.Height)/2,
	}
	version.Send(sel("setFrame:"), versionFrame)
	document.Send(sel("addSubview:"), version)
	if AppVersion() == "" {
		v.bindString(version, i18n.SettingsDevBuild)
	}
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
	v.addAboutLinkRow(document, top, kubai.Link, core.RGB(0x5393FF), i18n.SettingsAboutSource,
		"GitHub ↗", "openSourcePressed:", i18n.SettingsAboutSourceA11y)
	v.addAboutLinkRow(document, top+aboutInfoRowHeight, kubai.MessageSquare, core.RGB(0xFF9F0A),
		i18n.SettingsAboutFeedback, "GitHub Issues ↗",
		"openIssuesPressed:", i18n.SettingsAboutFeedbackA11y)
	v.addAboutValueRow(document, top+2*aboutInfoRowHeight, kubai.CircleCheck, core.RGB(0x30D158),
		i18n.SettingsAboutLicense, "Apache-2.0")
	v.addAboutValueRow(document, top+3*aboutInfoRowHeight, kubai.Desktop, core.RGB(0xBF5AF2),
		i18n.SettingsAboutSystem, aboutSystemVersion())
	return top + aboutInfoRows*aboutInfoRowHeight
}

func aboutSystemVersion() string {
	version, _, _ := strings.Cut(diagnosticSystemVersion(), " (Build ")
	return version
}

func (v *SettingsView) addAboutValueRow(document objc.ID, rowTop float64, symbol func() *kubai.Symbol, color core.Color, title i18n.Key, value string) {
	v.addAboutRowIcon(document, rowTop, symbol, color)
	label := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(i18n.T(title)))
	label.Send(sel("setFont:"), systemFont(13))
	label.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsContentX + aboutRowLabelInset, Y: rowTop + 17},
		Size:   CGSize{Width: settingsContentWidth/2 - settingsRowInset, Height: 16},
	})
	document.Send(sel("addSubview:"), label)
	v.bindString(label, title)

	valueLabel := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(value))
	valueLabel.Send(sel("setFont:"), systemFont(13))
	valueLabel.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	valueLabel.Send(sel("sizeToFit"))
	valueFrame := objc.Send[CGRect](valueLabel, sel("frame"))
	valueFrame.Origin = aboutRowValueOrigin(valueFrame.Size, rowTop)
	valueLabel.Send(sel("setFrame:"), valueFrame)
	document.Send(sel("addSubview:"), valueLabel)
}

func (v *SettingsView) addAboutLinkRow(document objc.ID, rowTop float64, symbol func() *kubai.Symbol, color core.Color, title i18n.Key, value, action string, help i18n.Key) {
	v.addAboutRowIcon(document, rowTop, symbol, color)
	label := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(i18n.T(title)))
	label.Send(sel("setFont:"), systemFont(13))
	label.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsContentX + aboutRowLabelInset, Y: rowTop + 17},
		Size:   CGSize{Width: settingsContentWidth/2 - settingsRowInset, Height: 16},
	})
	document.Send(sel("addSubview:"), label)
	v.bindString(label, title)

	button := objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(value), settingsTarget, sel(action))
	button.Send(sel("setBordered:"), false)
	button.Send(sel("setFont:"), systemFont(13))
	button.Send(sel("setContentTintColor:"), objc.ID(classNSColor).Send(sel("controlAccentColor")))
	button.Send(sel("sizeToFit"))
	buttonFrame := objc.Send[CGRect](button, sel("frame"))
	buttonFrame.Origin = aboutRowValueOrigin(buttonFrame.Size, rowTop)
	button.Send(sel("setFrame:"), buttonFrame)
	button.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(title)))
	button.Send(sel("setAccessibilityHelp:"), nsString(i18n.T(help)))
	document.Send(sel("addSubview:"), button)
	settingsViews[button] = v
	v.bindText(func() {
		button.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(title)))
		button.Send(sel("setAccessibilityHelp:"), nsString(i18n.T(help)))
	})
}

func aboutRowValueOrigin(size CGSize, rowTop float64) CGPoint {
	return CGPoint{
		X: settingsContentX + settingsContentWidth - settingsRowInset - size.Width,
		Y: rowTop + (aboutInfoRowHeight-size.Height)/2,
	}
}

func (v *SettingsView) addAboutRowIcon(document objc.ID, rowTop float64, symbol func() *kubai.Symbol, color core.Color) {
	icon := nsgo.NewView(func(*core.Scope) core.Node {
		return core.View{
			Style: core.Style{
				Width:          core.Pt(aboutRowIconSize),
				Height:         core.Pt(aboutRowIconSize),
				AlignItems:     core.AlignCenter,
				JustifyContent: core.JustifyCenter,
			},
			Children: []core.Node{
				kubai.Icon{
					Symbol: symbol(),
					Size:   aboutRowIconSize,
					Color:  color,
				},
			},
		}
	})
	view := objc.ID(icon.Handle())
	view.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsContentX + settingsRowInset, Y: rowTop + 17},
		Size:   CGSize{Width: aboutRowIconSize, Height: aboutRowIconSize},
	})
	document.Send(sel("addSubview:"), view)
	v.aboutIcons = append(v.aboutIcons, icon)
}

func (v *SettingsView) addAboutActions(document objc.ID, top float64) {
	const gap = 48.0
	v.addPageButton(document, settingsContentX, top, aboutButtonWidth, aboutButtonHeight,
		i18n.SettingsReset, "resetSettingsPressed:", i18n.SettingsResetBody)
	v.addPageButton(document, settingsContentX+aboutButtonWidth+gap, top,
		aboutButtonWidth, aboutButtonHeight,
		i18n.SettingsUpdateCheck, "updateCheckPressed:", i18n.SettingsUpdateIntro)
}
