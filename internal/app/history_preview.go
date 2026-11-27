package app

import (
	"context"
	"strings"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/clipboard"
	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/EcoPasteHub/EcoPaste/internal/ui"
)

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
	preview := ui.HistoryPreview{Enabled: true, Kind: string(item.Kind)}
	switch item.Kind {
	case database.KindImage:
		preview.Meta = previewMeta(item, "")
		m.fillImagePreview(&preview, item)
	case database.KindFiles:
		fillFilesPreview(&preview, item)
	default:
		preview.Meta = previewMeta(item, "")
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
