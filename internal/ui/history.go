package ui

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/nsgo-dev/nsgo/core"
)

// ClipboardPanel builds the history panel. hide closes the panel after
// an action that the user configured to close it.
func ClipboardPanel(model HistoryModel, hide func()) func(*core.Scope) core.Node {
	return func(cx *core.Scope) core.Node {
		rows := core.NewSignalFunc(cx, []macos.HistoryRow(nil), nil)
		total := core.NewSignal(cx, int64(0))
		loading := core.NewSignal(cx, true)
		errorText := core.NewSignal(cx, "")
		hasMore := core.NewSignal(cx, false)
		preview := core.NewSignal(cx, macos.PreviewContent{})
		groups := core.NewSignalFunc(cx, []macos.GroupOption(nil), nil)
		filter := core.NewSignal(cx, "")
		group := core.NewSignal(cx, "")
		ref := &core.Ref{}
		behavior := macos.PanelBehavior{}
		if model != nil {
			search := model.SearchBehavior()
			preview := model.PreviewBehavior()
			behavior = macos.PanelBehavior{
				DefaultFocus: search.DefaultFocus,
				ClearOnHide:  search.ClearOnHide,
				SpacePreview: preview.SpaceEnabled,
			}
		}

		opening := HistoryOpenBehavior{Filter: "preserve", GroupID: "preserve"}
		if model != nil {
			opening = model.OpenBehavior()
		}
		initialFilter, initialGroup := openSelection(opening, "", "")
		var (
			query         string
			filterValue   = initialFilter
			selectedGroup = initialGroup
			selectedID    string
			pinned        bool
			scrollTop     atomic.Bool
			pageSize      = 80
			request       atomic.Int64
		)

		load := func(keyword, activeFilter, groupID string, appendRows bool) {
			if model == nil {
				errorText.Set(i18n.T(i18n.DatabaseOpenFailed))
				loading.Set(false)
				return
			}
			loading.Set(true)
			errorText.Set("")
			offset := 0
			if appendRows {
				offset = len(rows.GetUntracked())
			}
			serial := request.Add(1)
			ctx := cx.Context()
			go func(keyword, activeFilter, groupID string, offset int, appendRows bool, serial int64) {
				page, err := model.List(ctx, HistoryQuery{
					Keyword: keyword,
					Filter:  activeFilter,
					GroupID: groupID,
					Limit:   pageSize,
					Offset:  offset,
				})
				cx.Post(func() {
					if request.Load() != serial {
						return
					}
					if err != nil {
						errorText.Set(err.Error())
						loading.Set(false)
						return
					}
					next := make([]macos.HistoryRow, 0, len(page.Items)+offset)
					if appendRows {
						next = append(next, rows.GetUntracked()...)
					}
					for _, item := range page.Items {
						next = append(next, historyRow(item))
					}
					rows.Set(next)
					total.Set(page.Total)
					hasMore.Set(page.HasMore)
					loading.Set(false)
					if scrollTop.Load() {
						scrollTop.Store(false)
						ref.Command("scrollTop", nil)
					}
				})
			}(keyword, activeFilter, groupID, offset, appendRows, serial)
		}
		reload := func() { load(query, filterValue, selectedGroup, false) }

		loadGroups := func(selectID string, reloadRows bool) {
			if model == nil {
				return
			}
			ctx := cx.Context()
			go func() {
				nextGroups, err := model.Groups(ctx)
				cx.Post(func() {
					if err != nil {
						errorText.Set(err.Error())
						return
					}
					options := make([]macos.GroupOption, 0, len(nextGroups))
					foundSelected := selectID == ""
					for _, option := range nextGroups {
						options = append(options, macos.GroupOption{ID: option.ID, Name: option.Name})
						if option.ID == selectID {
							foundSelected = true
						}
					}
					groups.Set(options)
					if selectID != "" && foundSelected {
						selectedGroup = selectID
						filterValue = ""
						group.Set(selectedGroup)
						filter.Set("")
					} else if selectedGroup != "" && !groupExists(nextGroups, selectedGroup) {
						selectedGroup = ""
						group.Set("")
					}
					if reloadRows {
						reload()
					}
				})
			}()
		}

		actions := &itemActions{
			cx: cx, model: model, ref: ref,
			errorText: errorText, loading: loading, preview: preview,
			pinned: &pinned, hide: hide, reload: reload,
		}

		var pendingSearch func() bool
		cx.Cleanup(func() {
			if pendingSearch != nil {
				pendingSearch()
			}
		})

		var dragProvider func(string) (macos.DragPayload, bool)
		if model != nil {
			dragProvider = func(id string) (macos.DragPayload, bool) {
				payload, err := model.Drag(context.Background(), id)
				if err != nil {
					return macos.DragPayload{}, false
				}
				return payload, true
			}
		}

		actionID := func(data any) string {
			if value, ok := data.(string); ok && value != "" {
				return value
			}
			return selectedID
		}

		filter.Set(initialFilter)
		group.Set(initialGroup)
		load("", initialFilter, initialGroup, false)
		loadGroups(initialGroup, false)
		if model != nil {
			if updates := model.Updates(); updates != nil {
				ctx := cx.Context()
				go func() {
					for {
						select {
						case <-ctx.Done():
							return
						case <-updates:
							cx.Post(func() {
								preview.Set(macos.PreviewContent{})
								reload()
							})
						}
					}
				}()
			}
		}
		return core.Native{
			Component: macos.PanelComponent,
			Ref:       ref,
			Style:     core.Style{Width: core.Pct(100), Height: core.Pct(100)},
			PropsFunc: func() map[string]any {
				display := macos.PanelDisplay{TextMaxLines: 1, ImageMaxHeight: 32, FileMaxCount: 3}
				var actions []string
				if model != nil {
					density := model.Display()
					display = macos.PanelDisplay{
						TextMaxLines:   density.TextMaxLines,
						ImageMaxHeight: density.ImageMaxHeight,
						FileMaxCount:   density.FileMaxCount,
						ShowOriginal:   density.ShowOriginal,
					}
					actions = model.QuickActions()
				}
				return map[string]any{
					"rows":     rows.Get(),
					"total":    total.Get(),
					"loading":  loading.Get(),
					"error":    errorText.Get(),
					"preview":  preview.Get(),
					"groups":   groups.Get(),
					"group":    group.Get(),
					"filter":   filter.Get(),
					"query":    query,
					"behavior": behavior,
					"drag":     dragProvider,
					"display":  display,
					"actions":  actions,
				}
			},
			OnEvent: func(name string, data any) {
				switch name {
				case "openSettings":
					if model != nil {
						tab, _ := data.(string)
						model.OpenSettings(tab)
					}
				case "opened":
					if model == nil {
						return
					}
					opening := model.OpenBehavior()
					nextFilter, nextGroup := openSelection(opening, filterValue, selectedGroup)
					changed := nextFilter != filterValue || nextGroup != selectedGroup
					if nextFilter != filterValue {
						filterValue = nextFilter
						filter.Set(nextFilter)
					}
					if nextGroup != selectedGroup {
						selectedGroup = nextGroup
						group.Set(nextGroup)
					}
					if opening.ScrollTop {
						scrollTop.Store(true)
						changed = true
					}
					if changed {
						reload()
					}
				case "query":
					text, _ := data.(string)
					if pendingSearch != nil {
						pendingSearch()
					}
					pendingSearch = cx.AfterFunc(120*time.Millisecond, func() {
						query = text
						reload()
					})
				case "searchCleared":
					if pendingSearch != nil {
						pendingSearch()
						pendingSearch = nil
					}
					query = ""
					preview.Set(macos.PreviewContent{})
					reload()
				case "filter":
					value, _ := data.(string)
					filterValue = value
					selectedGroup = ""
					filter.Set(value)
					group.Set("")
					reload()
				case "group":
					value, _ := data.(string)
					selectedGroup = value
					filterValue = ""
					group.Set(value)
					filter.Set("")
					preview.Set(macos.PreviewContent{})
					reload()
				case "pinChanged":
					pinned, _ = data.(bool)
				case "select":
					selectedID, _ = data.(string)
				case "activate":
					actions.activate(actionID(data), false)
				case "activateSingleClick":
					actions.activateSingleClick(actionID(data))
				case "activateDoubleClick":
					actions.activateDoubleClick(actionID(data))
				case "middleClick":
					actions.middleClick(actionID(data))
				case "copy":
					actions.copy(actionID(data), false)
				case "copyPlain":
					actions.copy(actionID(data), true)
				case "pastePlain":
					actions.activate(actionID(data), true)
				case "pastePath":
					actions.activate(actionID(data), true)
				case "toggleFavorite":
					actions.run(actionID(data), true, func(ctx context.Context, id string) error {
						_, err := model.ToggleFavorite(ctx, id)
						return err
					})
				case "togglePinned":
					actions.run(actionID(data), true, func(ctx context.Context, id string) error {
						_, err := model.TogglePinned(ctx, id)
						return err
					})
				case "toggleWindowPinned":
					if ref != nil {
						ref.Command("toggleWindowPinned", nil)
					}
				case "delete":
					id := actionID(data)
					if id == "" || model == nil {
						return
					}
					ctx := cx.Context()
					go func() {
						err := model.Delete(ctx, id)
						cx.Post(func() {
							if err != nil {
								errorText.Set(err.Error())
								return
							}
							if selectedID == id {
								selectedID = ""
							}
							reload()
						})
					}()
				case "preview":
					if preview.GetUntracked().Active {
						preview.Set(macos.PreviewContent{})
						return
					}
					actions.showPreview(selectedID)
				case "hidePreview":
					preview.Set(macos.PreviewContent{})
				case "open":
					actions.run(actionID(data), false, model.Open)
				case "reveal":
					actions.run(actionID(data), false, model.Reveal)
				case "saveImage":
					actions.run(actionID(data), false, model.SaveImage)
				case "editNote":
					id := actionID(data)
					if id == "" || model == nil {
						return
					}
					ctx := cx.Context()
					go func() {
						saved, err := model.EditNote(ctx, id)
						cx.Post(func() {
							if err != nil {
								errorText.Set(err.Error())
								return
							}
							if saved {
								reload()
							}
						})
					}()
				case "newGroup":
					if model == nil {
						return
					}
					ctx := cx.Context()
					go func() {
						created, ok, err := model.CreateGroup(ctx)
						cx.Post(func() {
							if err != nil {
								errorText.Set(err.Error())
								return
							}
							if ok && created != nil {
								loadGroups(created.ID, true)
							}
						})
					}()
				case "renameGroup":
					if model == nil || selectedGroup == "" {
						return
					}
					ctx := cx.Context()
					go func(id string) {
						renamed, err := model.RenameGroup(ctx, id)
						cx.Post(func() {
							if err != nil {
								errorText.Set(err.Error())
								return
							}
							if renamed {
								loadGroups(id, false)
							}
						})
					}(selectedGroup)
				case "deleteGroup":
					if model == nil || selectedGroup == "" {
						return
					}
					ctx := cx.Context()
					go func(id string) {
						deleted, err := model.DeleteGroup(ctx, id)
						cx.Post(func() {
							if err != nil {
								errorText.Set(err.Error())
								return
							}
							if deleted {
								selectedGroup = ""
								group.Set("")
								loadGroups("", true)
							}
						})
					}(selectedGroup)
				case "clearHistory":
					if model == nil {
						return
					}
					ctx := cx.Context()
					go func() {
						err := model.ClearHistory(ctx)
						cx.Post(func() {
							if err != nil {
								errorText.Set(err.Error())
								return
							}
							selectedID = ""
							reload()
						})
					}()
				case "moveGroup":
					groupID, _ := data.(string)
					actions.run(selectedID, true, func(ctx context.Context, id string) error {
						return model.MoveToGroup(ctx, id, groupID)
					})
				case "loadMore":
					if hasMore.GetUntracked() && !loading.GetUntracked() {
						load(query, filterValue, selectedGroup, true)
					}
				case "hide":
					if hide != nil {
						hide()
					}
				case "blurHide":
					if !pinned && hide != nil {
						hide()
					}
				}
			},
		}
	}
}
