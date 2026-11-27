//go:build darwin

package macos

import (
	"sync"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

const (
	settingsContentX     = 28.0
	settingsContentWidth = settingsWidth - 2*settingsContentX
	settingsRowHeight    = 42.0
	settingsRowInset     = 16.0
	settingsCardRadius   = 12.0
	settingsGroupGap     = 24.0
	settingsSwitchWidth  = 40.0
	settingsSwitchHeight = 24.0
	// settingsControlHeight is the shared row control height: the native
	// NSSwitch size other interactive controls align to.
	settingsControlHeight   = settingsSwitchHeight
	settingsPageBottomInset = 24.0
)

var (
	registerSettingsLayoutOnce sync.Once
	settingsRootClass          objc.Class
	settingsDocumentClass      objc.Class
	settingsRoots              = map[objc.ID]*SettingsView{}
)

func settingsCardFillColor() objc.ID {
	return objc.ID(classNSColor).Send(sel("quaternarySystemFillColor"))
}

func settingsSeparatorColor() objc.ID {
	return objc.ID(classNSColor).Send(sel("separatorColor")).
		Send(sel("colorWithAlphaComponent:"), 0.55)
}

func (v *SettingsView) newPage(identifier string) objc.ID {
	registerSettingsLayoutOnce.Do(registerSettingsLayoutTargets)
	item := objc.ID(class("NSTabViewItem")).Send(sel("alloc")).Send(sel("initWithIdentifier:"), nsString(identifier))
	item.Send(sel("setLabel:"), nsString(SettingsTabTitle(identifier)))
	v.tabItems[identifier] = item

	page := objc.ID(class("NSView")).Send(sel("new"))
	setFrame(page, 0, 0, settingsWidth, settingsMinimumPageHeight)
	page.Send(sel("setAutoresizingMask:"), int64(18))

	scroll := objc.ID(class("NSScrollView")).Send(sel("new"))
	setFrame(scroll, 0, 0, settingsWidth, settingsMinimumPageHeight)
	scroll.Send(sel("setAutoresizingMask:"), int64(18))
	scroll.Send(sel("setHasVerticalScroller:"), true)
	scroll.Send(sel("setAutohidesScrollers:"), true)
	scroll.Send(sel("setDrawsBackground:"), false)
	scroll.Send(sel("setBorderType:"), int64(0))

	document := objc.ID(settingsDocumentClass).Send(sel("new"))
	setFrame(document, 0, 0, settingsWidth, settingsMinimumPageHeight)
	document.Send(sel("setAutoresizingMask:"), int64(2))
	scroll.Send(sel("setDocumentView:"), document)
	page.Send(sel("addSubview:"), scroll)

	item.Send(sel("setView:"), page)
	v.tabs.Send(sel("addTabViewItem:"), item)
	v.pageViews[identifier] = page

	document.Send(sel("release"))
	scroll.Send(sel("release"))
	page.Send(sel("release"))
	item.Send(sel("release"))

	v.currentTab = identifier
	return document
}

func (v *SettingsView) finishPage(document objc.ID) {
	height := settingsMeasuredPageHeight(document)
	v.pageHeights[v.currentTab] = height
	setFrame(document, 0, 0, settingsWidth, height)
}

func registerSettingsLayoutTargets() {
	var err error
	settingsRootClass, err = objc.RegisterClass("EcoPasteSettingsRoot", class("NSView"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("viewDidMoveToWindow"), Fn: settingsRootDidMoveToWindow},
			{Cmd: sel("setFrameSize:"), Fn: settingsRootSetFrameSize},
		})
	if err != nil {
		panic(err)
	}
	settingsDocumentClass, err = objc.RegisterClass("EcoPasteSettingsDocument", class("NSView"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("isFlipped"), Fn: func(objc.ID, objc.SEL) bool { return true }},
		})
	if err != nil {
		panic(err)
	}
}

func settingsRootDidMoveToWindow(self objc.ID, _ objc.SEL) {
	sendSuper[objc.ID](self, settingsRootClass, sel("viewDidMoveToWindow"))
	if view := settingsRoots[self]; view != nil && view.view.Send(sel("window")) != 0 {
		// The root joins its window before NSGo orders the window on screen.
		// Set the first page's exact size here so the user never sees the
		// placeholder frame used while the native tree is being built.
		view.resizeWindow(false)
	}
}

func settingsRootSetFrameSize(self objc.ID, _ objc.SEL, size CGSize) {
	sendSuper[objc.ID](self, settingsRootClass, sel("setFrameSize:"), size)
	if view := settingsRoots[self]; view != nil && view.tabs != 0 {
		view.tabs.Send(sel("setFrame:"), CGRect{Size: size})
	}
}

func settingsMeasuredPageHeight(document objc.ID) float64 {
	views := objc.Send[objc.ID](document, sel("subviews"))
	count := int(objc.Send[uint](views, sel("count")))
	contentBottom := 0.0
	for index := range count {
		view := objc.Send[objc.ID](views, sel("objectAtIndex:"), uint(index))
		frame := objc.Send[CGRect](view, sel("frame"))
		contentBottom = max(contentBottom, frame.Origin.Y+frame.Size.Height)
	}
	return max(settingsMinimumPageHeight, contentBottom+settingsPageBottomInset)
}

// addSectionCard places a section title and its rounded settings card.
// It returns the card's top edge in the flipped document.
func (v *SettingsView) addSectionCard(document objc.ID, top float64, title i18n.Key, rows int) (objc.ID, float64) {
	return v.addSectionCardInfo(document, top, title, "", rows)
}

// addSectionCardInfo is addSectionCard with an optional "?" hint next to the
// section title, for descriptions that explain a whole group.
func (v *SettingsView) addSectionCardInfo(document objc.ID, top float64, title, info i18n.Key, rows int) (objc.ID, float64) {
	v.addSectionLabel(document, top, title)
	if info != "" {
		v.addInfoAfterText(document, settingsContentX+4, top+8, title, settingsSectionFont(), info)
	}
	cardTop := top + 24
	height := float64(rows) * settingsRowHeight
	card := objc.ID(classNSBox).Send(sel("new"))
	card.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsContentX, Y: cardTop},
		Size:   CGSize{Width: settingsContentWidth, Height: height},
	})
	card.Send(sel("setBoxType:"), int64(4))
	card.Send(sel("setTitlePosition:"), int64(0))
	card.Send(sel("setBorderWidth:"), 0.0)
	card.Send(sel("setFillColor:"), settingsCardFillColor())
	card.Send(sel("setCornerRadius:"), settingsCardRadius)
	document.Send(sel("addSubview:"), card)
	card.Send(sel("release"))
	for row := 1; row < rows; row++ {
		v.addCardSeparator(document, cardTop, row)
	}
	return card, cardTop
}

func (v *SettingsView) addCardSeparator(document objc.ID, cardTop float64, row int) {
	v.addCardSeparatorOffset(document, cardTop, float64(row)*settingsRowHeight)
}

func (v *SettingsView) addCardSeparatorOffset(document objc.ID, cardTop, offset float64) {
	separator := objc.ID(classNSBox).Send(sel("new"))
	separator.Send(sel("setBoxType:"), int64(2))
	separator.Send(sel("setTitlePosition:"), int64(0))
	separator.Send(sel("setBorderColor:"), settingsSeparatorColor())
	separator.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{
			X: settingsContentX + settingsRowInset,
			Y: cardTop + offset - 1,
		},
		Size: CGSize{Width: settingsContentWidth - 2*settingsRowInset, Height: 1},
	})
	document.Send(sel("addSubview:"), separator)
	separator.Send(sel("release"))
}

func settingsRowTop(cardTop float64, row int) float64 {
	return cardTop + float64(row)*settingsRowHeight
}

func (v *SettingsView) addPageButton(document objc.ID, x, y, width, height float64, title i18n.Key, action string, help i18n.Key) objc.ID {
	button := objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(i18n.T(title)), settingsTarget, sel(action))
	button.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: x, Y: y}, Size: CGSize{Width: width, Height: height}})
	button.Send(sel("setBezelStyle:"), int64(1))
	v.bindText(func() {
		button.Send(sel("setTitle:"), nsString(i18n.T(title)))
	})
	if help != "" {
		button.Send(sel("setAccessibilityHelp:"), nsString(i18n.T(help)))
		v.bindText(func() {
			button.Send(sel("setAccessibilityHelp:"), nsString(i18n.T(help)))
		})
	}
	document.Send(sel("addSubview:"), button)
	settingsViews[button] = v
	return button
}
