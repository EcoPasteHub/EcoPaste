package ui

import (
	"context"
	"sync/atomic"
	"time"

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
			behavior = macos.PanelBehavior{DefaultFocus: search.DefaultFocus, ClearOnHide: search.ClearOnHide}
		}

		var (
			query         string
			filterValue   string
			selectedGroup string
			selectedID    string
			pinned        bool
			pageSize      = 80
			request       atomic.Int64
		)

		load := func(keyword, activeFilter, groupID string, appendRows bool) {
			if model == nil {
				errorText.Set("无法打开剪贴板数据库")
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

		copyItem := func(id string, forcePlain bool) {
			if id == "" || model == nil {
				return
			}
			ctx := cx.Context()
			go func() {
				copyAction := model.Copy
				if forcePlain {
					copyAction = model.CopyPlain
				}
				hideAfter, err := copyAction(ctx, id)
				cx.Post(func() {
					if err != nil {
						errorText.Set(err.Error())
						return
					}
					if hideAfterAction(pinned, hideAfter) && hide != nil {
						hide()
					}
					reload()
				})
			}()
		}

		activateItem := func(id string, action func(context.Context, string) (bool, bool, error)) {
			if id == "" || model == nil {
				return
			}
			ctx := cx.Context()
			go func() {
				paste, hideAfter, err := action(ctx, id)
				cx.Post(func() {
					if err != nil {
						errorText.Set(err.Error())
						loading.Set(false)
						if model.NeedsAccessibility(err) {
							ref.Command("showAccessibilityAlert", nil)
						}
						return
					}
					if hideAfterAction(pinned, hideAfter) && hide != nil {
						hide()
					}
					if paste {
						if pinned {
							ref.Command("resignKey", nil)
						}
						cx.AfterFunc(70*time.Millisecond, func() {
							if err := model.SendPaste(); err != nil {
								errorText.Set(err.Error())
								if model.NeedsAccessibility(err) {
									ref.Command("showAccessibilityAlert", nil)
								}
								return
							}
							if pinned {
								cx.AfterFunc(50*time.Millisecond, func() {
									ref.Command("makeKey", nil)
								})
							}
							reload()
						})
					} else {
						reload()
					}
				})
			}()
		}
		activate := func(id string, forcePlain bool) {
			if model == nil {
				return
			}
			action := model.Activate
			if forcePlain {
				action = model.ActivatePlain
			}
			activateItem(id, action)
		}
		activateSingleClick := func(id string) {
			if model == nil {
				return
			}
			activateItem(id, model.ActivateSingleClick)
		}
		activateDoubleClick := func(id string) {
			if model == nil {
				return
			}
			activateItem(id, model.ActivateDoubleClick)
		}

		middleClickItem := func(id string) {
			if id == "" || model == nil {
				return
			}
			ctx := cx.Context()
			go func() {
				hideAfter, err := model.MiddleClick(ctx, id)
				cx.Post(func() {
					if err != nil {
						errorText.Set(err.Error())
						loading.Set(false)
						if model.NeedsAccessibility(err) {
							ref.Command("showAccessibilityAlert", nil)
						}
						return
					}
					if hideAfterAction(pinned, hideAfter) && hide != nil {
						hide()
					}
					reload()
				})
			}()
		}

		applyAction := func(id string, run func(context.Context, string) error) {
			if id == "" || model == nil {
				return
			}
			ctx := cx.Context()
			go func() {
				err := run(ctx, id)
				cx.Post(func() {
					if err != nil {
						errorText.Set(err.Error())
						return
					}
					reload()
				})
			}()
		}

		runItemAction := func(id string, run func(context.Context, string) error) {
			if id == "" || model == nil {
				return
			}
			ctx := cx.Context()
			go func() {
				err := run(ctx, id)
				cx.Post(func() {
					if err != nil {
						errorText.Set(err.Error())
					}
				})
			}()
		}

		showPreview := func(id string) {
			if id == "" || model == nil {
				return
			}
			ctx := cx.Context()
			go func() {
				content, err := model.Preview(ctx, id)
				cx.Post(func() {
					if err != nil {
						errorText.Set(err.Error())
						return
					}
					if !content.Enabled {
						return
					}
					preview.Set(macos.PreviewContent{
						Active:    true,
						Title:     content.Title,
						Meta:      content.Meta,
						Text:      content.Text,
						ImagePath: content.ImagePath,
					})
				})
			}()
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

		load("", "", "", false)
		loadGroups("", false)
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
				}
			},
			OnEvent: func(name string, data any) {
				switch name {
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
					id := selectedID
					if value, ok := data.(string); ok && value != "" {
						id = value
					}
					activate(id, false)
				case "activateSingleClick":
					id := selectedID
					if value, ok := data.(string); ok && value != "" {
						id = value
					}
					activateSingleClick(id)
				case "activateDoubleClick":
					id := selectedID
					if value, ok := data.(string); ok && value != "" {
						id = value
					}
					activateDoubleClick(id)
				case "middleClick":
					id := selectedID
					if value, ok := data.(string); ok && value != "" {
						id = value
					}
					middleClickItem(id)
				case "copy":
					id := selectedID
					if value, ok := data.(string); ok && value != "" {
						id = value
					}
					copyItem(id, false)
				case "copyPlain":
					id := selectedID
					if value, ok := data.(string); ok && value != "" {
						id = value
					}
					if id == "" {
						return
					}
					copyItem(id, true)
				case "pastePlain":
					id := selectedID
					if value, ok := data.(string); ok && value != "" {
						id = value
					}
					if id == "" {
						return
					}
					activate(id, true)
				case "pastePath":
					id := selectedID
					if value, ok := data.(string); ok && value != "" {
						id = value
					}
					if id == "" {
						return
					}
					activate(id, true)
				case "toggleFavorite":
					applyAction(selectedID, func(ctx context.Context, id string) error {
						_, err := model.ToggleFavorite(ctx, id)
						return err
					})
				case "togglePinned":
					applyAction(selectedID, func(ctx context.Context, id string) error {
						_, err := model.TogglePinned(ctx, id)
						return err
					})
				case "delete":
					if selectedID == "" || model == nil {
						return
					}
					id := selectedID
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
					showPreview(selectedID)
				case "hidePreview":
					preview.Set(macos.PreviewContent{})
				case "open":
					runItemAction(selectedID, model.Open)
				case "reveal":
					runItemAction(selectedID, model.Reveal)
				case "saveImage":
					if selectedID == "" || model == nil {
						return
					}
					id := selectedID
					ctx := cx.Context()
					go func() {
						err := model.SaveImage(ctx, id)
						cx.Post(func() {
							if err != nil {
								errorText.Set(err.Error())
							}
						})
					}()
				case "editNote":
					if selectedID == "" || model == nil {
						return
					}
					id := selectedID
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
				case "moveGroup":
					if selectedID == "" || model == nil {
						return
					}
					value, _ := data.(string)
					ctx := cx.Context()
					go func(id, groupID string) {
						err := model.MoveToGroup(ctx, id, groupID)
						cx.Post(func() {
							if err != nil {
								errorText.Set(err.Error())
								return
							}
							reload()
						})
					}(selectedID, value)
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
