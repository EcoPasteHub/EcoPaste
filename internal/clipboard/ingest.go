package clipboard

import (
	"context"
	"os"
	"slices"
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
// exclude it, its size exceeds a limit, or EcoPaste wrote it itself.
func (s *Service) Capture(ctx context.Context, payload Payload, source SourceApp) (id string, changed bool, err error) {
	if s.Settings != nil && source.ID != "" && slices.Contains(s.Settings().Filters.ExcludedAppIDs, source.ID) {
		return "", false, nil
	}
	item, err := s.buildItem(payload, source)
	if err != nil {
		return "", false, err
	}
	if item == nil {
		return "", false, nil
	}
	if s.Guard != nil && s.Guard.Consume(item.ContentHash) {
		return "", false, nil
	}
	if source.ID != "" {
		if err := s.DB.EnsureApp(ctx, source.ID, source.Name, source.IconFile); err != nil {
			return "", false, err
		}
	}
	id, _, err = s.DB.Upsert(ctx, item)
	if err != nil {
		return "", false, err
	}
	return id, true, nil
}

func (s *Service) buildItem(payload Payload, source SourceApp) (*database.Item, error) {
	capture := settings.Capture{}
	sensitiveCfg := settings.Sensitive{CollectSecrets: true}
	if s.Settings != nil {
		cfg := s.Settings()
		capture = cfg.Capture
		sensitiveCfg = cfg.Sensitive
	}
	plain := strings.TrimSpace(payload.Text)
	sensitive := ContainsSecret(plain)
	if sensitive && !sensitiveCfg.CollectSecrets {
		return nil, nil
	}
	for _, kind := range orderedKinds(capture.Order) {
		if !captureEnabled(capture, kind) {
			continue
		}
		switch kind {
		case settings.CaptureFiles:
			if len(payload.Files) == 0 {
				continue
			}
			content := strings.Join(payload.Files, "\n")
			item := &database.Item{
				Kind:        database.KindFiles,
				Content:     content,
				ContentHash: database.Hash(database.KindFiles, content),
				FileTypes:   fileTypes(payload.Files),
				Platform:    "macos",
			}
			return s.withSource(item, source), nil
		case settings.CaptureImage:
			if len(payload.PNG) == 0 || exceeds(len(payload.PNG), capture.MaxImageMB) {
				continue
			}
			stored, err := s.Images.StorePNG(payload.PNG)
			if err != nil {
				return nil, err
			}
			item := &database.Item{
				Kind:        database.KindImage,
				Content:     stored.FileName,
				ContentHash: database.Hash(database.KindImage, stored.FileName),
				Size:        &stored.Size,
				Width:       &stored.Width,
				Height:      &stored.Height,
				Platform:    "macos",
			}
			return s.withSource(item, source), nil
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
		Platform:    "macos",
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

func orderedKinds(order []settings.CaptureKind) []settings.CaptureKind {
	seen := map[settings.CaptureKind]bool{}
	out := make([]settings.CaptureKind, 0, 5)
	for _, kind := range append(append([]settings.CaptureKind{}, order...), settings.DefaultCaptureOrder()...) {
		if !seen[kind] {
			seen[kind] = true
			out = append(out, kind)
		}
	}
	return out
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
	if utf8.RuneCountInString(trimmed) > SummaryMaxChars {
		runes := []rune(trimmed)
		trimmed = string(runes[:SummaryMaxChars])
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

// ItemHash is the guard key for writing an existing item back.
func ItemHash(item *database.Item) string {
	if item == nil {
		return ""
	}
	return item.ContentHash
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
