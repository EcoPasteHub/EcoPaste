package ui

import (
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/nsgo-dev/nsgo/core"
)

// ClipboardPanel is the history panel's root: the native AppKit view
// fills the NSGo panel.
func ClipboardPanel(cx *core.Scope) core.Node {
	return core.Native{
		Component: macos.PanelComponent,
		Style:     core.Style{Width: core.Pct(100), Height: core.Pct(100)},
	}
}
