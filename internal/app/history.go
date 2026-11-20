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
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
	"github.com/EcoPasteHub/EcoPaste/internal/ui"
)

type historyModel struct {
	runtime      *clipboardRuntime
	store        *settings.Store
	openSettings func(string)
}

var inputTrusted = keystroke.Trusted

func newHistoryModel(runtime *clipboardRuntime, store *settings.Store) *historyModel {
	return &historyModel{
		runtime: runtime, store: store,
	}
}

// snapshot returns the published settings, falling back to defaults when the
// store failed to open.
func (m *historyModel) snapshot() settings.Settings {
	if m.store == nil {
		return settings.Default()
	}
	return m.store.Snapshot()
}

// clipboardConfig returns the published clipboard settings, falling back to
// the defaults when settings failed to open.
func (m *historyModel) clipboardConfig() settings.Clipboard {
	return m.snapshot().Clipboard
}

// item loads one history item, reporting a user-facing error when it is gone.
func (m *historyModel) item(ctx context.Context, id string) (*database.Item, error) {
	item, err := m.runtime.db.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, i18n.Error(i18n.HistoryItemMissing)
	}
	resolveItemFileReferences(item)
	return item, nil
}

// resolveItemFileReferences normalizes an old files item in memory so
// display, drag and paste all use the real paths.
func resolveItemFileReferences(item *database.Item) {
	if item.Kind != database.KindFiles {
		return
	}
	files := clipboard.DecodeFiles(item.Content)
	resolved := resolveFileReferences(files)
	if sameFilePaths(files, resolved) {
		return
	}
	item.Content = strings.Join(resolved, "\n")
}

func (m *historyModel) List(ctx context.Context, q ui.HistoryQuery) (ui.HistoryPage, error) {
	cfg := m.clipboardConfig()
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
		row := m.historyItem(ctx, item, cfg)
		row.CanDelete, row.DeleteConfirm = deletePolicy(item, q, cfg.Content)
		items = append(items, row)
	}
	return ui.HistoryPage{Items: items, Total: page.Total, HasMore: page.HasMore}, nil
}

// deletePolicy applies the published per-row delete rules: favorites and
// pinned rows can be protected, and each category has its own confirm
// switch.
func deletePolicy(item database.Item, q ui.HistoryQuery, content settings.Content) (bool, bool) {
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
	item, err := m.item(ctx, id)
	if err != nil {
		return false, err
	}
	cfg := m.clipboardConfig()
	plain := forcePlain || item.Kind == database.KindText && cfg.Content.CopyPlain
	if err := m.write(ctx, item, plain); err != nil {
		return false, err
	}
	return cfg.Content.CopyThenHideWindow, nil
}

func (m *historyModel) Updates() <-chan struct{} { return m.runtime.changed }

func (m *historyModel) SearchBehavior() ui.HistorySearchBehavior {
	search := m.clipboardConfig().Search
	return ui.HistorySearchBehavior{DefaultFocus: search.DefaultFocus, ClearOnHide: search.ClearOnHide}
}

func (m *historyModel) PreviewBehavior() ui.HistoryPreviewBehavior {
	return ui.HistoryPreviewBehavior{SpaceEnabled: m.clipboardConfig().Preview.SpaceEnabled}
}

// Display reports the clamped list density, so a hand-edited settings
// file cannot produce rows without text.
func (m *historyModel) Display() ui.HistoryDisplay {
	return displayOf(m.clipboardConfig())
}

func displayOf(cfg settings.Clipboard) ui.HistoryDisplay {
	return ui.HistoryDisplay{
		TextMaxLines:   clampInt(int(cfg.Display.TextMaxLines), 1, 5),
		ImageMaxHeight: clampInt(int(cfg.Display.ImageMaxHeight), 20, 100),
		FileMaxCount:   clampInt(int(cfg.Display.FileMaxCount), 1, 5),
		ShowOriginal:   cfg.Content.ShowOriginalPreview,
	}
}

// OpenBehavior resolves the configured panel-open selections into the
// list's initial filter and group.
func (m *historyModel) OpenBehavior() ui.HistoryOpenBehavior {
	window := m.clipboardConfig().Window
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
	content := m.clipboardConfig().Content
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

func (m *historyModel) OpenSettings(tab string) {
	if m.openSettings != nil {
		m.openSettings(tab)
	}
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
	return m.activateItem(ctx, id, true)
}

func (m *historyModel) ActivateSingleClick(ctx context.Context, id string) (bool, bool, error) {
	return m.activateByMode(ctx, id, settings.AutoPasteSingleClickPaste, settings.AutoPasteSingleClickCopy)
}

func (m *historyModel) ActivateDoubleClick(ctx context.Context, id string) (bool, bool, error) {
	return m.activateByMode(ctx, id, settings.AutoPasteDoubleClickPaste, settings.AutoPasteDoubleClickCopy)
}

func (m *historyModel) activateByMode(ctx context.Context, id string, pasteMode, copyMode settings.AutoPaste) (bool, bool, error) {
	mode := m.clipboardConfig().Content.AutoPaste
	switch mode {
	case pasteMode:
		return m.activateItem(ctx, id, true)
	case copyMode:
		return m.activateItem(ctx, id, false)
	}
	return false, false, nil
}

func (m *historyModel) activateItem(ctx context.Context, id string, paste bool) (bool, bool, error) {
	item, err := m.item(ctx, id)
	if err != nil {
		return false, false, err
	}
	cfg := m.clipboardConfig()
	if paste && !inputTrusted() {
		return false, false, keystroke.ErrPermission
	}
	plain := !paste && item.Kind == database.KindText && cfg.Content.CopyPlain
	if err := m.write(ctx, item, plain); err != nil {
		return false, false, err
	}
	return paste, paste || cfg.Content.CopyThenHideWindow, nil
}

func (m *historyModel) SendPaste() error { return keystroke.SendPaste() }

// Drag builds the pasteboard representation a row offers when it is
// dragged out of the panel.
func (m *historyModel) Drag(ctx context.Context, id string) (macos.DragPayload, error) {
	item, err := m.item(ctx, id)
	if err != nil {
		return macos.DragPayload{}, err
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
	content := m.clipboardConfig().Content
	if item.IsFavorite && !content.DeleteFavoriteItems {
		return i18n.Error(i18n.ErrorDeleteFavorite)
	}
	if item.IsPinned && !content.DeletePinnedItems {
		return i18n.Error(i18n.ErrorDeletePinned)
	}
	if err := m.runtime.db.Delete(ctx, id); err != nil {
		return err
	}
	if item.Kind == database.KindImage {
		if err := m.runtime.images.Remove(item.Content); err != nil {
			return fmt.Errorf(i18n.T(i18n.ErrorDeleteImage), err)
		}
	}
	return nil
}

// ClearHistory asks which protected rows to keep and removes the rest.
func (m *historyModel) ClearHistory(ctx context.Context) error {
	choice := macos.ConfirmClearHistory()
	if choice == macos.ClearHistoryCancel || m.runtime == nil {
		return nil
	}
	all := choice == macos.ClearHistoryAll
	return m.runtime.clearHistory(ctx, all)
}

func (m *historyModel) Preview(ctx context.Context, id string) (ui.HistoryPreview, error) {
	cfg := m.clipboardConfig()
	if !cfg.Preview.SpaceEnabled {
		return ui.HistoryPreview{}, nil
	}
	item, err := m.item(ctx, id)
	if err != nil {
		return ui.HistoryPreview{}, err
	}
	redact := item.IsSensitive && cfg.Sensitive.RedactSecrets
	preview := ui.HistoryPreview{Enabled: true}
	preview.Meta = previewMeta(item, "")
	switch item.Kind {
	case database.KindImage:
		m.fillImagePreview(&preview, item)
	case database.KindFiles:
		fillFilesPreview(&preview, item)
	default:
		fillTextPreview(&preview, item, redact)
	}
	return preview, nil
}

func (m *historyModel) fillImagePreview(preview *ui.HistoryPreview, item *database.Item) {
	preview.Title = i18n.T(i18n.KindImage)
	if item.Width != nil && item.Height != nil {
		preview.Title = i18n.Tf(i18n.ImageSizeTitle, *item.Width, *item.Height)
	}
	preview.ImagePath = m.runtime.images.OriginPath(item.Content)
}

func fillFilesPreview(preview *ui.HistoryPreview, item *database.Item) {
	files := resolveFileReferences(clipboard.DecodeFiles(item.Content))
	entries := describeFileEntries(files, item.FileTypes, filePreviewEntryLimit)
	if len(entries) == 1 {
		preview.Title = entries[0].Name
	} else {
		preview.Title = i18n.Tf(i18n.FilesCountTitle, len(files))
	}
	preview.Meta = previewMeta(item, fileSummary(entries, len(files)))
	preview.Text = filePreviewText(entries, len(files))
	if len(entries) == 1 && entries[0].Exists && entries[0].IsImage {
		if _, _, ok := macos.ImagePixelSize(entries[0].Path); ok {
			preview.ImagePath = entries[0].Path
		}
	}
}

func fillTextPreview(preview *ui.HistoryPreview, item *database.Item, redact bool) {
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

func previewText(item *database.Item) string {
	if item.SubKind != nil && (*item.SubKind == database.SubHTML || *item.SubKind == database.SubRTF) && item.SearchText != nil {
		return *item.SearchText
	}
	return item.Content
}

func previewMeta(item *database.Item, leading string) string {
	subKind := ""
	if item.SubKind != nil {
		subKind = string(*item.SubKind)
	}
	label := leading
	if label == "" || subKind != "" {
		label = ui.TypeName(string(item.Kind), subKind)
	}
	parts := []string{label}
	if item.Size != nil && item.Kind != database.KindImage {
		parts = append(parts, formatBytes(*item.Size))
	}
	if item.SourceAppName != nil && *item.SourceAppName != "" {
		parts = append(parts, *item.SourceAppName)
	}
	parts = append(parts, item.CreatedAt.Local().Format(time.DateTime))
	return strings.Join(parts, " · ")
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

func (m *historyModel) write(ctx context.Context, item *database.Item, plain bool) error {
	if err := m.runtime.service.WriteItem(ctx, item, plain); err != nil {
		return err
	}
	return m.runtime.db.IncrementUseCount(ctx, item.ID)
}

func (m *historyModel) historyItem(ctx context.Context, item database.Item, cfg settings.Clipboard) ui.HistoryItem {
	row := m.historyItemBase(item)
	switch item.Kind {
	case database.KindText:
		fillTextRow(&row, item, cfg)
	case database.KindImage:
		m.fillImageRow(&row, item)
	case database.KindFiles:
		m.fillFilesRow(ctx, &row, item, cfg)
	}
	return row
}

func (m *historyModel) historyItemBase(item database.Item) ui.HistoryItem {
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
	return row
}

func fillTextRow(row *ui.HistoryItem, item database.Item, cfg settings.Clipboard) {
	if item.Summary != nil {
		row.Title = *item.Summary
	}
	if item.IsSensitive && cfg.Sensitive.RedactSecrets {
		row.Title = clipboard.MaskSecret(row.Title)
	}
}

func (m *historyModel) fillImageRow(row *ui.HistoryItem, item database.Item) {
	if item.Width != nil && item.Height != nil {
		row.Title = i18n.Tf(i18n.ImageSizeTitle, *item.Width, *item.Height)
		row.ImageWidth = int(*item.Width)
		row.ImageHeight = int(*item.Height)
	} else {
		row.Title = i18n.T(i18n.KindImage)
	}
	if err := m.runtime.images.EnsureThumbnail(item.Content); err == nil {
		row.ImagePath = m.runtime.images.ThumbPath(item.Content)
	} else {
		row.ImagePath = m.runtime.images.OriginPath(item.Content)
	}
}

func (m *historyModel) fillFilesRow(ctx context.Context, row *ui.HistoryItem, item database.Item, cfg settings.Clipboard) {
	files := resolveFileReferences(clipboard.DecodeFiles(item.Content))
	display := displayOf(cfg)
	row.Title = ui.FileTitle(files, display.FileMaxCount)
	entries := describeFileEntries(files, item.FileTypes, display.FileMaxCount)
	row.FileSummary = fileSummary(entries, len(files))
	if len(files) == 1 && len(entries) == 1 && !entries[0].Exists {
		row.Title += i18n.T(i18n.ItemMissing)
	}
	// A single copied file that is itself an image reads as an image
	// preview only when its pixels can actually be decoded. Files inside
	// protected folders without Full Disk Access still have metadata, so a
	// failed read must fall through to the regular file layout instead of
	// leaving a card with just the file name.
	if len(files) == 1 && len(entries) == 1 && entries[0].Exists && entries[0].IsImage {
		if width, height, ok := macos.ImagePixelSize(entries[0].Path); ok {
			row.ImageWidth = width
			row.ImageHeight = height
			row.ImagePath = entries[0].Path
			return
		}
	}
	// Every file, single or in a group, lists on its own line with its
	// Finder icon and name.
	row.Files = make([]macos.HistoryFile, 0, len(entries))
	for _, entry := range entries {
		file := macos.HistoryFile{
			Name:    entry.Name,
			IsDir:   entry.IsDir,
			IsImage: entry.IsImage,
			Missing: !entry.Exists,
		}
		if entry.Exists && m.runtime.fileIconCache != nil {
			file.IconPath = m.runtime.fileIconCache.Path(ctx, entry.Path, entry.IsDir)
		}
		row.Files = append(row.Files, file)
	}
}

func fileTypeAt(types *string, index int) bool {
	if types == nil {
		return false
	}
	parts := strings.Split(*types, ",")
	return index >= 0 && index < len(parts) && parts[index] == "d"
}
