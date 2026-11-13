package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/clipboard"
	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/EcoPasteHub/EcoPaste/internal/keystroke"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/EcoPasteHub/EcoPaste/internal/onboarding"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
	"github.com/EcoPasteHub/EcoPaste/internal/ui"
)

type historyModel struct {
	runtime      *clipboardRuntime
	store        *settings.Store
	openSettings func(string)
	legacy       *onboarding.LegacyImporter
}

var inputTrusted = keystroke.Trusted

func newHistoryModel(runtime *clipboardRuntime, store *settings.Store) *historyModel {
	return &historyModel{
		runtime: runtime, store: store,
		legacy: &onboarding.LegacyImporter{
			DB: runtime.db, Images: runtime.images,
			Settings: func() settings.Clipboard { return store.Snapshot().Clipboard },
		},
	}
}

func (m *historyModel) List(ctx context.Context, q ui.HistoryQuery) (ui.HistoryPage, error) {
	cfg := m.store.Snapshot().Clipboard
	var groupID *string
	if q.GroupID != "" {
		groupID = &q.GroupID
	}
	page, err := m.runtime.db.Page(ctx, database.Query{
		Keyword: q.Keyword,
		Group:   q.Filter,
		GroupID: groupID,
		Sort:    database.Sort(cfg.Content.Sort),
		Limit:   q.Limit,
		Offset:  q.Offset,
	})
	if err != nil {
		return ui.HistoryPage{}, err
	}
	items := make([]ui.HistoryItem, 0, len(page.Items))
	for _, item := range page.Items {
		row := m.historyItem(ctx, item)
		row.CanDelete, row.DeleteConfirm = m.deletePolicy(item, q)
		items = append(items, row)
	}
	return ui.HistoryPage{Items: items, Total: page.Total, HasMore: page.HasMore}, nil
}

// deletePolicy applies the published per-row delete rules: favorites and
// pinned rows can be protected, and each category has its own confirm
// switch.
func (m *historyModel) deletePolicy(item database.Item, q ui.HistoryQuery) (bool, bool) {
	content := settings.Default().Clipboard.Content
	if m.store != nil {
		content = m.store.Snapshot().Clipboard.Content
	}
	canDelete := true
	if item.IsFavorite && !content.DeleteFavoriteItems {
		canDelete = false
	}
	if item.IsFavorite && content.DeleteFavoriteItemsOnlyInFavGroup && q.Filter != "favorite" {
		canDelete = false
	}
	if item.IsPinned && !content.DeletePinnedItems {
		canDelete = false
	}
	confirm := content.DeleteConfirm
	if item.IsPinned {
		confirm = content.DeletePinnedConfirm
	} else if item.IsFavorite {
		confirm = content.DeleteFavoriteConfirm
	}
	return canDelete, confirm
}

func (m *historyModel) Copy(ctx context.Context, id string) (bool, error) {
	return m.copyItem(ctx, id, false)
}

func (m *historyModel) CopyPlain(ctx context.Context, id string) (bool, error) {
	return m.copyItem(ctx, id, true)
}

func (m *historyModel) copyItem(ctx context.Context, id string, forcePlain bool) (bool, error) {
	item, err := m.runtime.db.FindByID(ctx, id)
	if err != nil {
		return false, err
	}
	if item == nil {
		return false, fmt.Errorf("剪贴板记录不存在")
	}
	cfg := m.store.Snapshot().Clipboard
	plain := forcePlain || item.Kind == database.KindText && cfg.Content.CopyPlain
	if err := m.write(ctx, item, plain, cfg.Content.UpdateOnReuse); err != nil {
		return false, err
	}
	return cfg.Content.CopyThenHideWindow, nil
}

func (m *historyModel) Updates() <-chan struct{} { return m.runtime.changed }

func (m *historyModel) SearchBehavior() ui.HistorySearchBehavior {
	search := m.store.Snapshot().Clipboard.Search
	return ui.HistorySearchBehavior{DefaultFocus: search.DefaultFocus, ClearOnHide: search.ClearOnHide}
}

func (m *historyModel) PreviewBehavior() ui.HistoryPreviewBehavior {
	preview := settings.Default().Clipboard.Preview
	if m.store != nil {
		preview = m.store.Snapshot().Clipboard.Preview
	}
	return ui.HistoryPreviewBehavior{SpaceEnabled: preview.SpaceEnabled}
}

// Display reports the clamped list density, so a hand-edited settings
// file cannot produce rows without text.
func (m *historyModel) Display() ui.HistoryDisplay {
	density := settings.Default().Clipboard.Display
	content := settings.Default().Clipboard.Content
	if m.store != nil {
		clipboard := m.store.Snapshot().Clipboard
		density = clipboard.Display
		content = clipboard.Content
	}
	return ui.HistoryDisplay{
		TextMaxLines:   clampInt(int(density.TextMaxLines), 1, 5),
		ImageMaxHeight: clampInt(int(density.ImageMaxHeight), 20, 100),
		FileMaxCount:   clampInt(int(density.FileMaxCount), 1, 5),
		ShowOriginal:   content.ShowOriginalPreview,
	}
}

// OpenBehavior resolves the configured panel-open selections into the
// list's initial filter and group.
func (m *historyModel) OpenBehavior() ui.HistoryOpenBehavior {
	window := settings.Default().Clipboard.Window
	if m.store != nil {
		window = m.store.Snapshot().Clipboard.Window
	}
	filter := "preserve"
	switch window.SelectCategoryOnOpen {
	case settings.CategoryAll:
		if window.SelectRangeOnOpen == settings.RangeFavorite {
			filter = "favorite"
		} else {
			filter = ""
		}
	case settings.CategoryText:
		filter = "text"
	case settings.CategoryImage:
		filter = "image"
	case settings.CategoryFiles:
		filter = "files"
	case settings.CategoryPreserve:
		switch window.SelectRangeOnOpen {
		case settings.RangeAll:
			filter = ""
		case settings.RangeFavorite:
			filter = "favorite"
		}
	}
	group := "preserve"
	switch {
	case window.SelectGroupOnOpen == "all":
		group = ""
	case strings.HasPrefix(window.SelectGroupOnOpen, "group:"):
		group = strings.TrimPrefix(window.SelectGroupOnOpen, "group:")
	}
	return ui.HistoryOpenBehavior{
		ScrollTop: window.ScrollToTopOnOpen,
		Filter:    filter,
		GroupID:   group,
	}
}

// QuickActions reports the hover actions in their configured order.
func (m *historyModel) QuickActions() []string {
	content := settings.Default().Clipboard.Content
	if m.store != nil {
		content = m.store.Snapshot().Clipboard.Content
	}
	enabled := make(map[settings.ItemAction]struct{}, len(content.ItemActions))
	for _, action := range content.ItemActions {
		enabled[action] = struct{}{}
	}
	order := content.ItemActionOrder
	if len(order) == 0 {
		order = settings.Default().Clipboard.Content.ItemActionOrder
	}
	actions := make([]string, 0, len(order))
	for _, action := range order {
		if _, ok := enabled[action]; ok {
			actions = append(actions, string(action))
		}
	}
	return actions
}

// Onboarding reports the first-run flow state persisted in settings.
func (m *historyModel) Onboarding() ui.HistoryOnboarding {
	data := ui.HistoryOnboarding{}
	if m.store == nil {
		return data
	}
	snapshot := m.store.Snapshot()
	step := int(snapshot.Onboarding.LastStep)
	if step < 0 {
		step = 0
	}
	if step > 5 {
		step = 5
	}
	data.Active = !snapshot.Onboarding.Completed
	data.Step = step
	data.OpenShortcut = snapshot.Shortcuts.OpenClipboard
	data.PreferenceShortcut = snapshot.Shortcuts.OpenPreference
	data.ExcludedApps = len(snapshot.Clipboard.Filters.ExcludedAppIDs)
	return data
}

// SaveOnboarding records the current step, or completes the flow.
func (m *historyModel) SaveOnboarding(step int, completed bool) error {
	if m.store == nil {
		return nil
	}
	if step < 0 {
		step = 0
	}
	if step > 5 {
		step = 5
	}
	patch := map[string]any{"lastStep": step}
	if completed {
		patch["completed"] = true
	}
	_, err := m.store.Update(map[string]any{"onboarding": patch})
	return err
}

func (m *historyModel) OpenSettings(tab string) {
	if m.openSettings != nil {
		m.openSettings(tab)
	}
}

// DetectLegacy scans the v0.6 data directory and records that the import
// step ran, matching the old onboarding behavior.
func (m *historyModel) DetectLegacy(ctx context.Context) (ui.LegacyDetection, error) {
	if m.legacy == nil {
		return ui.LegacyDetection{}, fmt.Errorf("剪贴板服务未启动")
	}
	detection, err := m.legacy.Detect(ctx)
	if err != nil {
		return ui.LegacyDetection{}, err
	}
	if _, err := m.store.Update(map[string]any{
		"onboarding": map[string]any{"legacyImport": map[string]any{"checked": true}},
	}); err != nil {
		return ui.LegacyDetection{}, err
	}
	return ui.LegacyDetection{
		Found:               detection.Found,
		Path:                detection.Path,
		ImportableItemCount: detection.ImportableItemCount,
		NormalItemCount:     detection.NormalItemCount,
		FavoriteItemCount:   detection.FavoriteItemCount,
	}, nil
}

// ImportLegacy writes the selected v0.6 rows and records the result so the
// onboarding step never offers the same import again.
func (m *historyModel) ImportLegacy(ctx context.Context, types []string) (ui.LegacyImportResult, error) {
	if m.legacy == nil {
		return ui.LegacyImportResult{}, fmt.Errorf("剪贴板服务未启动")
	}
	selections := make([]onboarding.LegacyImportSelection, 0, len(types))
	for _, item := range types {
		selections = append(selections, onboarding.LegacyImportSelection(item))
	}
	result, err := m.legacy.Import(ctx, selections)
	if err != nil {
		return ui.LegacyImportResult{}, err
	}
	importTypes := make([]string, 0, len(result.SelectedTypes))
	for _, item := range result.SelectedTypes {
		importTypes = append(importTypes, string(item))
	}
	importedAt := result.ImportedAt.Format(time.RFC3339)
	if _, err := m.store.Update(map[string]any{
		"onboarding": map[string]any{"legacyImport": map[string]any{
			"checked":     true,
			"imported":    result.Imported > 0,
			"importTypes": importTypes,
			"importedAt":  importedAt,
		}},
	}); err != nil {
		return ui.LegacyImportResult{}, err
	}
	m.runtime.Signal()
	return ui.LegacyImportResult{
		Imported:         result.Imported,
		Skipped:          result.Skipped,
		ImportedNormal:   result.ImportedNormal,
		ImportedFavorite: result.ImportedFavorite,
	}, nil
}

func clampInt(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func (m *historyModel) Activate(ctx context.Context, id string) (bool, bool, error) {
	return m.activateItem(ctx, id, false, true)
}

func (m *historyModel) ActivatePlain(ctx context.Context, id string) (bool, bool, error) {
	return m.activateItem(ctx, id, true, true)
}

func (m *historyModel) ActivateSingleClick(ctx context.Context, id string) (bool, bool, error) {
	return m.activateByMode(ctx, id, settings.AutoPasteSingleClickPaste, settings.AutoPasteSingleClickCopy)
}

func (m *historyModel) ActivateDoubleClick(ctx context.Context, id string) (bool, bool, error) {
	return m.activateByMode(ctx, id, settings.AutoPasteDoubleClickPaste, settings.AutoPasteDoubleClickCopy)
}

func (m *historyModel) activateByMode(ctx context.Context, id string, pasteMode, copyMode settings.AutoPaste) (bool, bool, error) {
	mode := m.store.Snapshot().Clipboard.Content.AutoPaste
	switch mode {
	case pasteMode:
		return m.activateItem(ctx, id, false, true)
	case copyMode:
		return m.activateItem(ctx, id, false, false)
	}
	return false, false, nil
}

func (m *historyModel) activateItem(ctx context.Context, id string, forcePlain, paste bool) (bool, bool, error) {
	item, err := m.runtime.db.FindByID(ctx, id)
	if err != nil {
		return false, false, err
	}
	if item == nil {
		return false, false, fmt.Errorf("剪贴板记录不存在")
	}
	cfg := m.store.Snapshot().Clipboard
	if paste && !inputTrusted() {
		return false, false, keystroke.ErrPermission
	}
	plain := forcePlain ||
		item.Kind == database.KindText && (cfg.Content.CopyPlain || cfg.Content.PastePlain) ||
		item.Kind == database.KindFiles && cfg.Content.PasteFilesAsPath
	if err := m.write(ctx, item, plain, cfg.Content.UpdateOnReuse); err != nil {
		return false, false, err
	}
	return paste, paste || cfg.Content.CopyThenHideWindow, nil
}

func (m *historyModel) SendPaste() error { return keystroke.SendPaste() }

// Drag builds the pasteboard representation a row offers when it is
// dragged out of the panel.
func (m *historyModel) Drag(ctx context.Context, id string) (macos.DragPayload, error) {
	item, err := m.runtime.db.FindByID(ctx, id)
	if err != nil {
		return macos.DragPayload{}, err
	}
	if item == nil {
		return macos.DragPayload{}, fmt.Errorf("剪贴板记录不存在")
	}
	payload, err := m.runtime.service.BuildWrite(item, false)
	if err != nil {
		return macos.DragPayload{}, err
	}
	return macos.DragPayload{
		Text:      payload.Text,
		HTML:      payload.HTML,
		RTF:       payload.RTF,
		ImageData: payload.PNG,
		Files:     payload.Files,
	}, nil
}

// MiddleClick runs the configured middle-button action for one row.
func (m *historyModel) MiddleClick(ctx context.Context, id string) (bool, error) {
	item, err := m.runtime.db.FindByID(ctx, id)
	if err != nil {
		return false, err
	}
	if item == nil {
		return false, fmt.Errorf("剪贴板记录不存在")
	}
	cfg := m.store.Snapshot().Clipboard
	plain := false
	paste := false
	switch cfg.Content.MiddleClick {
	case settings.MiddleClickSingleClickPaste:
		paste = true
	case settings.MiddleClickSingleClickPlain:
		plain, paste = true, true
	case settings.MiddleClickSingleClickCopy:
	case settings.MiddleClickSingleClickCopyPl:
		plain = true
	default:
		return false, nil
	}
	if paste && !keystroke.Trusted() {
		return false, keystroke.ErrPermission
	}
	if err := m.write(ctx, item, plain, cfg.Content.UpdateOnReuse); err != nil {
		return false, err
	}
	return paste || cfg.Content.CopyThenHideWindow, nil
}

func (m *historyModel) NeedsAccessibility(err error) bool {
	return errors.Is(err, keystroke.ErrPermission)
}

func (m *historyModel) ToggleFavorite(ctx context.Context, id string) (bool, error) {
	return m.runtime.db.ToggleFavorite(ctx, id)
}

func (m *historyModel) TogglePinned(ctx context.Context, id string) (bool, error) {
	return m.runtime.db.TogglePinned(ctx, id)
}

func (m *historyModel) Delete(ctx context.Context, id string) error {
	item, err := m.runtime.db.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if item == nil {
		return nil
	}
	content := settings.Default().Clipboard.Content
	if m.store != nil {
		content = m.store.Snapshot().Clipboard.Content
	}
	if item.IsFavorite && !content.DeleteFavoriteItems {
		return fmt.Errorf("收藏记录不允许删除")
	}
	if item.IsPinned && !content.DeletePinnedItems {
		return fmt.Errorf("置顶记录不允许删除")
	}
	if err := m.runtime.db.Delete(ctx, id); err != nil {
		return err
	}
	if item.Kind == database.KindImage {
		if err := m.runtime.images.Remove(item.Content); err != nil {
			return fmt.Errorf("删除图片文件: %w", err)
		}
	}
	return nil
}

func (m *historyModel) Preview(ctx context.Context, id string) (ui.HistoryPreview, error) {
	if m.store == nil || !m.store.Snapshot().Clipboard.Preview.SpaceEnabled {
		return ui.HistoryPreview{}, nil
	}
	item, err := m.runtime.db.FindByID(ctx, id)
	if err != nil {
		return ui.HistoryPreview{}, err
	}
	if item == nil {
		return ui.HistoryPreview{}, fmt.Errorf("剪贴板记录不存在")
	}
	redact := item.IsSensitive && m.store.Snapshot().Clipboard.Sensitive.RedactSecrets
	preview := ui.HistoryPreview{Enabled: true}
	preview.Meta = previewMeta(item, "")
	switch item.Kind {
	case database.KindImage:
		preview.Title = i18n.T(i18n.KindImage)
		if item.Width != nil && item.Height != nil {
			preview.Title = i18n.Tf(i18n.ImageSizeTitle, *item.Width, *item.Height)
		}
		preview.ImagePath = m.runtime.images.OriginPath(item.Content)
	case database.KindFiles:
		files := clipboard.DecodeFiles(item.Content)
		entries := describeFileEntries(files, item.FileTypes, filePreviewEntryLimit)
		if len(entries) == 1 {
			preview.Title = entries[0].Name
		} else {
			preview.Title = i18n.Tf(i18n.FilesCountTitle, len(files))
		}
		preview.Meta = previewMeta(item, fileSummary(entries, len(files)))
		preview.Text = filePreviewText(entries, len(files))
		if len(entries) == 1 && entries[0].Exists && entries[0].IsImage {
			preview.ImagePath = entries[0].Path
		}
	default:
		preview.Title = i18n.T(i18n.KindText)
		if item.Summary != nil {
			preview.Title = *item.Summary
		}
		preview.Text = previewText(item)
		if item.SubKind != nil && !redact {
			switch *item.SubKind {
			case database.SubHTML, database.SubRTF:
				preview.RichFormat = string(*item.SubKind)
				preview.RichText = item.Content
			}
		}
		if redact {
			preview.Title = clipboard.MaskSecret(preview.Title)
			preview.Text = clipboard.MaskSecret(preview.Text)
		}
	}
	return preview, nil
}

func previewText(item *database.Item) string {
	if item.SubKind != nil && (*item.SubKind == database.SubHTML || *item.SubKind == database.SubRTF) && item.SearchText != nil {
		return *item.SearchText
	}
	return item.Content
}

func previewMeta(item *database.Item, leading string) string {
	parts := []string{leading}
	if leading == "" {
		parts[0] = kindLabel(item.Kind)
	}
	if item.SubKind != nil {
		switch *item.SubKind {
		case database.SubHTML:
			parts[0] = "HTML"
		case database.SubRTF:
			parts[0] = "RTF"
		case database.SubURL:
			parts[0] = i18n.T(i18n.SubKindURL)
		case database.SubEmail:
			parts[0] = i18n.T(i18n.SubKindEmail)
		case database.SubColor:
			parts[0] = i18n.T(i18n.SubKindColor)
		case database.SubPath:
			parts[0] = i18n.T(i18n.SubKindPath)
		}
	}
	if item.Size != nil && item.Kind != database.KindImage {
		parts = append(parts, formatBytes(*item.Size))
	}
	if item.SourceAppName != nil && *item.SourceAppName != "" {
		parts = append(parts, *item.SourceAppName)
	}
	parts = append(parts, item.CreatedAt.Local().Format(time.DateTime))
	return strings.Join(parts, " · ")
}

func kindLabel(kind database.Kind) string {
	switch kind {
	case database.KindImage:
		return i18n.T(i18n.KindImage)
	case database.KindFiles:
		return i18n.T(i18n.KindFiles)
	default:
		return i18n.T(i18n.KindText)
	}
}

func formatBytes(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	value := float64(size)
	for _, unit := range []string{"KB", "MB", "GB", "TB"} {
		value /= 1024
		if value < 1024 {
			return fmt.Sprintf("%.1f %s", value, unit)
		}
	}
	return fmt.Sprintf("%.1f PB", value/1024)
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

func (m *historyModel) historyItem(ctx context.Context, item database.Item) ui.HistoryItem {
	row := ui.HistoryItem{
		ID:        item.ID,
		Kind:      string(item.Kind),
		CreatedAt: item.CreatedAt,
		Favorite:  item.IsFavorite,
		Pinned:    item.IsPinned,
	}
	if item.GroupID != nil {
		row.GroupID = *item.GroupID
	}
	if item.SubKind != nil {
		row.SubKind = string(*item.SubKind)
	}
	if item.SourceAppName != nil {
		row.SourceApp = *item.SourceAppName
	}
	if item.SourceAppIcon != nil && m.runtime.appIcons != nil {
		if path := m.runtime.appIcons.Path(*item.SourceAppIcon); path != "" {
			row.SourceAppIcon = path
		}
	}
	if item.Note != nil {
		row.Note = *item.Note
	}
	switch item.Kind {
	case database.KindText:
		if item.Summary != nil {
			row.Title = *item.Summary
		}
		if item.IsSensitive && m.store != nil && m.store.Snapshot().Clipboard.Sensitive.RedactSecrets {
			row.Title = clipboard.MaskSecret(row.Title)
		}
	case database.KindImage:
		if item.Width != nil && item.Height != nil {
			row.Title = i18n.Tf(i18n.ImageSizeTitle, *item.Width, *item.Height)
		} else {
			row.Title = i18n.T(i18n.KindImage)
		}
		if err := m.runtime.images.EnsureThumbnail(item.Content); err == nil {
			row.ImagePath = m.runtime.images.ThumbPath(item.Content)
		} else {
			row.ImagePath = m.runtime.images.OriginPath(item.Content)
		}
	case database.KindFiles:
		files := clipboard.DecodeFiles(item.Content)
		display := m.Display()
		row.Title = ui.FileTitle(files, display.FileMaxCount)
		entries := describeFileEntries(files, item.FileTypes, display.FileMaxCount)
		row.FileSummary = fileSummary(entries, len(files))
		if len(files) == 1 && len(entries) == 1 && !entries[0].Exists {
			row.Title += i18n.T(i18n.ItemMissing)
		}
		if len(entries) > 0 && m.runtime.fileIconCache != nil {
			row.ImagePath = m.runtime.fileIconCache.Path(ctx, entries[0].Path, entries[0].IsDir)
		}
	}
	return row
}

func fileTypeAt(types *string, index int) bool {
	if types == nil {
		return false
	}
	parts := strings.Split(*types, ",")
	return index >= 0 && index < len(parts) && parts[index] == "d"
}
