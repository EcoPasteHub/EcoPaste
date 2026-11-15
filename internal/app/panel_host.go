package app

import (
	"log"

	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
	"github.com/EcoPasteHub/EcoPaste/internal/storage"
	"github.com/EcoPasteHub/EcoPaste/internal/ui"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
)

// panelHost owns the clipboard panel window and its persisted frame.
type panelHost struct {
	app         *core.Scope
	store       *settings.Store
	windowState *storage.WindowState
	runtime     func() *clipboardRuntime
	content     func(*core.Scope) core.Node
	window      *desktop.Window
}

func newPanelHost(app *core.Scope, store *settings.Store, windowState *storage.WindowState,
	runtime func() *clipboardRuntime, history ui.HistoryModel) *panelHost {
	host := &panelHost{
		app:         app,
		store:       store,
		windowState: windowState,
		runtime:     runtime,
	}
	host.content = ui.ClipboardPanel(history, host.Hide)
	host.open()
	return host
}

func (h *panelHost) open() {
	h.window = desktop.OpenPanel(h.app, desktop.PanelOptions{
		Width:    420,
		Height:   560,
		Material: desktop.WindowMaterialSidebar,
	}, h.content)
}

func (h *panelHost) Rebuild() {
	if h.window != nil && !h.window.Closed() {
		if h.window.Visible() {
			h.saveFrame()
		}
		h.window.Close()
	}
	h.open()
}

func (h *panelHost) Show() {
	h.signal()
	h.place()
	h.window.Show()
}

func (h *panelHost) Hide() {
	h.saveFrame()
	h.window.Hide()
	// Window commands issued from a Carbon callback need a tree update
	// before NSGo applies them.
	h.signal()
}

func (h *panelHost) Toggle() {
	if h.window.Visible() {
		h.Hide()
		return
	}
	h.Show()
}

func (h *panelHost) Window() *desktop.Window {
	return h.window
}

func (h *panelHost) signal() {
	if runtime := h.runtime(); runtime != nil {
		runtime.Signal()
	}
}

func (h *panelHost) place() {
	mode := string(settings.Default().Clipboard.Window.Position)
	if h.store != nil {
		mode = string(h.store.Snapshot().Clipboard.Window.Position)
	}
	placement := macos.PanelPlacement{Mode: mode}
	if h.windowState != nil {
		if frame, ok := h.windowState.Frame(panelStateLabel); ok {
			placement.Frame = macos.PanelFrame{
				X: float64(frame.X), Y: float64(frame.Y),
				Width: float64(frame.Width), Height: float64(frame.Height),
			}
			placement.HasFrame = true
		}
	}
	macos.PlacePanelOnNextShow(placement)
}

func (h *panelHost) saveFrame() {
	if h.windowState == nil {
		return
	}
	frame, ok := macos.PanelFrameOf()
	if !ok {
		return
	}
	err := h.windowState.SaveFrame(panelStateLabel, storage.WindowFrame{
		X: int64(frame.X), Y: int64(frame.Y),
		Width: int64(frame.Width), Height: int64(frame.Height),
	})
	if err != nil {
		log.Printf("save window state: %v", err)
	}
}
