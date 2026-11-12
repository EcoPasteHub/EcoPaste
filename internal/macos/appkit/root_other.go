//go:build !darwin || ios

package appkit

import "github.com/nsgo-dev/nsgo/core"

type unsupportedRoot struct{}

// NewRoot returns a no-op component on platforms outside the first stage.
func NewRoot(*core.NativeContext) core.NativeComponent {
	return unsupportedRoot{}
}

func (unsupportedRoot) View() any           { return nil }
func (unsupportedRoot) SetProp(string, any) {}
func (unsupportedRoot) Command(string, any) {}
func (unsupportedRoot) Close()              {}
