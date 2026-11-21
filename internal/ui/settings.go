package ui

import (
	"log/slog"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	kubai "github.com/nsgo-dev/nsgo-icons-kubai"
	"github.com/nsgo-dev/nsgo/core"
)

// SettingsModel reads and writes the persisted settings. Apply receives
// a native control's path and its new value.
type SettingsModel interface {
	Load() macos.SettingsValues
	Apply(path string, value any) error
	SuspendShortcuts(bool)
}

type settingsTab struct {
	id   string
	icon func() *kubai.Symbol
}

func settingsTabs() []settingsTab {
	return []settingsTab{
		{id: "general", icon: kubai.Settings},
		{id: "appearance", icon: kubai.Palette},
		{id: "clipboard", icon: kubai.Clipboard},
		{id: "interaction", icon: kubai.Mouse},
		{id: "data", icon: kubai.Database},
		{id: "system", icon: kubai.Sliders},
		{id: "about", icon: kubai.Info},
	}
}

// SettingsWindow builds the native settings window.
func SettingsWindow(model SettingsModel) func(*core.Scope) core.Node {
	nativeRef := &core.Ref{}
	return func(cx *core.Scope) core.Node {
		cx.Effect(func() {
			i18n.Watch(cx)
			cx.SetTitle(i18n.T(i18n.SettingsWindowTitle))
		})
		values := core.NewSignalFunc(cx, model.Load(), nil)
		errorText := core.NewSignal(cx, "")
		selected := core.NewSignal(cx, macos.InitialSettingsTab())
		return core.View{
			Style: core.Style{
				Width:     core.Pct(100),
				Height:    core.Pct(100),
				Direction: core.Column,
			},
			Children: []core.Node{
				settingsTabBar(cx, selected),
				core.Native{
					Component: macos.SettingsComponent,
					Style:     core.Style{FlexGrow: 1, Width: core.Pct(100)},
					Ref:       nativeRef,
					PropsFunc: func() map[string]any {
						return map[string]any{
							"language": string(i18n.Watch(cx)),
							"values":   values.Get(),
							"error":    errorText.Get(),
							"tab":      selected.Get(),
						}
					},
					OnEvent: func(name string, data any) {
						switch name {
						case "tabSelected":
							if tab, ok := data.(string); ok && tab != "" {
								selected.Set(tab)
							}
						case "suspendShortcuts":
							suspended, _ := data.(bool)
							model.SuspendShortcuts(suspended)
						case "change":
							change, ok := data.(macos.SettingsChange)
							if !ok {
								return
							}
							shortcutChange := isShortcutPath(change.Path)
							go func() {
								err := model.Apply(change.Path, change.Value)
								if err != nil {
									slog.Error("apply preference setting", "module", "ui.settings", "setting", change.Path, "error", err)
								}
								cx.Post(func() {
									values.Set(model.Load())
									if err != nil {
										errorText.Set(err.Error())
										if shortcutChange {
											nativeRef.Command("shortcutRejected", change.Path)
										}
										return
									}
									errorText.Set("")
									if shortcutChange {
										nativeRef.Command("shortcutAccepted", change)
										model.SuspendShortcuts(false)
									}
								})
							}()
						}
					},
				},
			},
		}
	}
}

func isShortcutPath(path string) bool {
	return path == "shortcuts.openClipboard" || path == "shortcuts.openPreference"
}

func settingsTabBar(cx *core.Scope, selected *core.Signal[string]) core.Node {
	dark := cx.Env().Dark()
	muted := core.RGB(0x6E6E73)
	divider := core.RGBA(0x000000, 0.08)
	if dark {
		muted = core.RGB(0x98989D)
		divider = core.RGBA(0xFFFFFF, 0.10)
	}
	tabs := settingsTabs()
	buttons := make([]core.Node, 0, len(tabs))
	for _, tab := range tabs {
		tab := tab
		title := func() string {
			i18n.Watch(cx)
			return macos.SettingsTabTitle(tab.id)
		}
		buttons = append(buttons, core.View{
			StyleFunc: func() core.Style {
				return settingsTabButtonStyle(selected.Get() == tab.id, dark)
			},
			Role:      core.RoleTab,
			LabelFunc: title,
			OnPress: func(core.PressEvent) {
				selected.Set(tab.id)
			},
			Children: []core.Node{
				settingsTabIcon(tab, selected, dark),
				core.Text{
					ContentFunc: title,
					StyleFunc: func() core.Style {
						return settingsTabTextStyle(selected.Get() == tab.id, muted, dark)
					},
				},
			},
		})
	}
	return core.View{
		Style: core.Style{
			Direction:      core.Row,
			Height:         core.Pt(80),
			Padding:        core.Symmetric(6, 4),
			Gap:            2,
			AlignItems:     core.AlignCenter,
			JustifyContent: core.JustifyCenter,
			Borders:        core.Edges{Bottom: 1},
			BorderColor:    divider,
		},
		Children: buttons,
	}
}

func settingsTabButtonStyle(active, dark bool) core.Style {
	return core.Style{
		Direction:      core.Column,
		AlignItems:     core.AlignCenter,
		JustifyContent: core.JustifyCenter,
		Width:          core.Pt(64),
		Height:         core.Pt(64),
		Gap:            2,
		CornerRadius:   13,
		Background:     settingsTabBackground(active, dark),
	}
}

func settingsTabTextStyle(active bool, muted core.Color, dark bool) core.Style {
	return core.Style{
		FontSize:   11,
		FontWeight: core.FontWeightMedium,
		Color:      settingsTabTextColor(active, muted, dark),
	}
}

func settingsTabIcon(tab settingsTab, selected *core.Signal[string], dark bool) core.Node {
	active := func() bool { return selected.Get() == tab.id }
	return core.View{
		Style: settingsTabIconBoxStyle(),
		Children: []core.Node{
			core.View{
				StyleFunc: func() core.Style {
					style := settingsTabIconLayerStyle()
					if active() {
						style.Opacity = core.Alpha(0)
					} else {
						style.Opacity = core.Alpha(1)
					}
					return style
				},
				Children: []core.Node{
					kubai.Icon{
						Symbol:  tab.icon(),
						Size:    22,
						Color:   settingsTabIconColor(false, dark),
						Hovered: active,
					},
				},
			},
			core.Show{
				When: active,
				Then: func(*core.Scope) core.Node {
					return core.View{
						Style: settingsTabIconLayerStyle(),
						Children: []core.Node{
							kubai.Icon{
								Symbol:  tab.icon(),
								Size:    22,
								Color:   settingsTabIconColor(true, dark),
								Hovered: func() bool { return true },
							},
						},
					}
				},
			},
		},
	}
}

func settingsTabIconBoxStyle() core.Style {
	return core.Style{
		Position: core.Relative,
		Width:    core.Pt(32),
		Height:   core.Pt(32),
	}
}

func settingsTabIconLayerStyle() core.Style {
	return core.Style{
		Position:       core.Absolute,
		Top:            core.Pt(0),
		Left:           core.Pt(0),
		Width:          core.Pt(32),
		Height:         core.Pt(32),
		AlignItems:     core.AlignCenter,
		JustifyContent: core.JustifyCenter,
	}
}

func settingsTabBackground(active, dark bool) core.Color {
	if active {
		alpha := float32(0.13)
		if dark {
			alpha = 0.20
		}
		return core.RGBA(0x5393FF, alpha)
	}
	return 0
}

func settingsTabIconColor(active, dark bool) core.Color {
	if active {
		if dark {
			return core.RGB(0x64A8FF)
		}
		return core.RGB(0x3478F6)
	}
	if dark {
		return core.RGB(0x98989D)
	}
	return core.RGB(0x8E8E93)
}

func settingsTabTextColor(active bool, muted core.Color, dark bool) core.Color {
	if active {
		if dark {
			return core.RGB(0x64A8FF)
		}
		return core.RGB(0x3478F6)
	}
	return muted
}
