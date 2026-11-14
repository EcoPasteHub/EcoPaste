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
	previousTrusted := inputTrusted
	inputTrusted = func() bool { return true }
	t.Cleanup(func() { inputTrusted = previousTrusted })
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
	if !paste || !hide {
		t.Errorf("activate plain returned paste=%v hide=%v", paste, hide)
	}
	if writer.payload.Text != files || len(writer.payload.Files) != 0 {
		t.Errorf("path payload = %+v", writer.payload)
	}
}

func TestMiddleClickActions(t *testing.T) {
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
	plain := "middle text"
	sub := database.SubHTML
	id, _, err := db.Upsert(ctx, &database.Item{
		Kind:       database.KindText,
		SubKind:    &sub,
		Content:    "<b>middle text</b>",
		SearchText: &plain,
		Platform:   "macos",
	})
	if err != nil {
		t.Fatal(err)
	}
	setMiddleClick := func(value string) {
		t.Helper()
		if _, err := store.Update(map[string]any{
			"clipboard": map[string]any{"content": map[string]any{"middleClick": value}},
		}); err != nil {
			t.Fatal(err)
		}
	}
	setMiddleClick("singleClickCopyPlain")
	if hide, err := model.MiddleClick(ctx, id); err != nil || hide {
		t.Fatalf("copy plain = %t, %v", hide, err)
	}
	if writer.payload.Text != plain || writer.payload.HTML != "" {
		t.Errorf("copy plain payload = %+v", writer.payload)
	}
	writer.payload = clipboard.WritePayload{}
	setMiddleClick("singleClickCopy")
	if _, err := model.MiddleClick(ctx, id); err != nil {
		t.Fatal(err)
	}
	if writer.payload.Text != plain || writer.payload.HTML != "<b>middle text</b>" {
		t.Errorf("copy rich payload = %+v", writer.payload)
	}
	writer.payload = clipboard.WritePayload{}
	setMiddleClick("disabled")
	if hide, err := model.MiddleClick(ctx, id); err != nil || hide {
		t.Fatalf("disabled = %t, %v", hide, err)
	}
	if writer.payload.Text != "" || writer.payload.HTML != "" {
		t.Errorf("disabled wrote %+v", writer.payload)
	}
}

func TestClickActionModes(t *testing.T) {
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
	previousTrusted := inputTrusted
	inputTrusted = func() bool { return true }
	t.Cleanup(func() { inputTrusted = previousTrusted })
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
	content := "click text"
	id, _, err := db.Upsert(ctx, &database.Item{
		Kind:     database.KindText,
		Content:  content,
		Platform: "macos",
	})
	if err != nil {
		t.Fatal(err)
	}
	setMode := func(value string) {
		t.Helper()
		if _, err := store.Update(map[string]any{
			"clipboard": map[string]any{"content": map[string]any{"autoPaste": value}},
		}); err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct {
		mode         settings.AutoPaste
		singleWrites bool
		singlePastes bool
		doubleWrites bool
		doublePastes bool
	}{
		{settings.AutoPasteDisabled, false, false, false, false},
		{settings.AutoPasteSingleClickCopy, true, false, false, false},
		{settings.AutoPasteSingleClickPaste, true, true, false, false},
		{settings.AutoPasteDoubleClickCopy, false, false, true, false},
		{settings.AutoPasteDoubleClickPaste, false, false, true, true},
	}
	for _, tc := range cases {
		setMode(string(tc.mode))
		writer.payload = clipboard.WritePayload{}
		paste, _, err := model.ActivateSingleClick(ctx, id)
		if err != nil {
			t.Fatalf("%s single click: %v", tc.mode, err)
		}
		if wrote := writer.payload.Text == content; wrote != tc.singleWrites || paste != tc.singlePastes {
			t.Errorf("%s single click wrote=%t paste=%t", tc.mode, wrote, paste)
		}
		writer.payload = clipboard.WritePayload{}
		paste, _, err = model.ActivateDoubleClick(ctx, id)
		if err != nil {
			t.Fatalf("%s double click: %v", tc.mode, err)
		}
		if wrote := writer.payload.Text == content; wrote != tc.doubleWrites || paste != tc.doublePastes {
			t.Errorf("%s double click wrote=%t paste=%t", tc.mode, wrote, paste)
		}
	}
}
