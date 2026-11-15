package macos

import (
	"github.com/ebitengine/purego/objc"
)

func registerPanelTargets() {
	var err error
	panelViewClass, err = objc.RegisterClass("EcoPastePanelView", class("NSView"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("viewDidMoveToWindow"), Fn: func(self objc.ID, _ objc.SEL) {
				sendSuper[objc.ID](self, panelViewClass, sel("viewDidMoveToWindow"))
				if p := panelsByView[self]; p != nil {
					applyPlacement(p)
					attachNavWindow(p)
				}
			}},
			{Cmd: sel("setFrameSize:"), Fn: func(self objc.ID, _ objc.SEL, size CGSize) {
				sendSuper[objc.ID](self, panelViewClass, sel("setFrameSize:"), size)
				if p := panelsByView[self]; p != nil {
					layoutPanel(p, size.Width, size.Height)
				}
			}},
			{Cmd: sel("mouseDown:"), Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
				p := panelsByView[self]
				if p == nil || objc.Send[int64](event, sel("buttonNumber")) != 0 {
					sendSuper[objc.ID](self, panelViewClass, sel("mouseDown:"), event)
					return
				}
				panelDragWindow(p, self.Send(sel("window")), nil)
			}},
			{Cmd: sel("acceptsFirstMouse:"), Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) bool { return true }},
			{Cmd: sel("keyDown:"), Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
				p := panelsByView[self]
				if p != nil && p.previewing && objc.Send[uint16](event, sel("keyCode")) == 53 {
					p.cx.Emit("hidePreview", nil)
					return
				}
				sendSuper[objc.ID](self, panelViewClass, sel("keyDown:"), event)
			}},
			{Cmd: sel("cancelOperation:"), Fn: func(self objc.ID, _ objc.SEL, sender objc.ID) {
				p := panelsByView[self]
				if p != nil && p.previewing {
					p.cx.Emit("hidePreview", nil)
					return
				}
				if p != nil {
					p.cx.Emit("hide", nil)
				}
			}},
		})
	if err != nil {
		panic(err)
	}

	target, err := objc.RegisterClass("EcoPasteSearchTarget", class("NSObject"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("searchChanged:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
				if p := panelsBySearch[sender]; p != nil {
					p.query = goString(sender.Send(sel("stringValue")))
					p.cx.Emit("query", p.query)
				}
			}},
			{Cmd: sel("pinPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
				if p := panelsByControl[sender]; p != nil {
					p.togglePinned()
				}
			}},
			{Cmd: sel("filterPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
				if p := panelsByControl[sender]; p != nil {
					p.showFilterMenu(sender)
				}
			}},
			{Cmd: sel("filterMenuPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
				if p := menuPanel(sender); p != nil {
					p.applyFilterMenuItem(int(objc.Send[int64](sender, sel("tag"))))
				}
			}},
			{Cmd: sel("settingsPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
				if p := panelsByControl[sender]; p != nil {
					p.cx.Emit("openSettings", "")
				}
			}},
			{Cmd: sel("clearPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
				if p := panelsByControl[sender]; p != nil {
					p.cx.Emit("clearHistory", nil)
				}
			}},
			{Cmd: sel("elevateDialog:"), Fn: func(_ objc.ID, _ objc.SEL, timer objc.ID) {
				window := objc.Send[objc.ID](timer, sel("userInfo"))
				if window == 0 {
					return
				}
				window.Send(sel("setLevel:"), int64(1000))
				placeDialogOverPanel(window)
				window.Send(sel("orderFront:"), objc.ID(0))
			}},
			{Cmd: sel("shortcutHelpPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
				if p := panelsByControl[sender]; p != nil {
					p.toggleShortcutHelp()
				}
			}},
			{Cmd: sel("capsulePressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
				nav := navsByCapsule[sender]
				if nav == nil {
					return
				}
				index := int(objc.Send[int64](sender, sel("tag")))
				if index < 0 || index >= len(panelNavFilters) {
					return
				}
				nav.applyFilter(panelNavFilters[index])
				nav.panel.cx.Emit("filter", panelNavFilters[index])
			}},
			{Cmd: sel("navPlusPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
				if nav := navsByPlus[sender]; nav != nil {
					nav.panel.cx.Emit("newGroup", nil)
				}
			}},
			{Cmd: sel("hoverActionPressed:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
				if p := panelsByControl[sender]; p != nil {
					p.runHoverAction(int(objc.Send[int64](sender, sel("tag"))))
				}
			}},
			{Cmd: sel("menuNewGroup:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
				if p := menuPanel(sender); p != nil {
					p.cx.Emit("newGroup", nil)
				}
			}},
			{Cmd: sel("menuRenameGroup:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
				if p := menuPanel(sender); p != nil && p.groupID != "" {
					p.cx.Emit("renameGroup", p.groupID)
				}
			}},
			{Cmd: sel("menuDeleteGroup:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
				if p := menuPanel(sender); p != nil && p.groupID != "" {
					p.cx.Emit("deleteGroup", p.groupID)
				}
			}},
		})
	if err != nil {
		panic(err)
	}
	searchTarget = objc.ID(target).Send(sel("new"))

	if err := registerNavClasses(); err != nil {
		panic(err)
	}
	if err := registerIconButtonClass(); err != nil {
		panic(err)
	}
	searchFieldClass, err = objc.RegisterClass("EcoPasteSearchField", classNSTextField, nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("acceptsFirstMouse:"), Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) bool { return true }},
			{Cmd: sel("mouseDown:"), Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
				if window := objc.Send[objc.ID](self, sel("window")); window != 0 {
					window.Send(sel("makeKeyWindow"))
					window.Send(sel("makeFirstResponder:"), self)
				}
				sendSuper[objc.ID](self, searchFieldClass, sel("mouseDown:"), event)
			}},
		})
	if err != nil {
		panic(err)
	}
	searchCellClass, err = objc.RegisterClass("EcoPasteSearchCell", class("NSTextFieldCell"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("drawingRectForBounds:"), Fn: func(self objc.ID, _ objc.SEL, bounds CGRect) CGRect {
				size := objc.Send[CGSize](self, sel("cellSize"))
				return CGRect{
					Origin: CGPoint{X: bounds.Origin.X, Y: bounds.Origin.Y + (bounds.Size.Height-size.Height)/2},
					Size:   CGSize{Width: bounds.Size.Width, Height: size.Height},
				}
			}},
		})
	if err != nil {
		panic(err)
	}
	if err := registerPanelBackdrop(); err != nil {
		panic(err)
	}
	observerClass, err = objc.RegisterClass("EcoPastePanelObserver", class("NSObject"), nil, nil,
		panelObserverMethods())
	if err != nil {
		panic(err)
	}
	registerHistoryTargets()
	registerPreviewTargets()
}
