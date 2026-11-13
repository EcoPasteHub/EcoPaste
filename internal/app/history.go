package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/EcoPasteHub/EcoPaste/internal/clipboard"
	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/keystroke"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
	"github.com/EcoPasteHub/EcoPaste/internal/ui"
)

type historyModel struct {
	runtime *clipboardRuntime
	store   *settings.Store
}

func newHistoryModel(runtime *clipboardRuntime, store *settings.Store) *historyModel {
	return &historyModel{runtime: runtime, store: store}
}

func (m *historyModel) List(ctx context.Context, q ui.HistoryQuery) (ui.HistoryPage, error) {
	cfg := m.store.Snapshot().Clipboard
	page, err := m.runtime.db.Page(ctx, database.Query{
		Keyword: q.Keyword,
		Sort:    database.Sort(cfg.Content.Sort),
		Limit:   q.Limit,
		Offset:  q.Offset,
	})
	if err != nil {
		return ui.HistoryPage{}, err
	}
	items := make([]ui.HistoryItem, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, m.historyItem(item))
	}
	return ui.HistoryPage{Items: items, Total: page.Total, HasMore: page.HasMore}, nil
}

func (m *historyModel) Copy(ctx context.Context, id string) (bool, error) {
	item, err := m.runtime.db.FindByID(ctx, id)
	if err != nil {
		return false, err
	}
	if item == nil {
		return false, fmt.Errorf("剪贴板记录不存在")
	}
	cfg := m.store.Snapshot().Clipboard
	plain := item.Kind == database.KindText && cfg.Content.CopyPlain
	if err := m.write(ctx, item, plain, cfg.Content.UpdateOnReuse); err != nil {
		return false, err
	}
	return cfg.Content.CopyThenHideWindow, nil
}

func (m *historyModel) Updates() <-chan struct{} { return m.runtime.changed }

func (m *historyModel) Activate(ctx context.Context, id string) (bool, bool, error) {
	item, err := m.runtime.db.FindByID(ctx, id)
	if err != nil {
		return false, false, err
	}
	if item == nil {
		return false, false, fmt.Errorf("剪贴板记录不存在")
	}
	cfg := m.store.Snapshot().Clipboard
	paste := autoPaste(cfg.Content.AutoPaste)
	if paste && !keystroke.Trusted() {
		return false, false, keystroke.ErrPermission
	}
	plain := item.Kind == database.KindText && (cfg.Content.CopyPlain || cfg.Content.PastePlain)
	if err := m.write(ctx, item, plain, cfg.Content.UpdateOnReuse); err != nil {
		return false, false, err
	}
	return paste, paste || cfg.Content.CopyThenHideWindow, nil
}

func (m *historyModel) SendPaste() error { return keystroke.SendPaste() }

func (m *historyModel) NeedsAccessibility(err error) bool {
	return errors.Is(err, keystroke.ErrPermission)
}

func (m *historyModel) write(ctx context.Context, item *database.Item, plain, countReuse bool) error {
	if err := m.runtime.service.WriteItem(ctx, item, plain); err != nil {
		return err
	}
	if countReuse {
		return m.runtime.db.IncrementUseCount(ctx, item.ID)
	}
	return nil
}

func autoPaste(action settings.AutoPaste) bool {
	return action == settings.AutoPasteSingleClickPaste || action == settings.AutoPasteDoubleClickPaste
}

func (m *historyModel) historyItem(item database.Item) ui.HistoryItem {
	row := ui.HistoryItem{
		ID:        item.ID,
		Kind:      string(item.Kind),
		CreatedAt: item.CreatedAt,
		Favorite:  item.IsFavorite,
		Pinned:    item.IsPinned,
	}
	if item.SourceAppName != nil {
		row.SourceApp = *item.SourceAppName
	}
	if item.Note != nil {
		row.Note = *item.Note
	}
	switch item.Kind {
	case database.KindText:
		if item.Summary != nil {
			row.Title = *item.Summary
		}
	case database.KindImage:
		if item.Width != nil && item.Height != nil {
			row.Title = fmt.Sprintf("图片 %d × %d", *item.Width, *item.Height)
		} else {
			row.Title = "图片"
		}
		row.ImagePath = m.runtime.images.OriginPath(item.Content)
	case database.KindFiles:
		row.Title = ui.FileTitle(clipboard.DecodeFiles(item.Content), 3)
	}
	return row
}
