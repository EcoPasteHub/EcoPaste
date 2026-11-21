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
