package main

import (
	"github.com/EcoPasteHub/EcoPaste/internal/app"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/nsgo-dev/nsgo"
)

func main() {
	nsgo.Run(app.Run, nsgo.HideDock(), nsgo.Component(macos.PanelComponent, macos.NewPanel))
}
