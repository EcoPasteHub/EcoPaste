package ui

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/nsgo-dev/nsgo/core"
)

type historyPanel struct {
	cx    *core.Scope
	model HistoryModel
	hide  func()
	ref   *core.Ref

	rows      *core.Signal[[]macos.HistoryRow]
	total     *core.Signal[int64]
	loading   *core.Signal[bool]
	errorText *core.Signal[string]
	hasMore   *core.Signal[bool]
	preview   *core.Signal[macos.PreviewContent]
	groups    *core.Signal[[]macos.GroupOption]
	filter    *core.Signal[string]
	group     *core.Signal[string]

	behavior    macos.PanelBehavior
	query       string
	filterValue string
	groupID     string
	selectedID  string
	pinned      bool
	scrollTop   atomic.Bool
	pageSize    int
	request     atomic.Int64

	pendingSearch func() bool
	actions       *itemActions
	dragProvider  func(string) (macos.DragPayload, bool)
}

func newHistoryPanel(cx *core.Scope, model HistoryModel, hide func()) *historyPanel {
	p := &historyPanel{
		cx: cx, model: model, hide: hide, ref: &core.Ref{},
		rows:      core.NewSignalFunc(cx, []macos.HistoryRow(nil), nil),
		total:     core.NewSignal(cx, int64(0)),
		loading:   core.NewSignal(cx, true),
		errorText: core.NewSignal(cx, ""),
		hasMore:   core.NewSignal(cx, false),
		preview:   core.NewSignal(cx, macos.PreviewContent{}),
		groups:    core.NewSignalFunc(cx, []macos.GroupOption(nil), nil),
		filter:    core.NewSignal(cx, ""),
		group:     core.NewSignal(cx, ""),
		pageSize:  80,
	}
	if model != nil {
		search := model.SearchBehavior()
		previewBehavior := model.PreviewBehavior()
		p.behavior = macos.PanelBehavior{
			DefaultFocus: search.DefaultFocus,
			ClearOnHide:  search.ClearOnHide,
			SpacePreview: previewBehavior.SpaceEnabled,
		}
	}

	opening := HistoryOpenBehavior{Filter: "preserve", GroupID: "preserve"}
	if model != nil {
		opening = model.OpenBehavior()
	}
	p.filterValue, p.groupID = openSelection(opening, "", "")
	p.actions = &itemActions{
		cx: cx, model: model, ref: p.ref,
		errorText: p.errorText, loading: p.loading, preview: p.preview,
		pinned: &p.pinned, hide: hide, reload: p.reload,
	}
	if model != nil {
		p.dragProvider = func(id string) (macos.DragPayload, bool) {
			payload, err := model.Drag(context.Background(), id)
			if err != nil {
				return macos.DragPayload{}, false
			}
			return payload, true
		}
	}
	cx.Cleanup(p.cancelPendingSearch)
	return p
}

func (p *historyPanel) start() {
	p.filter.Set(p.filterValue)
	p.group.Set(p.groupID)
	p.load("", p.filterValue, p.groupID, false)
	p.loadGroups(p.groupID, false)
	if p.model == nil {
		return
	}
	if updates := p.model.Updates(); updates != nil {
		ctx := p.cx.Context()
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-updates:
					p.cx.Post(func() {
						p.preview.Set(macos.PreviewContent{})
						p.reload()
					})
				}
			}
		}()
	}
}

func (p *historyPanel) node() core.Node {
	return core.Native{
		Component: macos.PanelComponent,
		Ref:       p.ref,
		Style:     core.Style{Width: core.Pct(100), Height: core.Pct(100)},
		PropsFunc: func() map[string]any {
			display := macos.PanelDisplay{TextMaxLines: 1, ImageMaxHeight: 32, FileMaxCount: 3}
			var quickActions []string
			if p.model != nil {
				density := p.model.Display()
				display = macos.PanelDisplay{
					TextMaxLines:   density.TextMaxLines,
					ImageMaxHeight: density.ImageMaxHeight,
					FileMaxCount:   density.FileMaxCount,
					ShowOriginal:   density.ShowOriginal,
				}
				quickActions = p.model.QuickActions()
			}
			return map[string]any{
				"rows":     p.rows.Get(),
				"total":    p.total.Get(),
				"loading":  p.loading.Get(),
				"error":    p.errorText.Get(),
				"preview":  p.preview.Get(),
				"groups":   p.groups.Get(),
				"group":    p.group.Get(),
				"filter":   p.filter.Get(),
				"query":    p.query,
				"behavior": p.behavior,
				"drag":     p.dragProvider,
				"display":  display,
				"actions":  quickActions,
			}
		},
		OnEvent: p.handleEvent,
	}
}

func (p *historyPanel) load(keyword, activeFilter, groupID string, appendRows bool) {
	if p.model == nil {
		p.errorText.Set(i18n.T(i18n.DatabaseOpenFailed))
		p.loading.Set(false)
		return
	}
	p.loading.Set(true)
	p.errorText.Set("")
	offset := 0
	if appendRows {
		offset = len(p.rows.GetUntracked())
	}
	serial := p.request.Add(1)
	ctx := p.cx.Context()
	go func() {
		page, err := p.model.List(ctx, HistoryQuery{
			Keyword: keyword,
			Filter:  activeFilter,
			GroupID: groupID,
			Limit:   p.pageSize,
			Offset:  offset,
		})
		p.cx.Post(func() {
			if p.request.Load() != serial {
				return
			}
			if err != nil {
				p.errorText.Set(err.Error())
				p.loading.Set(false)
				return
			}
			next := make([]macos.HistoryRow, 0, len(page.Items)+offset)
			if appendRows {
				next = append(next, p.rows.GetUntracked()...)
			}
			for _, item := range page.Items {
				next = append(next, historyRow(item))
			}
			p.rows.Set(next)
			p.total.Set(page.Total)
			p.hasMore.Set(page.HasMore)
			p.loading.Set(false)
			if p.scrollTop.Load() {
				p.scrollTop.Store(false)
				p.ref.Command("scrollTop", nil)
			}
		})
	}()
}

func (p *historyPanel) reload() {
	p.load(p.query, p.filterValue, p.groupID, false)
}

func (p *historyPanel) loadGroups(selectID string, reloadRows bool) {
	if p.model == nil {
		return
	}
	ctx := p.cx.Context()
	go func() {
		nextGroups, err := p.model.Groups(ctx)
		p.cx.Post(func() {
			if err != nil {
				p.errorText.Set(err.Error())
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
			p.groups.Set(options)
			if selectID != "" && foundSelected {
				p.groupID = selectID
				p.filterValue = ""
				p.group.Set(p.groupID)
				p.filter.Set("")
			} else if p.groupID != "" && !groupExists(nextGroups, p.groupID) {
				p.groupID = ""
				p.group.Set("")
			}
			if reloadRows {
				p.reload()
			}
		})
	}()
}

func (p *historyPanel) cancelPendingSearch() {
	if p.pendingSearch != nil {
		p.pendingSearch()
		p.pendingSearch = nil
	}
}

func (p *historyPanel) actionID(data any) string {
	if value, ok := data.(string); ok && value != "" {
		return value
	}
	return p.selectedID
}

func (p *historyPanel) handleEvent(name string, data any) {
	switch {
	case p.handleListEvent(name, data):
	case p.handleWindowEvent(name, data):
	case p.handleCopyEvent(name, data):
	case p.handleActivationEvent(name, data):
	case p.handleRowEvent(name, data):
	case p.handleGroupEvent(name, data):
	}
}

func (p *historyPanel) handleListEvent(name string, data any) bool {
	switch name {
	case "opened":
		if p.model == nil {
			return true
		}
		opening := p.model.OpenBehavior()
		nextFilter, nextGroup := openSelection(opening, p.filterValue, p.groupID)
		changed := nextFilter != p.filterValue || nextGroup != p.groupID
		if nextFilter != p.filterValue {
			p.filterValue = nextFilter
			p.filter.Set(nextFilter)
		}
		if nextGroup != p.groupID {
			p.groupID = nextGroup
			p.group.Set(nextGroup)
		}
		if opening.ScrollTop {
			p.scrollTop.Store(true)
			changed = true
		}
		if changed {
			p.reload()
		}
	case "query":
		text, _ := data.(string)
		p.cancelPendingSearch()
		p.pendingSearch = p.cx.AfterFunc(120*time.Millisecond, func() {
			p.query = text
			p.reload()
		})
	case "searchCleared":
		p.cancelPendingSearch()
		p.query = ""
		p.preview.Set(macos.PreviewContent{})
		p.reload()
	case "filter":
		value, _ := data.(string)
		p.filterValue = value
		p.groupID = ""
		p.filter.Set(value)
		p.group.Set("")
		p.reload()
	case "group":
		value, _ := data.(string)
		p.groupID = value
		p.filterValue = ""
		p.group.Set(value)
		p.filter.Set("")
		p.preview.Set(macos.PreviewContent{})
		p.reload()
	case "select":
		p.selectedID, _ = data.(string)
	case "loadMore":
		if p.hasMore.GetUntracked() && !p.loading.GetUntracked() {
			p.load(p.query, p.filterValue, p.groupID, true)
		}
	default:
		return false
	}
	return true
}

func (p *historyPanel) handleWindowEvent(name string, data any) bool {
	switch name {
	case "openSettings":
		if p.model != nil {
			tab, _ := data.(string)
			p.model.OpenSettings(tab)
		}
	case "pinChanged":
		p.pinned, _ = data.(bool)
	case "toggleWindowPinned":
		if p.ref != nil {
			p.ref.Command("toggleWindowPinned", nil)
		}
	case "hide":
		if p.hide != nil {
			p.hide()
		}
	case "blurHide":
		if !p.pinned && p.hide != nil {
			p.hide()
		}
	default:
		return false
	}
	return true
}

func (p *historyPanel) handleCopyEvent(name string, data any) bool {
	switch name {
	case "copy":
		p.actions.copy(p.actionID(data), false)
	case "copyPlain":
		p.actions.copy(p.actionID(data), true)
	case "pastePlain", "pastePath":
		p.actions.activate(p.actionID(data), true)
	default:
		return false
	}
	return true
}

func (p *historyPanel) handleActivationEvent(name string, data any) bool {
	switch name {
	case "activate":
		p.actions.activate(p.actionID(data), false)
	case "activateSingleClick":
		p.actions.activateSingleClick(p.actionID(data))
	case "activateDoubleClick":
		p.actions.activateDoubleClick(p.actionID(data))
	case "middleClick":
		p.actions.middleClick(p.actionID(data))
	default:
		return false
	}
	return true
}

func (p *historyPanel) handleRowEvent(name string, data any) bool {
	switch name {
	case "toggleFavorite":
		p.actions.run(p.actionID(data), true, func(ctx context.Context, id string) error {
			_, err := p.model.ToggleFavorite(ctx, id)
			return err
		})
	case "togglePinned":
		p.actions.run(p.actionID(data), true, func(ctx context.Context, id string) error {
			_, err := p.model.TogglePinned(ctx, id)
			return err
		})
	case "delete":
		id := p.actionID(data)
		if id == "" || p.model == nil {
			return true
		}
		ctx := p.cx.Context()
		go func() {
			err := p.model.Delete(ctx, id)
			p.cx.Post(func() {
				if err != nil {
					p.errorText.Set(err.Error())
					return
				}
				if p.selectedID == id {
					p.selectedID = ""
				}
				p.reload()
			})
		}()
	case "preview":
		if p.preview.GetUntracked().Active {
			p.preview.Set(macos.PreviewContent{})
			return true
		}
		p.actions.showPreview(p.selectedID)
	case "hidePreview":
		p.preview.Set(macos.PreviewContent{})
	case "open":
		if p.model != nil {
			p.actions.run(p.actionID(data), false, p.model.Open)
		}
	case "reveal":
		if p.model != nil {
			p.actions.run(p.actionID(data), false, p.model.Reveal)
		}
	case "saveImage":
		if p.model != nil {
			p.actions.run(p.actionID(data), false, p.model.SaveImage)
		}
	case "editNote":
		id := p.actionID(data)
		if id == "" || p.model == nil {
			return true
		}
		ctx := p.cx.Context()
		go func() {
			saved, err := p.model.EditNote(ctx, id)
			p.cx.Post(func() {
				if err != nil {
					p.errorText.Set(err.Error())
					return
				}
				if saved {
					p.reload()
				}
			})
		}()
	case "moveGroup":
		groupID, _ := data.(string)
		p.actions.run(p.selectedID, true, func(ctx context.Context, id string) error {
			return p.model.MoveToGroup(ctx, id, groupID)
		})
	default:
		return false
	}
	return true
}

func (p *historyPanel) handleGroupEvent(name string, data any) bool {
	switch name {
	case "newGroup":
		if p.model == nil {
			return true
		}
		ctx := p.cx.Context()
		go func() {
			created, ok, err := p.model.CreateGroup(ctx)
			p.cx.Post(func() {
				if err != nil {
					p.errorText.Set(err.Error())
					return
				}
				if ok && created != nil {
					p.loadGroups(created.ID, true)
				}
			})
		}()
	case "renameGroup":
		if p.model == nil || p.groupID == "" {
			return true
		}
		ctx := p.cx.Context()
		go func(id string) {
			renamed, err := p.model.RenameGroup(ctx, id)
			p.cx.Post(func() {
				if err != nil {
					p.errorText.Set(err.Error())
					return
				}
				if renamed {
					p.loadGroups(id, false)
				}
			})
		}(p.groupID)
	case "deleteGroup":
		if p.model == nil || p.groupID == "" {
			return true
		}
		ctx := p.cx.Context()
		go func(id string) {
			deleted, err := p.model.DeleteGroup(ctx, id)
			p.cx.Post(func() {
				if err != nil {
					p.errorText.Set(err.Error())
					return
				}
				if deleted {
					p.groupID = ""
					p.group.Set("")
					p.loadGroups("", true)
				}
			})
		}(p.groupID)
	case "clearHistory":
		if p.model == nil {
			return true
		}
		ctx := p.cx.Context()
		go func() {
			err := p.model.ClearHistory(ctx)
			p.cx.Post(func() {
				if err != nil {
					p.errorText.Set(err.Error())
					return
				}
				p.selectedID = ""
				p.reload()
			})
		}()
	default:
		return false
	}
	return true
}
