package app

import _ "embed"

// trayIconPNG is the menu bar icon of the pre-refactor app: the EcoPaste "p"
// mark, kept as the PNG that assets/tray-mac.ico held there. It is drawn as a
// template image, so the menu bar tints it like the system's own icons.
//
//go:embed assets/tray-mac.png
var trayIconPNG []byte

func trayIcon() []byte {
	return trayIconPNG
}
