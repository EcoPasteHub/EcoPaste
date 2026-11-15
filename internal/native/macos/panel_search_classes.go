//go:build darwin

package macos

import (
	"github.com/ebitengine/purego/objc"
)

func registerSearchClasses() error {
	var err error
	searchFieldClass, err = objc.RegisterClass("EcoPasteSearchField", classNSTextField, nil, nil, []objc.MethodDef{
		{Cmd: sel("acceptsFirstMouse:"), Fn: searchFieldAcceptsFirstMouse},
		{Cmd: sel("mouseDown:"), Fn: searchFieldMouseDown},
	})
	if err != nil {
		return err
	}
	searchCellClass, err = objc.RegisterClass("EcoPasteSearchCell", class("NSTextFieldCell"), nil, nil, []objc.MethodDef{
		{Cmd: sel("drawingRectForBounds:"), Fn: searchCellDrawingRect},
	})
	return err
}

func searchFieldAcceptsFirstMouse(_ objc.ID, _ objc.SEL, _ objc.ID) bool {
	return true
}

func searchFieldMouseDown(self objc.ID, _ objc.SEL, event objc.ID) {
	if window := objc.Send[objc.ID](self, sel("window")); window != 0 {
		window.Send(sel("makeKeyWindow"))
		window.Send(sel("makeFirstResponder:"), self)
	}
	sendSuper[objc.ID](self, searchFieldClass, sel("mouseDown:"), event)
}

func searchCellDrawingRect(self objc.ID, _ objc.SEL, bounds CGRect) CGRect {
	size := objc.Send[CGSize](self, sel("cellSize"))
	return CGRect{
		Origin: CGPoint{X: bounds.Origin.X, Y: bounds.Origin.Y + (bounds.Size.Height-size.Height)/2},
		Size:   CGSize{Width: bounds.Size.Width, Height: size.Height},
	}
}
