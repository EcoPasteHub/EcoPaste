//go:build darwin

package macos

import (
	"sync"

	"github.com/ebitengine/purego/objc"
)

// PanelFrame is a panel window's frame in screen points.
type PanelFrame struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
}

// PanelPlacement says where the clipboard panel opens next. Frame is the
// remembered frame used by the remember mode.
type PanelPlacement struct {
	Mode     string
	Frame    PanelFrame
	HasFrame bool
}

const panelScreenMargin = 8.0

// ScreenBounds is one screen's frame and its visible part, in the global
// bottom-left coordinate space.
type ScreenBounds struct {
	Frame   CGRect
	Visible CGRect
}

var pendingPlacement struct {
	sync.Mutex
	value *PanelPlacement
}

// PlacePanelOnNextShow arranges where the clipboard panel opens. The
// platform places a panel every time it shows, so the placement is kept
// until the panel reports that it ordered on screen.
func PlacePanelOnNextShow(placement PanelPlacement) {
	pendingPlacement.Lock()
	pendingPlacement.value = &placement
	pendingPlacement.Unlock()
}

// PanelFrameOf returns the clipboard panel's current frame.
func PanelFrameOf() (PanelFrame, bool) {
	var frame PanelFrame
	var ok bool
	onMain(func() {
		for _, p := range panelsByView {
			window := p.view.Send(sel("window"))
			if window == 0 {
				continue
			}
			rect := objc.Send[CGRect](window, sel("frame"))
			frame = PanelFrame{X: rect.Origin.X, Y: rect.Origin.Y, Width: rect.Size.Width, Height: rect.Size.Height}
			ok = true
			return
		}
	})
	return frame, ok
}

func takePlacement() *PanelPlacement {
	pendingPlacement.Lock()
	defer pendingPlacement.Unlock()
	placement := pendingPlacement.value
	pendingPlacement.value = nil
	return placement
}

// applyPlacement moves the panel after the platform ordered it on
// screen, overriding the platform's own placement.
func applyPlacement(p *Panel) {
	placement := takePlacement()
	if placement == nil {
		return
	}
	window := p.view.Send(sel("window"))
	if window == 0 {
		return
	}
	frame := objc.Send[CGRect](window, sel("frame"))
	var saved *CGPoint
	if placement.HasFrame {
		saved = &CGPoint{X: placement.Frame.X, Y: placement.Frame.Y}
	}
	frame.Origin = panelOrigin(placement.Mode, frame.Size, mouseLocation(), screensBounds(), saved)
	window.Send(sel("setFrameOrigin:"), frame.Origin)
}

// panelOrigin computes a panel's origin for a placement mode: the saved
// position, the center of the screen with the pointer, or the pointer
// itself with the panel hanging below it.
func panelOrigin(mode string, size CGSize, mouse CGPoint, screens []ScreenBounds, saved *CGPoint) CGPoint {
	screen, ok := screenContaining(mouse, screens)
	if !ok {
		switch {
		case len(screens) > 0:
			screen = screens[0]
		default:
			return CGPoint{X: mouse.X, Y: mouse.Y - size.Height}
		}
	}
	switch mode {
	case "remember":
		if saved != nil {
			if remembered, found := screenContaining(*saved, screens); found {
				return clampOrigin(*saved, size, remembered.Visible)
			}
		}
		fallthrough
	case "center":
		visible := screen.Visible
		return clampOrigin(CGPoint{
			X: visible.Origin.X + (visible.Size.Width-size.Width)/2,
			Y: visible.Origin.Y + (visible.Size.Height-size.Height)/2,
		}, size, visible)
	default:
		return clampOrigin(CGPoint{X: mouse.X, Y: mouse.Y - size.Height}, size, screen.Visible)
	}
}

func screenContaining(point CGPoint, screens []ScreenBounds) (ScreenBounds, bool) {
	for _, screen := range screens {
		frame := screen.Frame
		if point.X >= frame.Origin.X && point.X < frame.Origin.X+frame.Size.Width &&
			point.Y >= frame.Origin.Y && point.Y < frame.Origin.Y+frame.Size.Height {
			return screen, true
		}
	}
	return ScreenBounds{}, false
}

func clampOrigin(origin CGPoint, size CGSize, visible CGRect) CGPoint {
	minX := visible.Origin.X + panelScreenMargin
	minY := visible.Origin.Y + panelScreenMargin
	maxX := max(minX, visible.Origin.X+visible.Size.Width-size.Width-panelScreenMargin)
	maxY := max(minY, visible.Origin.Y+visible.Size.Height-size.Height-panelScreenMargin)
	x := min(max(origin.X, minX), maxX)
	y := min(max(origin.Y, minY), maxY)
	return CGPoint{X: x, Y: y}
}

func mouseLocation() CGPoint {
	return objc.Send[CGPoint](objc.ID(class("NSEvent")), sel("mouseLocation"))
}

func screensBounds() []ScreenBounds {
	screens := objc.ID(class("NSScreen")).Send(sel("screens"))
	count := int(objc.Send[int64](screens, sel("count")))
	bounds := make([]ScreenBounds, 0, count)
	for i := range count {
		screen := screens.Send(sel("objectAtIndex:"), int64(i))
		bounds = append(bounds, ScreenBounds{
			Frame:   objc.Send[CGRect](screen, sel("frame")),
			Visible: objc.Send[CGRect](screen, sel("visibleFrame")),
		})
	}
	return bounds
}
