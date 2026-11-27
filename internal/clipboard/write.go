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

// Writer replaces the system pasteboard and returns its resulting change count.
type Writer interface {
	Write(WritePayload) (changeCount int, changed bool, err error)
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
		return WritePayload{Files: DecodeFiles(item.Content)}, nil
	}
	return WritePayload{}, fmt.Errorf("unknown clipboard kind %q", item.Kind)
}

// WriteItem puts an item back on the pasteboard and suppresses the
// watcher's own capture of that write.
func (s *Service) WriteItem(ctx context.Context, item *database.Item, plainText bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	payload, err := s.BuildWrite(item, plainText)
	if err != nil {
		return err
	}
	if s.Writer == nil {
		return fmt.Errorf("no pasteboard writer")
	}
	write := func() (int, bool, error) { return s.Writer.Write(payload) }
	if s.Guard != nil {
		if err := s.Guard.Write(write); err != nil {
			return fmt.Errorf("write pasteboard: %w", err)
		}
		return nil
	}
	if _, _, err := write(); err != nil {
		return fmt.Errorf("write pasteboard: %w", err)
	}
	return nil
}
