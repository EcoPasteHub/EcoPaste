package clipboard

import (
	"errors"
	"os"
	"strings"

	"github.com/EcoPasteHub/EcoPaste/internal/domain"
	"github.com/EcoPasteHub/EcoPaste/internal/resources"
	"github.com/EcoPasteHub/EcoPaste/internal/storage"
)

// Writer is the platform pasteboard write port.
type Writer interface {
	WriteText(plain, html, rtf string, forcePlain bool)
	WriteImage(png []byte) error
	WriteFiles(paths []string) error
}

// WriteItem writes an item back and arms the loop guard first.
func WriteItem(
	writer Writer,
	guard *WritebackGuard,
	images *resources.ImageStore,
	item domain.Item,
	plain bool,
) error {
	switch item.Kind {
	case domain.KindText:
		content := item.Content
		if plain {
			content = item.SearchText
			if content == "" {
				content = item.Content
			}
			guard.Suppress(storage.ContentHash(domain.KindText, content))
			writer.WriteText(content, "", "", true)
			return nil
		}
		if item.SubKind == "html" || item.SubKind == "rtf" {
			fallback := item.SearchText
			if fallback == "" {
				fallback = item.Content
			}
			guard.Suppress(storage.ContentHash(domain.KindText, item.Content))
			guard.Suppress(storage.ContentHash(domain.KindText, fallback))
			if item.SubKind == "html" {
				writer.WriteText(fallback, item.Content, "", false)
			} else {
				writer.WriteText(fallback, "", item.Content, false)
			}
			return nil
		}
		guard.Suppress(storage.ContentHash(domain.KindText, content))
		writer.WriteText(content, "", "", true)
		return nil
	case domain.KindImage:
		if images == nil {
			return errors.New("image store is unavailable")
		}
		raw, err := os.ReadFile(images.OriginPath(item.Content))
		if err != nil {
			return err
		}
		guard.Suppress(item.ContentHash)
		return writer.WriteImage(raw)
	case domain.KindFiles:
		if plain {
			guard.Suppress(storage.ContentHash(domain.KindText, item.Content))
			writer.WriteText(item.Content, "", "", true)
			return nil
		}
		paths := splitFiles(item.Content)
		if len(paths) == 0 {
			return errors.New("no files to write")
		}
		guard.Suppress(item.ContentHash)
		return writer.WriteFiles(paths)
	default:
		return errors.New("unsupported clipboard kind")
	}
}

func splitFiles(content string) []string {
	parts := strings.Split(content, "\n")
	out := parts[:0]
	for _, part := range parts {
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
