//go:build darwin && !ios

package appkit

import (
	"fmt"
	"sync"

	"github.com/ebitengine/purego/objc"
	"github.com/nsgo-dev/nsgo/core"
)

var sel = objc.RegisterName

type cgPoint struct{ X, Y float64 }
type cgSize struct{ Width, Height float64 }
type cgRect struct {
	Origin cgPoint
	Size   cgSize
}

func rect(x, y, width, height float64) cgRect {
	return cgRect{Origin: cgPoint{X: x, Y: y}, Size: cgSize{Width: width, Height: height}}
}

func nsString(value string) objc.ID {
	return objc.ID(objc.GetClass("NSString")).Send(sel("stringWithUTF8String:"), value+"\x00")
}

var (
	registerOnce sync.Once
	registerErr  error

	rootViewClass objc.Class
	rootTarget    objc.ID

	buttonRoots = map[objc.ID]*rootComponent{}
)

type rootComponent struct {
	view       objc.ID
	button     objc.ID
	clickLabel objc.ID
	cx         *core.NativeContext
}

// NewRoot creates the EcoPaste AppKit root view and its native controls.
func NewRoot(cx *core.NativeContext) core.NativeComponent {
	registerOnce.Do(registerRuntime)
	if registerErr != nil {
		panic(registerErr)
	}

	view := objc.ID(rootViewClass).Send(sel("new"))
	view.Send(sel("setFrame:"), rect(0, 0, 420, 640))

	component := &rootComponent{
		view: view,
		cx:   cx,
	}

	title := newLabel("EcoPaste", 24, 24, 372, 34)
	title.Send(sel("setFont:"), systemFont(24))
	view.Send(sel("addSubview:"), title)

	subtitle := newLabel("Go + nsgo + AppKit application foundation", 24, 62, 372, 24)
	subtitle.Send(sel("setTextColor:"), secondaryLabelColor())
	view.Send(sel("addSubview:"), subtitle)

	component.clickLabel = newLabel("AppKit event chain ready", 24, 112, 372, 24)
	component.clickLabel.Send(sel("setTextColor:"), secondaryLabelColor())
	view.Send(sel("addSubview:"), component.clickLabel)

	component.button = objc.ID(objc.GetClass("NSButton")).Send(
		sel("buttonWithTitle:target:action:"),
		nsString("Test native control"),
		rootTarget,
		sel("primaryAction:"),
	)
	component.button.Send(sel("setFrame:"), rect(24, 156, 180, 32))
	component.button.Send(sel("setAutoresizingMask:"), uint(2))
	view.Send(sel("addSubview:"), component.button)
	buttonRoots[component.button] = component

	return component
}

func registerRuntime() {
	rootViewClass, registerErr = objc.RegisterClass(
		"EcoPasteRootView",
		objc.GetClass("NSView"),
		nil,
		nil,
		[]objc.MethodDef{{
			Cmd: sel("isFlipped"),
			Fn:  func(objc.ID, objc.SEL) bool { return true },
		}},
	)
	if registerErr != nil {
		return
	}

	class, err := objc.RegisterClass(
		"EcoPasteAppKitRootTarget",
		objc.GetClass("NSObject"),
		nil,
		nil,
		[]objc.MethodDef{{
			Cmd: sel("primaryAction:"),
			Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
				if component := buttonRoots[sender]; component != nil {
					component.cx.Emit(EventPrimaryAction, nil)
				}
			},
		}},
	)
	if err != nil {
		registerErr = err
		return
	}
	rootTarget = objc.ID(class).Send(sel("new"))
}

func newLabel(text string, x, y, width, height float64) objc.ID {
	label := objc.ID(objc.GetClass("NSTextField")).Send(sel("labelWithString:"), nsString(text))
	label.Send(sel("setFrame:"), rect(x, y, width, height))
	label.Send(sel("setAutoresizingMask:"), uint(2))
	return label
}

func systemFont(size float64) objc.ID {
	return objc.ID(objc.GetClass("NSFont")).Send(sel("systemFontOfSize:"), size)
}

func secondaryLabelColor() objc.ID {
	return objc.ID(objc.GetClass("NSColor")).Send(sel("secondaryLabelColor"))
}

func (c *rootComponent) View() any {
	return uintptr(c.view)
}

func (c *rootComponent) SetProp(name string, value any) {
	if name != "clickCount" {
		return
	}
	c.clickLabel.Send(sel("setStringValue:"), nsString(fmt.Sprintf("Native button clicks: %v", value)))
}

func (c *rootComponent) Command(string, any) {}

func (c *rootComponent) Close() {
	delete(buttonRoots, c.button)
	c.view.Send(sel("release"))
}
