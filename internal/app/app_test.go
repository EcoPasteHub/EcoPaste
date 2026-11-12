package app

import (
	"testing"

	"github.com/nsgo-dev/nsgo/core"

	"github.com/EcoPasteHub/EcoPaste/internal/macos/appkit"
)

func TestRootUsesOneAppKitNativeComponent(t *testing.T) {
	native := rootNative(0)
	if native.Component != appkit.ComponentName {
		t.Fatalf("component = %q, want %q", native.Component, appkit.ComponentName)
	}
	if native.Style.Width == (core.Length{}) || native.Style.Height == (core.Length{}) {
		t.Fatalf("root does not fill the window: %+v", native.Style)
	}
}
