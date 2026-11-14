package clipboard

import (
	"context"
	"fmt"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
)

// WritePayload is the content handed to the platform pasteboard.
type WritePayload struct {
	Text  string
	HTML  string
	RTF   string
	PNG   []byte
	Files []string
}

// Writer replaces the system pasteboard.
type Writer interface {
	Write(WritePayload) error
}

// BuildWrite converts a history row into the pasteboard content that
// reproduces it. plainText drops HTML and RTF but keeps their plain
// fallback.
func (s *Service) BuildWrite(item *database.Item, plainText bool) (WritePayload, error) {
	if item == nil {
		return WritePayload{}, fmt.Errorf("nil clipboard item")
	}
	switch item.Kind {
	case database.KindText:
		plain := item.Content
		if item.SearchText != nil && *item.SearchText != "" {
			plain = *item.SearchText
		}
		if plainText || item.SubKind == nil {
			return WritePayload{Text: plain}, nil
		}
		switch *item.SubKind {
		case database.SubHTML:
			return WritePayload{Text: plain, HTML: item.Content}, nil
		case database.SubRTF:
			return WritePayload{Text: plain, RTF: item.Content}, nil
		}
		return WritePayload{Text: plain}, nil
	case database.KindImage:
		png, err := s.Images.Read(item.Content)
		if err != nil {
			return WritePayload{}, fmt.Errorf("read image: %w", err)
		}
		return WritePayload{PNG: png}, nil
	case database.KindFiles:
		if plainText {
			return WritePayload{Text: item.Content}, nil
		}
		return WritePayload{Files: DecodeFiles(item.Content)}, nil
	}
	return WritePayload{}, fmt.Errorf("unknown clipboard kind %q", item.Kind)
}

// WriteItem puts an item back on the pasteboard and suppresses the
// watcher's own capture of that write.
func (s *Service) WriteItem(ctx context.Context, item *database.Item, plainText bool) error {
	payload, err := s.BuildWrite(item, plainText)
	if err != nil {
		return err
	}
	if s.Writer == nil {
		return fmt.Errorf("no pasteboard writer")
	}
	if s.Guard != nil {
		for _, hash := range guardHashes(item, payload) {
			s.Guard.Mark(hash)
		}
	}
	if err := s.Writer.Write(payload); err != nil {
		return fmt.Errorf("write pasteboard: %w", err)
	}
	return nil
}

// guardHashes covers every representation the pasteboard write exposes.
// Rich text also carries a plain fallback, so marking only the item hash
// would let the watcher capture that fallback as a new item.
func guardHashes(item *database.Item, payload WritePayload) []string {
	hashes := []string{ItemHash(item)}
	if payload.Text != "" {
		hashes = append(hashes, database.Hash(database.KindText, payload.Text))
	}
	if payload.HTML != "" {
		hashes = append(hashes, database.Hash(database.KindText, payload.HTML))
	}
	if payload.RTF != "" {
		hashes = append(hashes, database.Hash(database.KindText, payload.RTF))
	}
	seen := make(map[string]struct{}, len(hashes))
	unique := hashes[:0]
	for _, hash := range hashes {
		if hash == "" {
			continue
		}
		if _, ok := seen[hash]; ok {
			continue
		}
		seen[hash] = struct{}{}
		unique = append(unique, hash)
	}
	return unique
}
