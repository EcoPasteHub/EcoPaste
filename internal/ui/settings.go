package ui

import (
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/nsgo-dev/nsgo/core"
)

// SettingsModel reads and writes the persisted settings. Apply receives
// a native control's path and its new value.
type SettingsModel interface {
	Load() macos.SettingsValues
	Apply(path string, value any) error
}

// SettingsWindow builds the native settings window.
func SettingsWindow(model SettingsModel) func(*core.Scope) core.Node {
	return func(cx *core.Scope) core.Node {
		values := core.NewSignalFunc(cx, model.Load(), nil)
		errorText := core.NewSignal(cx, "")
		return core.Native{
			Component: macos.SettingsComponent,
			Style:     core.Style{Width: core.Pct(100), Height: core.Pct(100)},
			PropsFunc: func() map[string]any {
				return map[string]any{
					"values": values.Get(),
					"error":  errorText.Get(),
				}
			},
			OnEvent: func(name string, data any) {
				change, ok := data.(macos.SettingsChange)
				if name != "change" || !ok {
					return
				}
				go func() {
					err := model.Apply(change.Path, change.Value)
					cx.Post(func() {
						if err != nil {
							errorText.Set(err.Error())
							values.Set(model.Load())
							return
						}
						errorText.Set("")
						values.Set(model.Load())
					})
				}()
			},
		}
	}
}
