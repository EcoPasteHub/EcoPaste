//go:build darwin

package keystroke

import (
	"errors"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego"
)

// ErrPermission means macOS has not allowed EcoPaste to post keyboard
// events.
type permissionError struct{}

func (permissionError) Error() string { return i18n.T(i18n.ErrorPastePermission) }

var ErrPermission error = permissionError{}

const (
	keyV                = 0x09
	eventFlagCommand    = 1 << 20
	eventSourceHIDState = 1
	eventTapHID         = 0
)

var (
	cgEventCreateKeyboardEvent func(source uintptr, key uint16, down bool) uintptr
	cgEventSetFlags            func(event uintptr, flags uint64)
	cgEventPost                func(tap uint32, event uintptr)
	cfRelease                  func(value uintptr)
	axIsProcessTrusted         func() bool
)

func init() {
	coreGraphics, err := purego.Dlopen("/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics",
		purego.RTLD_GLOBAL|purego.RTLD_NOW)
	if err != nil {
		panic(err)
	}
	purego.RegisterLibFunc(&cgEventCreateKeyboardEvent, coreGraphics, "CGEventCreateKeyboardEvent")
	purego.RegisterLibFunc(&cgEventSetFlags, coreGraphics, "CGEventSetFlags")
	purego.RegisterLibFunc(&cgEventPost, coreGraphics, "CGEventPost")

	coreFoundation, err := purego.Dlopen("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation",
		purego.RTLD_GLOBAL|purego.RTLD_NOW)
	if err != nil {
		panic(err)
	}
	purego.RegisterLibFunc(&cfRelease, coreFoundation, "CFRelease")

	services, err := purego.Dlopen("/System/Library/Frameworks/ApplicationServices.framework/ApplicationServices",
		purego.RTLD_GLOBAL|purego.RTLD_NOW)
	if err != nil {
		panic(err)
	}
	purego.RegisterLibFunc(&axIsProcessTrusted, services, "AXIsProcessTrusted")
}

// Trusted reports whether the app may post input events.
func Trusted() bool { return axIsProcessTrusted() }

// SendPaste posts Command-V to the frontmost application.
func SendPaste() error {
	if !Trusted() {
		return ErrPermission
	}
	source := uintptr(0)
	down := cgEventCreateKeyboardEvent(source, keyV, true)
	if down == 0 {
		return errors.New(i18n.T(i18n.ErrorPasteEvent))
	}
	defer cfRelease(down)
	up := cgEventCreateKeyboardEvent(source, keyV, false)
	if up == 0 {
		return errors.New(i18n.T(i18n.ErrorPasteRelease))
	}
	defer cfRelease(up)
	cgEventSetFlags(down, eventFlagCommand)
	cgEventSetFlags(up, eventFlagCommand)
	cgEventPost(eventTapHID, down)
	cgEventPost(eventTapHID, up)
	return nil
}
