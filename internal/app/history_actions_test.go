package app

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/clipboard"
	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
	"github.com/EcoPasteHub/EcoPaste/internal/storage"
)

type recordingWriter struct {
	payload clipboard.WritePayload
}

func (w *recordingWriter) Write(payload clipboard.WritePayload) error {
	w.payload = payload
	return nil
}

func TestPlainAndPathReuseActions(t *testing.T) {
	root := t.TempDir()
	db, err := database.Open(filepath.Join(root, "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	store, err := settings.Open(filepath.Join(root, settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(map[string]any{
		"clipboard": map[string]any{
			"content": map[string]any{"autoPaste": "disabled"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	writer := &recordingWriter{}
	model := &historyModel{
		store: store,
		runtime: &clipboardRuntime{
			db:      db,
			images:  storage.NewImageStore(filepath.Join(root, "resources")),
			service: &clipboard.Service{DB: db, Writer: writer},
		},
	}
	ctx := context.Background()
	plain := "rich text"
	sub := database.SubHTML
	htmlID, _, err := db.Upsert(ctx, &database.Item{
		Kind:       database.KindText,
		SubKind:    &sub,
		Content:    "<b>rich text</b>",
		SearchText: &plain,
		Platform:   "macos",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := model.CopyPlain(ctx, htmlID); err != nil {
		t.Fatal(err)
	}
	if writer.payload.Text != plain || writer.payload.HTML != "" {
		t.Errorf("copy plain payload = %+v", writer.payload)
	}
	if _, err := model.Copy(ctx, htmlID); err != nil {
		t.Fatal(err)
	}
	if writer.payload.Text != plain || writer.payload.HTML != "<b>rich text</b>" {
		t.Errorf("copy rich payload = %+v", writer.payload)
	}

	files := "/tmp/a\n/tmp/b"
	fileID, _, err := db.Upsert(ctx, &database.Item{
		Kind:     database.KindFiles,
		Content:  files,
		Platform: "macos",
	})
	if err != nil {
		t.Fatal(err)
	}
	paste, hide, err := model.ActivatePlain(ctx, fileID)
	if err != nil {
		t.Fatal(err)
	}
	if paste || hide {
		t.Errorf("activate plain returned paste=%v hide=%v", paste, hide)
	}
	if writer.payload.Text != files || len(writer.payload.Files) != 0 {
		t.Errorf("path payload = %+v", writer.payload)
	}
}
