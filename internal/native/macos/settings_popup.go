//go:build darwin

package macos

import (
	"sync"
	"time"

	"github.com/ebitengine/purego/objc"
)

const (
	settingsPopupHeight           = settingsControlHeight
	settingsPopupTextSize         = 13.0
	settingsPopupTextInset        = 12.0
	settingsPopupGap              = 8.0
	settingsPopupIconSize         = 15.0
	settingsPopupIconInset        = 4.0
	settingsPopupMinWidth         = settingsPopupTextInset + settingsPopupGap + settingsPopupIconSize + settingsPopupIconInset
	settingsPopupMaxWidth         = 260.0
	settingsPopupChevronPad       = 2.0
	settingsPopupChevronHalfWidth = 2.4
	settingsPopupChevronHeight    = 2.2
	settingsPopupChevronLineWidth = 1.4
)

var (
	registerSettingsPopupOnce sync.Once
	settingsPopupClass        objc.Class
	settingsPopupHovers       = map[objc.ID]*transition{}
	suppressedPopupHover      = map[objc.ID]time.Time{}
)

// registerSettingsPopupClass replaces only the collapsed popup drawing.
// NSPopUpButton still owns the click handling and the native NSMenu.
func registerSettingsPopupClass() {
	registerSettingsPopupOnce.Do(func() {
		var err error
		settingsPopupClass, err = objc.RegisterClass("EcoPasteSettingsPopup", class("NSPopUpButton"), nil, nil,
			[]objc.MethodDef{
				{Cmd: sel("drawRect:"), Fn: drawSettingsPopup},
				{Cmd: sel("updateTrackingAreas"), Fn: func(self objc.ID, _ objc.SEL) {
					sendSuper[objc.ID](self, settingsPopupClass, sel("updateTrackingAreas"))
					area := objc.ID(class("NSTrackingArea")).Send(sel("alloc")).
						Send(sel("initWithRect:options:owner:userInfo:"), CGRect{}, uint64(1|128|512), self, objc.ID(0))
					self.Send(sel("addTrackingArea:"), area)
					area.Send(sel("release"))
				}},
				{Cmd: sel("mouseEntered:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
					if until, ok := suppressedPopupHover[self]; ok && time.Now().Before(until) {
						return
					}
					delete(suppressedPopupHover, self)
					if hover := settingsPopupHovers[self]; hover != nil {
						hover.setTarget(1)
					}
				}},
				{Cmd: sel("mouseExited:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
					if hover := settingsPopupHovers[self]; hover != nil {
						hover.setTarget(0)
					}
				}},
				{Cmd: sel("mouseDown:"), Fn: func(self objc.ID, cmd objc.SEL, event objc.ID) {
					suppressSettingsPopupHover(self)
					sendSuper[objc.ID](self, settingsPopupClass, cmd, event)
				}},
				{Cmd: sel("viewDidChangeEffectiveAppearance"), Fn: func(self objc.ID, _ objc.SEL) {
					sendSuper[objc.ID](self, settingsPopupClass, sel("viewDidChangeEffectiveAppearance"))
					self.Send(sel("setNeedsDisplay:"), true)
				}},
			})
		if err != nil {
			panic(err)
		}
	})
}

func newSettingsPopup(frame CGRect) objc.ID {
	registerSettingsPopupClass()
	control := objc.ID(settingsPopupClass).Send(sel("alloc")).Send(sel("initWithFrame:pullsDown:"), frame, false)
	control.Send(sel("setFocusRingType:"), int64(1))
	settingsPopupHovers[control] = newTransition(func() { control.Send(sel("setNeedsDisplay:"), true) })
	return control
}

func drawSettingsPopup(self objc.ID, _ objc.SEL, _ CGRect) {
	rect := bounds(self)
	width := settingsPopupWidth(settingsPopupTitle(self))
	if delta := rect.Size.Width - width; delta < -0.5 || delta > 0.5 {
		fitSettingsPopupWidth(self, width)
		rect.Size.Width = width
	}
	radius := rect.Size.Height / 2
	fill := settingsPopupFill(self)
	hover := settingsPopupHoverValue(self)

	if hover > 0.001 {
		fillRoundedRect(rect, radius, fadeColor(fill, hover))
	}

	text := settingsPopupTitle(self)

	iconRect := CGRect{
		Origin: CGPoint{
			X: rect.Size.Width - settingsPopupIconInset - settingsPopupIconSize,
			Y: (rect.Size.Height - settingsPopupIconSize) / 2,
		},
		Size: CGSize{Width: settingsPopupIconSize, Height: settingsPopupIconSize},
	}
	iconPath := objc.ID(class("NSBezierPath")).Send(sel("bezierPathWithOvalInRect:"), iconRect)
	if hover < 0.999 {
		fadeColor(fill, 1-hover).Send(sel("setFill"))
		iconPath.Send(sel("fill"))
	}

	textX := settingsPopupTextInset
	textWidth := rect.Size.Width - textX - settingsPopupGap - settingsPopupIconSize - settingsPopupIconInset
	if textWidth > 0 {
		paragraph := objc.ID(class("NSMutableParagraphStyle")).Send(sel("new"))
		paragraph.Send(sel("setLineBreakMode:"), int64(4))
		attributes := objc.ID(class("NSMutableDictionary")).Send(sel("dictionary"))
		attributes.Send(sel("setObject:forKey:"), systemFont(settingsPopupTextSize), nsString("NSFont"))
		attributes.Send(sel("setObject:forKey:"), settingsPopupTextColor(self), nsString("NSColor"))
		attributes.Send(sel("setObject:forKey:"), paragraph, nsString("NSParagraphStyle"))
		textHeight := objc.Send[CGSize](nsString(text), sel("sizeWithAttributes:"), attributes).Height
		textRect := CGRect{
			Origin: CGPoint{X: textX, Y: (rect.Size.Height - textHeight) / 2},
			Size:   CGSize{Width: textWidth, Height: textHeight},
		}
		nsString(text).Send(sel("drawInRect:withAttributes:"), textRect, attributes)
		paragraph.Send(sel("release"))
	}

	drawSettingsPopupChevrons(self, iconRect)
}

func drawSettingsPopupChevrons(view objc.ID, rect CGRect) {
	iconColor := objc.ID(classNSColor).Send(sel("secondaryLabelColor"))
	if settingsPopupDark(view) {
		iconColor = objc.ID(classNSColor).Send(sel("whiteColor")).Send(sel("colorWithAlphaComponent:"), 0.78)
	}
	centerX := rect.Origin.X + rect.Size.Width/2
	centerY := rect.Origin.Y + rect.Size.Height/2

	for _, direction := range []float64{1, -1} {
		path := objc.ID(class("NSBezierPath")).Send(sel("bezierPath"))
		path.Send(sel("moveToPoint:"), CGPoint{
			X: centerX - settingsPopupChevronHalfWidth,
			Y: centerY + direction*settingsPopupChevronPad,
		})
		path.Send(sel("lineToPoint:"), CGPoint{
			X: centerX,
			Y: centerY + direction*(settingsPopupChevronPad+settingsPopupChevronHeight),
		})
		path.Send(sel("lineToPoint:"), CGPoint{
			X: centerX + settingsPopupChevronHalfWidth,
			Y: centerY + direction*settingsPopupChevronPad,
		})
		path.Send(sel("setLineWidth:"), settingsPopupChevronLineWidth)
		path.Send(sel("setLineCapStyle:"), int64(1))
		path.Send(sel("setLineJoinStyle:"), int64(1))
		iconColor.Send(sel("setStroke"))
		path.Send(sel("stroke"))
	}
}

func settingsPopupFill(view objc.ID) objc.ID {
	if settingsPopupDark(view) {
		return objc.ID(classNSColor).Send(sel("whiteColor")).Send(sel("colorWithAlphaComponent:"), 0.11)
	}
	return objc.ID(classNSColor).Send(sel("blackColor")).Send(sel("colorWithAlphaComponent:"), 0.08)
}

func settingsPopupTextColor(view objc.ID) objc.ID {
	if settingsPopupDark(view) {
		return objc.ID(classNSColor).Send(sel("whiteColor"))
	}
	return objc.ID(classNSColor).Send(sel("labelColor"))
}

func settingsPopupDark(view objc.ID) bool {
	if appearanceIsDark(view) {
		return true
	}
	if window := objc.ID(view.Send(sel("window"))); window != 0 && appearanceIsDark(window) {
		return true
	}
	app := objc.ID(class("NSApplication")).Send(sel("sharedApplication"))
	return app != 0 && appearanceIsDark(app)
}

func settingsPopupWidth(title string) float64 {
	width := settingsPopupTextInset + textWidth(title, settingsPopupTextSize) +
		settingsPopupGap + settingsPopupIconSize + settingsPopupIconInset
	return min(max(width, settingsPopupMinWidth), settingsPopupMaxWidth)
}

func settingsPopupTitle(view objc.ID) string {
	text := goString(view.Send(sel("titleOfSelectedItem")))
	if text == "" {
		text = goString(view.Send(sel("title")))
	}
	return text
}

func fitSettingsPopupWidth(view objc.ID, width float64) {
	frame := objc.Send[CGRect](view, sel("frame"))
	frame.Origin.X += frame.Size.Width - width
	frame.Size.Width = width
	view.Send(sel("setFrame:"), frame)
}

func settingsPopupHoverValue(view objc.ID) float64 {
	if hover := settingsPopupHovers[view]; hover != nil {
		return hover.value
	}
	return 0
}

func clearSettingsPopupHover(view objc.ID) {
	if hover := settingsPopupHovers[view]; hover != nil {
		hover.snap(0)
	}
	view.Send(sel("setNeedsDisplay:"), true)
}

func suppressSettingsPopupHover(view objc.ID) {
	suppressedPopupHover[view] = time.Now().Add(200 * time.Millisecond)
	clearSettingsPopupHover(view)
}

func discardSettingsPopup(view objc.ID) {
	if hover := settingsPopupHovers[view]; hover != nil {
		hover.stop()
		delete(settingsPopupHovers, view)
	}
	delete(suppressedPopupHover, view)
}
