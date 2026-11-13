package ui

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
)

// hideAfterAction keeps a pinned panel visible even when an action asks
// to close the window after copy or paste.
func hideAfterAction(pinned, requested bool) bool {
	return requested && !pinned
}

func groupExists(groups []GroupOption, id string) bool {
	for _, group := range groups {
		if group.ID == id {
			return true
		}
	}
	return false
}

// openSelection resolves the category and group the panel selects when it
// opens. A configured group wins over a category; "all groups" clears only
// the group so a category setting still applies.
func openSelection(opening HistoryOpenBehavior, filter, group string) (string, string) {
	nextFilter, nextGroup := filter, group
	if opening.Filter != "preserve" {
		nextFilter = opening.Filter
	}
	switch opening.GroupID {
	case "preserve":
	case "":
		nextGroup = ""
	default:
		nextGroup = opening.GroupID
		nextFilter = ""
	}
	return nextFilter, nextGroup
}

func historyRow(item HistoryItem) macos.HistoryRow {
	title := strings.TrimSpace(item.Title)
	if title == "" {
		title = kindName(item.Kind)
	}
	detail := make([]string, 0, 5)
	detail = append(detail, typeName(item.Kind, item.SubKind))
	if item.FileSummary != "" {
		detail = append(detail, item.FileSummary)
	}
	if item.SourceApp != "" {
		detail = append(detail, item.SourceApp)
	}
	if !item.CreatedAt.IsZero() {
		detail = append(detail, relativeTime(time.Now(), item.CreatedAt))
	}
	if item.Pinned {
		detail = append(detail, i18n.T(i18n.RowPinned))
	}
	if item.Favorite {
		detail = append(detail, i18n.T(i18n.RowFavorite))
	}
	return macos.HistoryRow{
		ID:             item.ID,
		GroupID:        item.GroupID,
		Kind:           item.Kind,
		SubKind:        item.SubKind,
		Title:          title,
		Detail:         strings.Join(detail, " · "),
		ImagePath:      item.ImagePath,
		SourceIconPath: item.SourceAppIcon,
		Note:           item.Note,
		Favorite:       item.Favorite,
		Pinned:         item.Pinned,
		CanDelete:      item.CanDelete,
		DeleteConfirm:  item.DeleteConfirm,
	}
}

func kindName(kind string) string {
	switch kind {
	case "image":
		return i18n.T(i18n.KindImage)
	case "files":
		return i18n.T(i18n.KindFiles)
	default:
		return i18n.T(i18n.KindText)
	}
}

func typeName(kind, subKind string) string {
	switch subKind {
	case "html":
		return "HTML"
	case "rtf":
		return "RTF"
	case "url":
		return i18n.T(i18n.SubKindURL)
	case "email":
		return i18n.T(i18n.SubKindEmail)
	case "color":
		return i18n.T(i18n.SubKindColor)
	case "path":
		return i18n.T(i18n.SubKindPath)
	}
	return kindName(kind)
}

func relativeTime(now, then time.Time) string {
	if then.IsZero() {
		return ""
	}
	diff := now.Sub(then)
	switch {
	case diff < time.Minute:
		return i18n.T(i18n.TimeNow)
	case diff < time.Hour:
		return i18n.Tf(i18n.TimeMinutesAgo, int(diff/time.Minute))
	case diff < 24*time.Hour:
		return i18n.Tf(i18n.TimeHoursAgo, int(diff/time.Hour))
	case diff < 48*time.Hour:
		return i18n.T(i18n.TimeYesterday)
	case diff < 7*24*time.Hour:
		return i18n.Tf(i18n.TimeDaysAgo, int(diff/(24*time.Hour)))
	default:
		return then.Local().Format("01-02")
	}
}

// FileTitle summarizes paths by their base names, as the list shows
// them.
func FileTitle(paths []string, max int) string {
	if len(paths) == 0 {
		return ""
	}
	names := make([]string, 0, min(len(paths), max))
	for i, path := range paths {
		if max > 0 && i == max {
			break
		}
		names = append(names, filepath.Base(path))
	}
	title := strings.Join(names, i18n.T(i18n.FilesSeparator))
	if max > 0 && len(paths) > max {
		title += i18n.Tf(i18n.FilesMore, len(paths))
	}
	return title
}
