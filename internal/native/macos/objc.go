//go:build darwin

package macos

import (
	"structs"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// CGPoint, CGSize and CGRect match CoreGraphics' layout on 64-bit Apple
// platforms.
type CGPoint struct {
	X, Y float64
}

type CGSize struct {
	Width, Height float64
}

type CGRect struct {
	Origin CGPoint
	Size   CGSize
}

var (
	sel = objc.RegisterName

	cfStringCreateWithBytes func(alloc uintptr, bytes *byte, numBytes int64, encoding uint32, external bool) uintptr
	msgSendSuper2           uintptr

	classNSFont        objc.Class
	classNSColor       objc.Class
	classNSBox         objc.Class
	classNSTextField   objc.Class
	classNSSearchField objc.Class
	classNSString      objc.Class

	utf8StringEncoding = uint32(0x08000100)
	windowDidBecomeKey = "NSWindowDidBecomeKeyNotification"
	windowDidResignKey = "NSWindowDidResignKeyNotification"
)

func init() {
	for _, path := range []string{
		"/System/Library/Frameworks/Foundation.framework/Foundation",
		"/System/Library/Frameworks/AppKit.framework/AppKit",
	} {
		if _, err := purego.Dlopen(path, purego.RTLD_GLOBAL|purego.RTLD_NOW); err != nil {
			panic(err)
		}
	}
	cf, err := purego.Dlopen("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation", purego.RTLD_GLOBAL|purego.RTLD_NOW)
	if err != nil {
		panic(err)
	}
	purego.RegisterLibFunc(&cfStringCreateWithBytes, cf, "CFStringCreateWithBytes")

	libobjc, err := purego.Dlopen("/usr/lib/libobjc.A.dylib", purego.RTLD_GLOBAL|purego.RTLD_NOW)
	if err != nil {
		panic(err)
	}
	if msgSendSuper2, err = purego.Dlsym(libobjc, "objc_msgSendSuper2"); err != nil {
		panic(err)
	}

	classNSFont = class("NSFont")
	classNSColor = class("NSColor")
	classNSBox = class("NSBox")
	classNSTextField = class("NSTextField")
	classNSSearchField = class("NSSearchField")
	classNSString = class("NSString")
}

// class returns a registered class, which every AppKit class is once
// the framework is loaded.
func class(name string) objc.Class {
	c := objc.GetClass(name)
	if c == 0 {
		panic("appkit: class not found: " + name)
	}
	return c
}

func nsString(s string) objc.ID {
	if s == "" {
		return objc.ID(classNSString).Send(sel("string"))
	}
	b := []byte(s)
	ref := cfStringCreateWithBytes(0, &b[0], int64(len(b)), utf8StringEncoding, false)
	return objc.ID(ref).Send(sel("autorelease"))
}

func goString(id objc.ID) string {
	if id == 0 {
		return ""
	}
	p := objc.Send[*byte](id, sel("UTF8String"))
	if p == nil {
		return ""
	}
	n := 0
	for *(*byte)(unsafe.Add(unsafe.Pointer(p), n)) != 0 {
		n++
	}
	return string(unsafe.Slice(p, n))
}

func unsafeSlice(p *byte, n uint) []byte {
	return unsafe.Slice(p, int(n))
}

func newView(name string) objc.ID {
	return objc.ID(class(name)).Send(sel("new"))
}

func setFrame(view objc.ID, x, y, width, height float64) {
	view.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: x, Y: y}, Size: CGSize{Width: width, Height: height}})
}

func bounds(view objc.ID) CGRect {
	return objc.Send[CGRect](view, sel("bounds"))
}

func systemFont(size float64) objc.ID {
	return objc.ID(classNSFont).Send(sel("systemFontOfSize:"), size)
}

// sendSuper calls cmd on the superclass of the class self was made
// from, even when AppKit has replaced self's class since.
func sendSuper[T any](self objc.ID, own objc.Class, cmd objc.SEL, args ...any) T {
	type objcSuper struct {
		_        structs.HostLayout
		receiver objc.ID
		class    objc.Class
	}
	var fn func(super *objcSuper, cmd objc.SEL, args ...any) T
	purego.RegisterFunc(&fn, msgSendSuper2)
	return fn(&objcSuper{receiver: self, class: own}, cmd, args...)
}
