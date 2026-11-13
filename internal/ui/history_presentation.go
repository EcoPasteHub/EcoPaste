package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

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

func historyRow(item HistoryItem) macos.HistoryRow {
	title := strings.TrimSpace(item.Title)
	if title == "" {
		title = kindName(item.Kind)
	}
	detail := make([]string, 0, 4)
	if item.FileSummary != "" {
		detail = append(detail, item.FileSummary)
	}
	if item.SourceApp != "" {
		detail = append(detail, item.SourceApp)
	}
	if !item.CreatedAt.IsZero() {
		detail = append(detail, relativeTime(time.Now(), item.CreatedAt))
	}
	if item.Note != "" {
		detail = append(detail, strings.TrimSpace(item.Note))
	}
	if item.Pinned {
		detail = append(detail, "已固定")
	}
	if item.Favorite {
		detail = append(detail, "收藏")
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
	}
}

func kindName(kind string) string {
	switch kind {
	case "image":
		return "图片"
	case "files":
		return "文件"
	default:
		return "文本"
	}
}

func relativeTime(now, then time.Time) string {
	if then.IsZero() {
		return ""
	}
	diff := now.Sub(then)
	switch {
	case diff < time.Minute:
		return "刚刚"
	case diff < time.Hour:
		return fmt.Sprintf("%d 分钟前", int(diff/time.Minute))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%d 小时前", int(diff/time.Hour))
	case diff < 48*time.Hour:
		return "昨天"
	case diff < 7*24*time.Hour:
		return fmt.Sprintf("%d 天前", int(diff/(24*time.Hour)))
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
	title := strings.Join(names, "、")
	if max > 0 && len(paths) > max {
		title += fmt.Sprintf(" 等 %d 个", len(paths))
	}
	return title
}
