package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/renameio/v2"

	"github.com/EcoPasteHub/EcoPaste/internal/clipboard"
	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
)

var (
	openExternal = macos.OpenURL
	openFile     = macos.OpenPath
	revealFiles  = macos.RevealPaths
	chooseSave   = macos.SavePanel
)

const noteMaxChars = 256

// Open opens a link, email or existing path.
func (m *historyModel) Open(ctx context.Context, id string) error {
	item, err := m.item(ctx, id)
	if err != nil {
		return err
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
	return i18n.Error(i18n.ErrorOpenUnsupported)
}

// Reveal shows files or a path item in Finder.
func (m *historyModel) Reveal(ctx context.Context, id string) error {
	item, err := m.item(ctx, id)
	if err != nil {
		return err
	}
	switch {
	case item.Kind == database.KindFiles:
		return revealFiles(clipboard.DecodeFiles(item.Content))
	case item.Kind == database.KindText && item.SubKind != nil && *item.SubKind == database.SubPath:
		return revealFiles([]string{strings.TrimSpace(item.Content)})
	default:
		return i18n.Error(i18n.ErrorRevealUnsupported)
	}
}

// SaveImage writes an image item to a user-chosen PNG file.
func (m *historyModel) SaveImage(ctx context.Context, id string) error {
	item, err := m.item(ctx, id)
	if err != nil {
		return err
	}
	if item.Kind != database.KindImage {
		return i18n.Error(i18n.ErrorNotImage)
	}
	defaultName := fmt.Sprintf("EcoPaste-%s.png", item.CreatedAt.Local().Format("20060102-150405"))
	path, ok := chooseSave(defaultName, i18n.T(i18n.SaveImagePanel))
	if !ok {
		return nil
	}
	data, err := m.runtime.images.Read(item.Content)
	if err != nil {
		return fmt.Errorf(i18n.T(i18n.ErrorReadImage), err)
	}
	if err := writeAtomic(path, data); err != nil {
		return fmt.Errorf(i18n.T(i18n.ErrorSaveImage), err)
	}
	return nil
}

// SaveNote persists a note typed in the panel's inline note popover,
// normalizes the value and applies auto-favorite when the setting is enabled.
func (m *historyModel) SaveNote(ctx context.Context, id, value string) error {
	if _, err := m.item(ctx, id); err != nil {
		return err
	}
	value = strings.TrimSpace(value)
	var note *string
	if value != "" {
		if runes := []rune(value); len(runes) > noteMaxChars {
			value = string(runes[:noteMaxChars])
		}
		note = &value
	}
	if err := m.runtime.db.SetNote(ctx, id, note); err != nil {
		return err
	}
	if note != nil && m.clipboardConfig().Content.AutoFavorite {
		if err := m.runtime.db.MarkFavorite(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func writeAtomic(path string, data []byte) error {
	return renameio.WriteFile(path, data, 0o644)
}
