package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/EcoPasteHub/EcoPaste/internal/clipboard"
	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
)

var (
	openExternal = macos.OpenURL
	openFile     = macos.OpenPath
	revealFiles  = macos.RevealPaths
	chooseSave   = macos.SavePanel
)

// Open opens a link, email or existing path.
func (m *historyModel) Open(ctx context.Context, id string) error {
	item, err := m.runtime.db.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if item == nil {
		return fmt.Errorf("剪贴板记录不存在")
	}
	if item.Kind == database.KindText && item.SubKind != nil {
		value := strings.TrimSpace(item.Content)
		switch *item.SubKind {
		case database.SubURL:
			if strings.HasPrefix(strings.ToLower(value), "www.") {
				value = "https://" + value
			}
			return openExternal(value)
		case database.SubEmail:
			return openExternal("mailto:" + value)
		case database.SubPath:
			return openFile(value)
		}
	}
	return fmt.Errorf("该记录不支持打开")
}

// Reveal shows files or a path item in Finder.
func (m *historyModel) Reveal(ctx context.Context, id string) error {
	item, err := m.runtime.db.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if item == nil {
		return fmt.Errorf("剪贴板记录不存在")
	}
	switch {
	case item.Kind == database.KindFiles:
		return revealFiles(clipboard.DecodeFiles(item.Content))
	case item.Kind == database.KindText && item.SubKind != nil && *item.SubKind == database.SubPath:
		return revealFiles([]string{strings.TrimSpace(item.Content)})
	default:
		return fmt.Errorf("该记录不支持在 Finder 中显示")
	}
}

// SaveImage writes an image item to a user-chosen PNG file.
func (m *historyModel) SaveImage(ctx context.Context, id string) error {
	item, err := m.runtime.db.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if item == nil {
		return fmt.Errorf("剪贴板记录不存在")
	}
	if item.Kind != database.KindImage {
		return fmt.Errorf("该记录不是图片")
	}
	defaultName := fmt.Sprintf("EcoPaste-%s.png", item.CreatedAt.Local().Format("20060102-150405"))
	path, ok := chooseSave(defaultName, "存储剪贴板图片")
	if !ok {
		return nil
	}
	data, err := m.runtime.images.Read(item.Content)
	if err != nil {
		return fmt.Errorf("读取图片: %w", err)
	}
	if err := writeAtomic(path, data); err != nil {
		return fmt.Errorf("保存图片: %w", err)
	}
	return nil
}

func writeAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".ecopaste-*.png")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}
