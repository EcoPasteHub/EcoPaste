package app

import (
	"github.com/nsgo-dev/nsgo"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"

	"github.com/EcoPasteHub/EcoPaste/internal/macos/appkit"
)

const (
	windowKey    = "clipboard"
	windowTitle  = "EcoPaste"
	windowWidth  = 420
	windowHeight = 640
)

// Main starts the macOS application shell.
func Main() {
	nsgo.Run(
		Run,
		nsgo.Component(appkit.ComponentName, appkit.NewRoot),
		nsgo.WindowSize(windowWidth, windowHeight),
	)
}

// Run opens the clipboard window used by the AppKit application shell.
func Run(app *core.Scope) {
	desktop.SetShutdownMode(app, desktop.ShutdownOnMainWindowClose)

	window := desktop.OpenWindow(app, desktop.WindowOptions{
		Title:  windowTitle,
		Width:  windowWidth,
		Height: windowHeight,
		Key:    windowKey,
		Constraints: desktop.WindowConstraints{
			MinWidth:   360,
			MinHeight:  480,
			NoMaximize: true,
		},
	}, Root)

	desktop.SetMainWindow(app, window)
}

// Root returns the single AppKit native root that owns the window content.
func Root(cx *core.Scope) core.Node {
	clicks := core.NewSignal(cx, 0)
	native := rootNative(0)
	native.PropsFunc = func() map[string]any {
		return map[string]any{
			"clickCount": clicks.Get(),
		}
	}
	native.OnEvent = func(name string, _ any) {
		if name != appkit.EventPrimaryAction {
			return
		}
		clicks.Update(func(value int) int {
			return value + 1
		})
	}
	return native
}

func rootNative(clickCount int) core.Native {
	return core.Native{
		Component: appkit.ComponentName,
		Style: core.Style{
			FlexGrow: 1,
			Width:    core.Pct(100),
			Height:   core.Pct(100),
		},
		PropsFunc: func() map[string]any {
			return map[string]any{
				"clickCount": clickCount,
			}
		},
	}
}
