package clipboard

import (
	"context"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
	"github.com/EcoPasteHub/EcoPaste/internal/storage"
)

// SummaryMaxChars is the list summary's published limit.
const SummaryMaxChars = 256

// Service turns pasteboard snapshots into history rows.
type Service struct {
	DB       *database.DB
	Images   *storage.ImageStore
	Settings func() settings.Clipboard
	Guard    *Guard
	Writer   Writer
}

// Capture stores one snapshot, or returns changed=false when the settings
// exclude it or its size exceeds a limit.
func (s *Service) Capture(ctx context.Context, payload Payload, source SourceApp) (id string, changed bool, err error) {
	id, changed, _, err = s.CaptureDetailed(ctx, payload, source)
	return id, changed, err
}

// CaptureDetailed also reports whether the snapshot reused an existing row.
func (s *Service) CaptureDetailed(ctx context.Context, payload Payload, source SourceApp) (id string, changed, deduplicated bool, err error) {
	cfg := settings.Clipboard{Sensitive: settings.Sensitive{CollectSecrets: true}}
	if s.Settings != nil {
		cfg = s.Settings()
	}
	if source.ID != "" && cfg.SourceApps.BlocksCapture(source.ID) {
		return "", false, false, nil
	}
	item, err := s.buildItem(payload, source, cfg)
	if err != nil {
		return "", false, false, err
	}
	if item == nil {
		return "", false, false, nil
	}
	if source.ID != "" {
		if err := s.DB.EnsureApp(ctx, source.ID, source.Name, source.IconFile); err != nil {
			return "", false, false, err
		}
	}
	id, deduplicated, err = s.DB.Upsert(ctx, item)
	if err != nil {
		return "", false, false, err
	}
	return id, true, deduplicated, nil
}

func (s *Service) buildItem(payload Payload, source SourceApp, cfg settings.Clipboard) (*database.Item, error) {
	capture := cfg.Capture
	plain := strings.TrimSpace(payload.Text)
	sensitive := ContainsSecret(plain)
	if sensitive && !cfg.Sensitive.CollectSecrets {
		return nil, nil
	}
	for _, kind := range settings.NormalizeCaptureOrder(capture.Order) {
		if !captureEnabled(capture, kind) {
			continue
		}
		switch kind {
		case settings.CaptureFiles:
			if item := filesItem(payload); item != nil {
				return s.withSource(item, source), nil
			}
		case settings.CaptureImage:
			item, err := s.imageItem(payload, capture)
			if err != nil {
				return nil, err
			}
			if item != nil {
				return s.withSource(item, source), nil
			}
		case settings.CaptureHTML:
			if plain == "" || strings.TrimSpace(payload.HTML) == "" {
				continue
			}
			return s.textItem(payload.HTML, plain, database.SubHTML, capture.MaxTextMB, source, sensitive), nil
		case settings.CaptureRTF:
			if plain == "" || strings.TrimSpace(payload.RTF) == "" {
				continue
			}
			return s.textItem(payload.RTF, plain, database.SubRTF, capture.MaxTextMB, source, sensitive), nil
		case settings.CaptureText:
			if plain == "" {
				continue
			}
			return s.textItem(plain, plain, "", capture.MaxTextMB, source, sensitive), nil
		}
	}
	return nil, nil
}

func filesItem(payload Payload) *database.Item {
	if len(payload.Files) == 0 {
		return nil
	}
	content := strings.Join(payload.Files, "\n")
	return &database.Item{
		Kind:        database.KindFiles,
		Content:     content,
		ContentHash: database.Hash(database.KindFiles, content),
		FileTypes:   fileTypes(payload.Files),
	}
}

func (s *Service) imageItem(payload Payload, capture settings.Capture) (*database.Item, error) {
	if len(payload.PNG) == 0 || exceeds(len(payload.PNG), capture.MaxImageMB) {
		return nil, nil
	}
	stored, err := s.Images.StorePNG(payload.PNG)
	if err != nil {
		return nil, err
	}
	return &database.Item{
		Kind:        database.KindImage,
		Content:     stored.FileName,
		ContentHash: database.Hash(database.KindImage, stored.FileName),
		Size:        &stored.Size,
		Width:       &stored.Width,
		Height:      &stored.Height,
	}, nil
}

func (s *Service) textItem(content, plain string, sub database.SubKind, maxMB uint32, source SourceApp, sensitive bool) *database.Item {
	if exceeds(len(content), maxMB) {
		return nil
	}
	size := int64(len(content))
	search := plain
	summary := summarize(plain)
	item := &database.Item{
		Kind:        database.KindText,
		Content:     content,
		ContentHash: database.Hash(database.KindText, content),
		SearchText:  &search,
		Summary:     summary,
		Size:        &size,
		IsSensitive: sensitive,
	}
	if sub != "" {
		item.SubKind = &sub
	} else if kind, ok := DetectTextSubKind(plain); ok {
		item.SubKind = &kind
	}
	return s.withSource(item, source)
}

func (s *Service) withSource(item *database.Item, source SourceApp) *database.Item {
	if source.ID != "" {
		id := source.ID
		item.SourceAppID = &id
	}
	return item
}

func captureEnabled(c settings.Capture, kind settings.CaptureKind) bool {
	switch kind {
	case settings.CaptureText:
		return c.Text
	case settings.CaptureHTML:
		return c.HTML
	case settings.CaptureRTF:
		return c.RTF
	case settings.CaptureImage:
		return c.Image
	case settings.CaptureFiles:
		return c.Files
	}
	return false
}

func exceeds(size int, maxMB uint32) bool {
	return maxMB != 0 && int64(size) > int64(maxMB)*1024*1024
}

func summarize(plain string) *string {
	trimmed := strings.TrimSpace(plain)
	if trimmed == "" {
		return nil
	}
	characters := 0
	for offset := range trimmed {
		if characters == SummaryMaxChars {
			prefix := trimmed[:offset]
			if utf8.ValidString(prefix) {
				trimmed = strings.Clone(prefix)
			} else {
				// Keep one replacement rune per malformed byte when truncating.
				trimmed = string([]rune(prefix))
			}
			break
		}
		characters++
	}
	return &trimmed
}

func fileTypes(paths []string) *string {
	types := make([]string, 0, len(paths))
	for _, path := range paths {
		info, err := os.Stat(path)
		switch {
		case err == nil && info.IsDir():
			types = append(types, "d")
		default:
			types = append(types, "f")
		}
	}
	if len(types) == 0 {
		return nil
	}
	value := strings.Join(types, ",")
	return &value
}

// DecodeFiles splits a file item's content back into paths.
func DecodeFiles(content string) []string {
	var paths []string
	for _, line := range strings.Split(content, "\n") {
		if line != "" {
			paths = append(paths, line)
		}
	}
	return paths
}
