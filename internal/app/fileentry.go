package app

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
)

const (
	filePreviewEntryLimit = 64
)

type fileEntry struct {
	Path    string
	Name    string
	IsDir   bool
	IsImage bool
	Exists  bool
	Size    int64
}

// describeFileEntries reads the metadata the list and preview need. When a
// path is gone, the file_types hint preserved at capture time still lets
// the UI distinguish a folder from a file.
func describeFileEntries(paths []string, types *string, limit int) []fileEntry {
	count := len(paths)
	if limit > 0 && count > limit {
		count = limit
	}
	entries := make([]fileEntry, 0, count)
	for index, path := range paths[:count] {
		entry := fileEntry{Path: path, Name: filepath.Base(path)}
		info, err := os.Stat(path)
		if err == nil {
			entry.Exists = true
			entry.IsDir = info.IsDir()
			if !entry.IsDir {
				entry.Size = info.Size()
			}
		} else {
			entry.IsDir = fileTypeAt(types, index)
		}
		entry.IsImage = !entry.IsDir && isImageFile(path)
		entries = append(entries, entry)
	}
	return entries
}

func fileEntryLabel(entry fileEntry) string {
	switch {
	case !entry.Exists:
		return i18n.T(i18n.FileMissing)
	case entry.IsDir:
		return i18n.T(i18n.FileFolder)
	case entry.IsImage:
		return i18n.T(i18n.KindImage)
	default:
		ext := strings.TrimPrefix(filepath.Ext(entry.Name), ".")
		if ext == "" {
			return i18n.T(i18n.KindFiles)
		}
		return strings.ToUpper(ext)
	}
}

// fileSummary is the compact metadata shown below a file row.
func fileSummary(entries []fileEntry, total int) string {
	if total <= 0 {
		return ""
	}
	if total == 1 && len(entries) == 1 {
		entry := entries[0]
		label := fileEntryLabel(entry)
		if entry.Exists && !entry.IsDir {
			return label + " · " + formatBytes(entry.Size)
		}
		return label
	}
	summary := i18n.Tf(i18n.FileSummaryCount, total)
	if total == len(entries) {
		missing := 0
		for _, entry := range entries {
			if !entry.Exists {
				missing++
			}
		}
		if missing > 0 {
			summary += i18n.Tf(i18n.FileSummaryMissing, missing)
		}
	}
	return summary
}

// filePreviewText lays out file metadata in the native text preview. It
// keeps every path visible while marking missing files explicitly.
func filePreviewText(entries []fileEntry, total int) string {
	var lines []string
	for _, entry := range entries {
		label := fileEntryLabel(entry)
		if entry.Exists && !entry.IsDir {
			label += " · " + formatBytes(entry.Size)
		}
		lines = append(lines, entry.Name+" · "+label, entry.Path, "")
	}
	if total > len(entries) {
		lines = append(lines, i18n.Tf(i18n.FilePreviewTruncated, len(entries), total))
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

func isImageFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg", ".png", ".webp", ".avif", ".gif", ".svg",
		".bmp", ".ico", ".tif", ".tiff", ".heic", ".apng":
		return true
	}
	return false
}
