//go:build darwin

package macos

import (
	"strings"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

// The clipboard panel is two windows: this capsule bar above, and the
// history window below, which owns it as a child window. The bar has a
// frosted capsule surface of its own so it stays translucent over a dark
// desktop.

const (
	panelNavHeight      = 48.0
	panelNavGap         = 10.0
	panelNavPadding     = 16.0
	panelNavPaddingEnd  = 12.0
	panelNavCapsuleH    = 30.0
	panelNavCapsulePad  = 11.0
	panelNavDotSize     = 10.0
	panelNavDotLabelGap = 6.0
	panelNavCapsuleGap  = 5.0
	panelNavActionSize  = 24.0
	panelNavTitleSize   = 13.0

	windowStyleTitled         = 1
	windowStyleNonactivating  = 1 << 7
	windowMaskFullSizeContent = 1 << 15
	windowOrderedAbove        = 1
)

// panelNavFilters lists the categories of the panel in the order the
// navigation bar shows them.
var panelNavFilters = []string{"", "text", "image", "files", "favorite"}

// panelNavDots are the capsule dot colors of each category.
var panelNavDots = []string{
	"tertiaryLabelColor", "systemBlueColor", "systemOrangeColor",
	"systemTealColor", "systemPinkColor",
}

var (
	navPanelClass      objc.Class
	navBackgroundClass objc.Class
	navEffectClass     objc.Class
	navSurfaceClass    objc.Class
	navCapsuleClass    objc.Class
	navIconClass       objc.Class
	navObserverClass   objc.Class

	navsByCapsule    = map[objc.ID]*panelNav{}
	navsByPlus       = map[objc.ID]*panelNav{}
	navsByBackground = map[objc.ID]*panelNav{}
	navsByParentNote = map[objc.ID]*panelNav{}
	hoveredCapsules  = map[objc.ID]bool{}
)

// panelNav is the top child window of the clipboard panel.
type panelNav struct {
	panel    *Panel
	window   objc.ID
	surface  objc.ID
	effect   objc.ID
	tint     objc.ID
	selected string
	capsules []objc.ID
	plus     objc.ID
	width    float64
	pad      float64
	observer objc.ID
}

func registerNavClasses() error {
	var err error
	navPanelClass, err = objc.RegisterClass("EcoPasteNavPanel", class("NSPanel"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("canBecomeKeyWindow"), Fn: func(_ objc.ID, _ objc.SEL) bool { return false }},
			{Cmd: sel("canBecomeMainWindow"), Fn: func(_ objc.ID, _ objc.SEL) bool { return false }},
			{Cmd: sel("acceptsFirstMouse:"), Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) bool { return true }},
		})
	if err != nil {
		return err
	}
	navBackgroundClass, err = objc.RegisterClass("EcoPasteNavBackground", class("NSView"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("mouseDown:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if nav := navsByBackground[self]; nav != nil {
					nav.beginNavDrag()
				}
			}},
			{Cmd: sel("mouseDownCanMoveWindow"), Fn: func(_ objc.ID, _ objc.SEL) bool { return false }},
			{Cmd: sel("acceptsFirstMouse:"), Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) bool { return true }},
		})
	if err != nil {
		return err
	}
	navEffectClass, err = objc.RegisterClass("EcoPasteNavEffect", class("NSVisualEffectView"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("hitTest:"), Fn: func(_ objc.ID, _ objc.SEL, _ CGPoint) objc.ID { return 0 }},
		})
	if err != nil {
		return err
	}
	navSurfaceClass, err = objc.RegisterClass("EcoPasteNavTint", class("NSView"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("drawRect:"), Fn: func(self objc.ID, _ objc.SEL, _ CGRect) {
				fillRoundedRect(bounds(self), panelNavHeight/2, navSurfaceTint(self))
			}},
			{Cmd: sel("viewDidChangeEffectiveAppearance"), Fn: func(self objc.ID, _ objc.SEL) {
				sendSuper[objc.ID](self, navSurfaceClass, sel("viewDidChangeEffectiveAppearance"))
				self.Send(sel("setNeedsDisplay:"), true)
			}},
			{Cmd: sel("hitTest:"), Fn: func(_ objc.ID, _ objc.SEL, _ CGPoint) objc.ID { return 0 }},
		})
	if err != nil {
		return err
	}
	navCapsuleClass, err = objc.RegisterClass("EcoPasteNavCapsule", class("NSButton"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("drawRect:"), Fn: drawNavCapsule},
			{Cmd: sel("acceptsFirstMouse:"), Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) bool { return true }},
			{Cmd: sel("hitTest:"), Fn: func(self objc.ID, _ objc.SEL, point CGPoint) objc.ID {
				frame := objc.Send[CGRect](self, sel("frame"))
				rect := objc.Send[CGRect](self, sel("bounds"))
				if point.X >= frame.Origin.X && point.Y >= frame.Origin.Y &&
					point.X < frame.Origin.X+rect.Size.Width && point.Y < frame.Origin.Y+rect.Size.Height {
					return self
				}
				return 0
			}},
			{Cmd: sel("updateTrackingAreas"), Fn: func(self objc.ID, _ objc.SEL) {
				sendSuper[objc.ID](self, navCapsuleClass, sel("updateTrackingAreas"))
				area := objc.ID(class("NSTrackingArea")).Send(sel("alloc")).
					Send(sel("initWithRect:options:owner:userInfo:"), CGRect{}, uint64(1|128|512), self, objc.ID(0))
				self.Send(sel("addTrackingArea:"), area)
				area.Send(sel("release"))
			}},
			{Cmd: sel("mouseEntered:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				hoveredCapsules[self] = true
				self.Send(sel("setNeedsDisplay:"), true)
			}},
			{Cmd: sel("mouseExited:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				hoveredCapsules[self] = false
				self.Send(sel("setNeedsDisplay:"), true)
			}},
			{Cmd: sel("viewDidChangeEffectiveAppearance"), Fn: func(self objc.ID, _ objc.SEL) {
				sendSuper[objc.ID](self, navCapsuleClass, sel("viewDidChangeEffectiveAppearance"))
				self.Send(sel("setNeedsDisplay:"), true)
			}},
		})
	if err != nil {
		return err
	}
	navIconClass, err = objc.RegisterClass("EcoPasteNavIcon", class("NSButton"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("acceptsFirstMouse:"), Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) bool { return true }},
		})
	if err != nil {
		return err
	}
	navObserverClass, err = objc.RegisterClass("EcoPasteNavObserver", class("NSObject"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("parentChanged:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if nav := navsByParentNote[self]; nav != nil {
					nav.position()
				}
			}},
		})
	return err
}

// newPanelNav builds the capsule bar window for p and makes it a child
// window of the panel window.
func newPanelNav(p *Panel, parent objc.ID) *panelNav {
	nav := &panelNav{panel: p, pad: panelNavCapsulePad}
	parentFrame := objc.Send[CGRect](parent, sel("frame"))
	nav.width = parentFrame.Size.Width
	nav.window = objc.ID(navPanelClass).Send(sel("alloc")).
		Send(sel("initWithContentRect:styleMask:backing:defer:"),
			CGRect{Size: CGSize{Width: nav.width, Height: panelNavHeight}},
			uint(windowStyleTitled|windowStyleNonactivating|windowMaskFullSizeContent), uint(2), false)
	nav.window.Send(sel("setTitle:"), nsString(""))
	nav.window.Send(sel("setTitlebarAppearsTransparent:"), true)
	nav.window.Send(sel("setTitleVisibility:"), int64(1))
	for i := 0; i < 3; i++ {
		if button := nav.window.Send(sel("standardWindowButton:"), int64(i)); button != 0 {
			button.Send(sel("setHidden:"), true)
		}
	}
	nav.window.Send(sel("setReleasedWhenClosed:"), false)
	nav.window.Send(sel("setMovable:"), false)
	nav.window.Send(sel("setFloatingPanel:"), true)
	nav.window.Send(sel("setHidesOnDeactivate:"), false)
	nav.window.Send(sel("setLevel:"), objc.Send[int64](parent, sel("level")))
	nav.window.Send(sel("setCollectionBehavior:"), int64(1|256))

	surface := objc.ID(navBackgroundClass).Send(sel("new"))
	surface.Send(sel("setWantsLayer:"), true)
	surface.Send(sel("setAutoresizesSubviews:"), false)
	layer := surface.Send(sel("layer"))
	layer.Send(sel("setCornerRadius:"), panelNavHeight/2)
	layer.Send(sel("setMasksToBounds:"), true)
	nav.surface = surface
	navsByBackground[surface] = nav
	nav.window.Send(sel("setContentView:"), surface)
	nav.window.Send(sel("setOpaque:"), false)
	nav.window.Send(sel("setBackgroundColor:"), objc.ID(classNSColor).Send(sel("clearColor")))
	nav.window.Send(sel("setHasShadow:"), true)

	nav.effect = objc.ID(navEffectClass).Send(sel("new"))
	nav.effect.Send(sel("setMaterial:"), int64(6))
	nav.effect.Send(sel("setBlendingMode:"), int64(0))
	nav.effect.Send(sel("setState:"), int64(1))
	surface.Send(sel("addSubview:"), nav.effect)

	nav.tint = objc.ID(navSurfaceClass).Send(sel("new"))
	nav.tint.Send(sel("setAutoresizesSubviews:"), false)
	surface.Send(sel("addSubview:"), nav.tint)
	nav.tint.Send(sel("release"))
	nav.effect.Send(sel("release"))
	surface.Send(sel("release"))

	labels := panelNavLabels()
	for index := range panelNavFilters {
		capsule := objc.ID(navCapsuleClass).Send(sel("new"))
		capsule.Send(sel("setBordered:"), false)
		capsule.Send(sel("setTag:"), int64(index))
		capsule.Send(sel("setTarget:"), searchTarget)
		capsule.Send(sel("setAction:"), sel("capsulePressed:"))
		capsule.Send(sel("setAccessibilityLabel:"), nsString(labels[index]))
		capsule.Send(sel("setAccessibilityRole:"), nsString("AXButton"))
		navsByCapsule[capsule] = nav
		nav.capsules = append(nav.capsules, capsule)
		nav.surface.Send(sel("addSubview:"), capsule)
	}

	nav.plus = objc.ID(navIconClass).Send(sel("alloc")).Send(sel("initWithFrame:"), CGRect{})
	nav.plus.Send(sel("setBordered:"), false)
	nav.plus.Send(sel("setImage:"), systemSymbol("plus"))
	nav.plus.Send(sel("setImagePosition:"), int64(1))
	nav.plus.Send(sel("setContentTintColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	nav.plus.Send(sel("setTarget:"), searchTarget)
	nav.plus.Send(sel("setAction:"), sel("navPlusPressed:"))
	nav.plus.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(i18n.PanelNewGroup)))
	navsByPlus[nav.plus] = nav
	nav.surface.Send(sel("addSubview:"), nav.plus)

	parent.Send(sel("addChildWindow:ordered:"), nav.window, int64(windowOrderedAbove))
	nav.layout()
	nav.position()
	nav.observeParent(parent)
	nav.applyFilter(p.filterValue)
	return nav
}

func (n *panelNav) observeParent(parent objc.ID) {
	center := objc.ID(class("NSNotificationCenter")).Send(sel("defaultCenter"))
	n.observer = objc.ID(navObserverClass).Send(sel("new"))
	navsByParentNote[n.observer] = n
	for _, name := range []string{"NSWindowDidMoveNotification", "NSWindowDidResizeNotification"} {
		center.Send(sel("addObserver:selector:name:object:"), n.observer, sel("parentChanged:"),
			nsString(name), parent)
	}
}

// position puts the bar right above the panel window, with the same
// width, so the two read as one floating stack.
func (n *panelNav) position() {
	parent := n.panel.view.Send(sel("window"))
	if parent == 0 || n.window == 0 {
		return
	}
	frame := objc.Send[CGRect](parent, sel("frame"))
	if frame.Size.Width != n.width {
		n.width = frame.Size.Width
		n.layout()
	}
	n.window.Send(sel("setFrame:display:"),
		CGRect{
			Origin: CGPoint{X: frame.Origin.X, Y: frame.Origin.Y + frame.Size.Height + panelNavGap},
			Size:   CGSize{Width: frame.Size.Width, Height: panelNavHeight},
		}, true)
}

func (n *panelNav) layout() {
	setFrame(n.effect, 0, 0, n.width, panelNavHeight)
	setFrame(n.tint, 0, 0, n.width, panelNavHeight)
	labels := panelNavLabels()
	pad, gap := panelNavCapsulePadding(labels, n.width)
	n.pad = pad
	x := panelNavPadding
	y := (panelNavHeight - panelNavCapsuleH) / 2
	for index, capsule := range n.capsules {
		capsuleWidth := panelNavCapsuleWidthWith(labels[index], pad)
		setFrame(capsule, x, y, capsuleWidth, panelNavCapsuleH)
		x += capsuleWidth + gap
	}
	setFrame(n.plus, n.width-panelNavPaddingEnd-panelNavActionSize,
		(panelNavHeight-panelNavActionSize)/2, panelNavActionSize, panelNavActionSize)
}

func panelNavLabels() []string {
	return []string{
		i18n.T(i18n.PanelAll), i18n.T(i18n.PanelTextCategory), i18n.T(i18n.PanelImageCategory),
		i18n.T(i18n.PanelFilesCategory), i18n.T(i18n.PanelFavorites),
	}
}

func panelNavCapsuleWidth(label string) float64 {
	return panelNavCapsuleWidthWith(label, panelNavCapsulePad)
}

func panelNavCapsuleWidthWith(label string, pad float64) float64 {
	return pad*2 + panelNavDotSize + panelNavDotLabelGap + textWidth(label, panelNavTitleSize) + 1
}

// panelNavCapsulePadding tightens the capsules when a longer locale
// would push the new-item button out of the bar.
func panelNavCapsulePadding(labels []string, width float64) (pad, gap float64) {
	pad, gap = panelNavCapsulePad, panelNavCapsuleGap
	if panelNavCapsulesFit(labels, pad, gap, width) {
		return pad, gap
	}
	return 6, 2
}

func panelNavCapsulesFit(labels []string, pad, gap, width float64) bool {
	used := panelNavPadding + panelNavPaddingEnd + panelNavActionSize
	for _, label := range labels {
		used += panelNavCapsuleWidthWith(label, pad) + gap
	}
	return used <= width
}

// applyFilter selects the capsule that matches a panel filter value.
func (n *panelNav) applyFilter(filter string) {
	n.selected = filter
	for _, capsule := range n.capsules {
		capsule.Send(sel("setNeedsDisplay:"), true)
	}
}

func (n *panelNav) show() {
	if n.window == 0 {
		return
	}
	n.position()
	n.window.Send(sel("orderFront:"), objc.ID(0))
}

func (n *panelNav) hide() {
	if n.window == 0 {
		return
	}
	n.window.Send(sel("orderOut:"), objc.ID(0))
}

func (n *panelNav) close() {
	if n.window == 0 {
		return
	}
	center := objc.ID(class("NSNotificationCenter")).Send(sel("defaultCenter"))
	if n.observer != 0 {
		center.Send(sel("removeObserver:"), n.observer)
		delete(navsByParentNote, n.observer)
		n.observer.Send(sel("release"))
		n.observer = 0
	}
	if parent := n.panel.view.Send(sel("window")); parent != 0 {
		parent.Send(sel("removeChildWindow:"), n.window)
	}
	delete(navsByBackground, n.surface)
	n.window.Send(sel("orderOut:"), objc.ID(0))
	n.window.Send(sel("close"))
	for _, capsule := range n.capsules {
		delete(navsByCapsule, capsule)
		delete(hoveredCapsules, capsule)
		capsule.Send(sel("release"))
	}
	delete(navsByPlus, n.plus)
	n.plus.Send(sel("release"))
	n.window.Send(sel("release"))
	n.window = 0
}

func drawNavCapsule(self objc.ID, _ objc.SEL, _ CGRect) {
	nav := navsByCapsule[self]
	if nav == nil {
		return
	}
	index := int(objc.Send[int64](self, sel("tag")))
	capsuleRect := bounds(self)
	labels := panelNavLabels()
	selected := index < len(panelNavFilters) && panelNavFilters[index] == nav.selected
	if selected || hoveredCapsules[self] {
		fillRoundedRect(capsuleRect, panelNavCapsuleH/2, navCapsuleFill(self, selected))
	}

	dotRect := CGRect{
		Origin: CGPoint{X: nav.pad, Y: (capsuleRect.Size.Height - panelNavDotSize) / 2},
		Size:   CGSize{Width: panelNavDotSize, Height: panelNavDotSize},
	}
	if index < len(panelNavDots) {
		dotColor := objc.ID(classNSColor).Send(sel(panelNavDots[index]))
		path := objc.ID(class("NSBezierPath")).Send(sel("bezierPathWithOvalInRect:"), dotRect)
		dotColor.Send(sel("setFill"))
		path.Send(sel("fill"))
	}

	label := ""
	if index < len(labels) {
		label = labels[index]
	}
	textColor := objc.ID(classNSColor).Send(sel("secondaryLabelColor"))
	if selected {
		textColor = objc.ID(classNSColor).Send(sel("labelColor"))
	}
	attributes := objc.ID(class("NSMutableDictionary")).Send(sel("dictionary"))
	attributes.Send(sel("setObject:forKey:"), systemFont(panelNavTitleSize), nsString("NSFont"))
	attributes.Send(sel("setObject:forKey:"), textColor, nsString("NSForegroundColor"))
	textSize := objc.Send[CGSize](nsString(label), sel("sizeWithAttributes:"), attributes)
	textX := nav.pad + panelNavDotSize + panelNavDotLabelGap
	textRect := CGRect{
		Origin: CGPoint{X: textX, Y: (capsuleRect.Size.Height - textSize.Height) / 2},
		Size:   CGSize{Width: capsuleRect.Size.Width - textX - nav.pad, Height: textSize.Height},
	}
	nsString(label).Send(sel("drawInRect:withAttributes:"), textRect, attributes)
}

func navCapsuleFill(view objc.ID, selected bool) objc.ID {
	color := objc.ID(classNSColor).Send(sel("whiteColor"))
	if appearanceIsDark(view) {
		if selected {
			return color.Send(sel("colorWithAlphaComponent:"), 0.18)
		}
		return color.Send(sel("colorWithAlphaComponent:"), 0.08)
	}
	if selected {
		return color.Send(sel("colorWithAlphaComponent:"), 0.62)
	}
	return color.Send(sel("colorWithAlphaComponent:"), 0.30)
}

func navSurfaceTint(view objc.ID) objc.ID {
	color := objc.ID(classNSColor).Send(sel("whiteColor"))
	if appearanceIsDark(view) {
		return color.Send(sel("colorWithAlphaComponent:"), 0.06)
	}
	return color.Send(sel("colorWithAlphaComponent:"), 0.16)
}

func fillRoundedRect(rect CGRect, radius float64, color objc.ID) {
	path := objc.ID(class("NSBezierPath")).Send(sel("bezierPathWithRoundedRect:xRadius:yRadius:"),
		rect, radius, radius)
	color.Send(sel("setFill"))
	path.Send(sel("fill"))
}

func appearanceIsDark(view objc.ID) bool {
	name := goString(view.Send(sel("effectiveAppearance")).Send(sel("name")))
	return strings.Contains(name, "Dark")
}

func textWidth(label string, size float64) float64 {
	attributes := objc.ID(class("NSDictionary")).Send(sel("dictionaryWithObject:forKey:"),
		systemFont(size), nsString("NSFont"))
	measured := objc.Send[CGSize](nsString(label), sel("sizeWithAttributes:"), attributes)
	return measured.Width + 0.5
}
