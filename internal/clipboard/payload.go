package clipboard

import (
	"path/filepath"

	"github.com/EcoPasteHub/EcoPaste/internal/domain"
	"github.com/EcoPasteHub/EcoPaste/internal/imageutil"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
)

// Payload is the first enabled representation found on the pasteboard.
type Payload struct {
	Kind  domain.Kind
	Text  Text
	Image Image
	Files []string
}

type Text struct {
	Plain string
	HTML  string
	RTF   string
}

type Image struct {
	Bytes  []byte
	Width  int
	Height int
}

// BuildItem converts a payload into the persisted clipboard item shape.
func BuildItem(payload Payload, capture settings.Capture, sensitive settings.Sensitive) (domain.Item, bool, error) {
	var item domain.Item
	item.Platform = "macos"
	item.UseCount = 1

	switch payload.Kind {
	case domain.KindText:
		if !capture.Text && !capture.HTML && !capture.RTF {
			return item, false, nil
		}
		plain := trimSpace(payload.Text.Plain)
		if plain == "" {
			return item, false, nil
		}
		if containsSecret(plain) {
			if !sensitive.CollectSecrets {
				return item, false, nil
			}
			item.IsSensitive = true
		}

		content := plain
		search := plain
		chosen := false
		for _, kind := range orderedKinds(capture) {
			if !captureEnabled(capture, kind) {
				continue
			}
			switch kind {
			case "html":
				if payload.Text.HTML != "" {
					content = payload.Text.HTML
					item.SubKind = "html"
					chosen = true
				}
			case "rtf":
				if payload.Text.RTF != "" {
					content = payload.Text.RTF
					item.SubKind = "rtf"
					chosen = true
				}
			case "text":
				content = plain
				item.SubKind = detectTextSubKind(plain)
				chosen = true
			}
			if chosen {
				break
			}
		}
		if !chosen {
			return item, false, nil
		}
		item.Kind = domain.KindText
		item.Content = content
		item.SearchText = search
		item.Summary = makeSummary(plain)
		item.Size = int64(len(content))
	case domain.KindFiles:
		if !capture.Files || len(payload.Files) == 0 {
			return item, false, nil
		}
		item.Kind = domain.KindFiles
		item.Content = joinFiles(payload.Files)
		if trimSpace(item.Content) == "" {
			return item, false, nil
		}
		item.FileTypes = fileTypes(payload.Files)
		item.SearchText = fileSearchText(payload.Files)
	case domain.KindImage:
		if !capture.Image || len(payload.Image.Bytes) == 0 {
			return item, false, nil
		}
		item.Kind = domain.KindImage
		item.Content = pngFileName(payload.Image.Bytes)
		item.Size = int64(len(payload.Image.Bytes))
		item.Width = int64(payload.Image.Width)
		item.Height = int64(payload.Image.Height)
	}
	return item, true, nil
}

// DecodeImage normalizes image bytes to PNG and reports its dimensions.
func DecodeImage(data []byte) (Image, error) {
	image, err := imageutil.Decode(data)
	if err != nil {
		return Image{}, err
	}
	return Image{Bytes: image.Bytes, Width: image.Width, Height: image.Height}, nil
}

func pngFileName(data []byte) string {
	return hashBytes(data) + ".png"
}

func joinFiles(files []string) string {
	var out string
	for index, path := range files {
		if index > 0 {
			out += "\n"
		}
		out += path
	}
	return out
}

func fileTypes(files []string) string {
	out := ""
	for index, path := range files {
		if index > 0 {
			out += ","
		}
		if isDir(path) {
			out += "d"
		} else {
			out += "f"
		}
	}
	return out
}

func fileSearchText(files []string) string {
	out := ""
	for index, path := range files {
		if index > 0 {
			out += "\n"
		}
		out += filepath.Base(path)
	}
	return out
}

func makeSummary(plain string) string {
	runes := []rune(plain)
	if len(runes) > 256 {
		runes = runes[:256]
	}
	return string(runes)
}

func orderedKinds(capture settings.Capture) []string {
	seen := map[string]bool{}
	out := make([]string, 0, 5)
	for _, kind := range append(capture.Order, "files", "image", "html", "rtf", "text") {
		if !seen[kind] {
			seen[kind] = true
			out = append(out, kind)
		}
	}
	return out
}

func captureEnabled(capture settings.Capture, kind string) bool {
	switch kind {
	case "files":
		return capture.Files
	case "image":
		return capture.Image
	case "html":
		return capture.HTML
	case "rtf":
		return capture.RTF
	default:
		return capture.Text
	}
}
