//go:build darwin

package macos

import (
	"math"
	"sync"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

const (
	settingsInfoButtonSize  = 15.0
	settingsInfoButtonGap   = 6.0
	settingsInfoTextSize    = 12.0
	settingsInfoPadding     = 12.0
	settingsInfoMinWidth    = 120.0
	settingsInfoMaxWidth    = 260.0
	settingsInfoMinHeight   = 40.0
	settingsInfoBubbleSlack = 26.0
	settingsInfoPopoverEdge = 3
)

// settingsInfo is one "?" hint button and the popover it reveals on hover.
// The popover is laid out once and only re-measured when the language text
// changes, because moving the content view on every show pushes it out of
// the inset NSPopover applies to its content.
type settingsInfo struct {
	button     objc.ID
	popover    objc.ID
	controller objc.ID
	view       objc.ID
	label      objc.ID
	key        i18n.Key
	text       string
}

var (
	registerSettingsInfoOnce sync.Once
	settingsInfoButtonClass  objc.Class
	settingsInfos            = map[objc.ID]*settingsInfo{}
)

func registerSettingsInfoTargets() {
	var err error
	settingsInfoButtonClass, err = objc.RegisterClass("EcoPasteSettingsInfoButton", class("NSButton"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("updateTrackingAreas"), Fn: func(self objc.ID, _ objc.SEL) {
				sendSuper[objc.ID](self, settingsInfoButtonClass, sel("updateTrackingAreas"))
				area := objc.ID(class("NSTrackingArea")).Send(sel("alloc")).
					Send(sel("initWithRect:options:owner:userInfo:"), CGRect{}, uint64(1|128|512), self, objc.ID(0))
				self.Send(sel("addTrackingArea:"), area)
				area.Send(sel("release"))
			}},
			{Cmd: sel("mouseEntered:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if info := settingsInfos[self]; info != nil {
					self.Send(sel("setContentTintColor:"), objc.ID(classNSColor).Send(sel("controlAccentColor")))
					info.show()
				}
			}},
			{Cmd: sel("mouseExited:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if info := settingsInfos[self]; info != nil {
					self.Send(sel("setContentTintColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
					info.hide()
				}
			}},
		})
	if err != nil {
		panic(err)
	}
}

// newSettingsInfoButton builds a borderless "?" button that owns its popover.
func (v *SettingsView) newSettingsInfoButton(key i18n.Key) objc.ID {
	registerSettingsInfoOnce.Do(registerSettingsInfoTargets)
	button := objc.ID(settingsInfoButtonClass).Send(sel("alloc")).Send(sel("initWithFrame:"), CGRect{})
	button.Send(sel("setBordered:"), false)
	button.Send(sel("setImagePosition:"), int64(1))
	button.Send(sel("setImageScaling:"), int64(2))
	button.Send(sel("setFocusRingType:"), int64(1))
	button.Send(sel("setRefusesFirstResponder:"), true)
	button.Send(sel("setImage:"), settingsInfoSymbol())
	button.Send(sel("setContentTintColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))

	info := &settingsInfo{button: button, key: key}
	info.buildPopover()
	settingsInfos[button] = info
	v.infoButtons = append(v.infoButtons, button)
	v.bindText(func() {
		button.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.SettingsInfoA11y)))
		button.Send(sel("setAccessibilityHelp:"), nsString(i18n.T(key)))
		info.hide()
	})
	button.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.SettingsInfoA11y)))
	button.Send(sel("setAccessibilityHelp:"), nsString(i18n.T(key)))
	return button
}

// addInfoAfterText places a "?" immediately after a title label and keeps it
// aligned whenever the title's translated width changes.
func (v *SettingsView) addInfoAfterText(content objc.ID, x, centerY float64, title i18n.Key, font objc.ID, key i18n.Key) objc.ID {
	button := v.newSettingsInfoButton(key)
	content.Send(sel("addSubview:"), button)
	place := func() {
		width := settingsTextWidth(i18n.T(title), font)
		setFrame(button, x+width+settingsInfoButtonGap, centerY-settingsInfoButtonSize/2,
			settingsInfoButtonSize, settingsInfoButtonSize)
	}
	place()
	v.bindText(place)
	return button
}

func settingsInfoSymbol() objc.ID {
	image := systemSymbol("questionmark.circle")
	image.Send(sel("setSize:"), CGSize{Width: 13, Height: 13})
	return image
}

func settingsTextWidth(text string, font objc.ID) float64 {
	attributes := objc.ID(class("NSDictionary")).Send(sel("dictionaryWithObject:forKey:"),
		font, nsString("NSFont"))
	return objc.Send[CGSize](nsString(text), sel("sizeWithAttributes:"), attributes).Width
}

func (i *settingsInfo) buildPopover() {
	i.view = objc.ID(class("NSView")).Send(sel("new"))
	i.label = objc.ID(classNSTextField).Send(sel("wrappingLabelWithString:"), nsString(""))
	i.label.Send(sel("setFont:"), systemFont(settingsInfoTextSize))
	i.label.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("labelColor")))
	i.label.Send(sel("setLineBreakMode:"), int64(0))
	i.label.Send(sel("setMaximumNumberOfLines:"), int64(0))
	i.view.Send(sel("addSubview:"), i.label)

	i.controller = objc.ID(class("NSViewController")).Send(sel("new"))
	i.controller.Send(sel("setView:"), i.view)
	i.popover = objc.ID(class("NSPopover")).Send(sel("new"))
	// Transient lets AppKit dismiss the hint on outside interaction; hover
	// exit also closes it explicitly.
	i.popover.Send(sel("setBehavior:"), int64(1))
	i.popover.Send(sel("setAnimates:"), true)
	i.popover.Send(sel("setContentViewController:"), i.controller)
	i.layout()
}

// layout measures the hint text and sizes the popover once. The content
// view must not be moved again for the same text on later shows.
func (i *settingsInfo) layout() {
	text := i18n.T(i.key)
	width, height := settingsInfoPopoverSize(text)
	contentWidth := width - 2*settingsInfoPadding
	i.label.Send(sel("setStringValue:"), nsString(text))
	i.label.Send(sel("setPreferredMaxLayoutWidth:"), contentWidth)
	setFrame(i.view, 0, 0, width, height)
	setFrame(i.label, settingsInfoPadding, settingsInfoPadding,
		contentWidth, height-2*settingsInfoPadding)
	i.popover.Send(sel("setContentSize:"), CGSize{Width: width, Height: height})
	i.text = text
}

func (i *settingsInfo) show() {
	if i.popover == 0 {
		return
	}
	if text := i18n.T(i.key); text != i.text {
		i.hide()
		i.layout()
	}
	if !objc.Send[bool](i.popover, sel("isShown")) {
		i.popover.Send(sel("showRelativeToRect:ofView:preferredEdge:"),
			bounds(i.button), i.button, uint64(settingsInfoPopoverEdge))
	}
}

func (i *settingsInfo) hide() {
	if i.popover != 0 && objc.Send[bool](i.popover, sel("isShown")) {
		i.popover.Send(sel("close"))
	}
}

func (i *settingsInfo) close() {
	i.hide()
	if i.popover != 0 {
		i.popover.Send(sel("release"))
		i.popover = 0
	}
	if i.controller != 0 {
		i.controller.Send(sel("release"))
		i.controller = 0
	}
	if i.view != 0 {
		i.view.Send(sel("release"))
		i.view = 0
	}
	// label is autoreleased and owned by view, so it must not be released here.
	i.label = 0
}

// settingsInfoPopoverSize sizes the popover to its copy, wrapping long text at
// settingsInfoMaxWidth so the hint stays compact.
func settingsInfoPopoverSize(text string) (float64, float64) {
	attributes := objc.ID(class("NSDictionary")).Send(sel("dictionaryWithObject:forKey:"),
		systemFont(settingsInfoTextSize), nsString("NSFont"))
	single := objc.Send[CGSize](nsString(text), sel("sizeWithAttributes:"), attributes)
	// NSPopover insets its content view, so leave extra room on top of the
	// measured text to avoid clipping the last glyphs of a single line.
	width := single.Width + 2*settingsInfoPadding + settingsInfoBubbleSlack
	if width > settingsInfoMaxWidth {
		width = settingsInfoMaxWidth
	}
	if width < settingsInfoMinWidth {
		width = settingsInfoMinWidth
	}
	rect := objc.Send[CGRect](nsString(text), sel("boundingRectWithSize:options:attributes:"),
		CGSize{Width: width - 2*settingsInfoPadding, Height: 1e6}, uint64(1|2), attributes)
	height := math.Ceil(rect.Size.Height) + 2*settingsInfoPadding
	if height < settingsInfoMinHeight {
		height = settingsInfoMinHeight
	}
	return math.Ceil(width), height
}

func (v *SettingsView) closeSettingsInfoButtons() {
	for _, button := range v.infoButtons {
		if info := settingsInfos[button]; info != nil {
			info.close()
			delete(settingsInfos, button)
		}
		button.Send(sel("release"))
	}
	v.infoButtons = nil
}
