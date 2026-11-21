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
	payload     clipboard.WritePayload
	err         error
	changeCount int
}

func (w *recordingWriter) Write(payload clipboard.WritePayload) (int, bool, error) {
	w.payload = payload
	if w.err != nil {
		return 0, false, w.err
	}
	w.changeCount++
	return w.changeCount, true, nil
}

func TestReuseActionsPreserveRepresentations(t *testing.T) {
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
		Kind:    database.KindFiles,
		Content: files,
	})
	if err != nil {
		t.Fatal(err)
	}
	paste, hide, err := model.Activate(ctx, fileID)
	if err != nil {
		t.Fatal(err)
	}
	if !paste || !hide {
		t.Errorf("activate files returned paste=%v hide=%v", paste, hide)
	}
	if writer.payload.Text != "" || len(writer.payload.Files) != 2 || writer.payload.Files[0] != "/tmp/a" || writer.payload.Files[1] != "/tmp/b" {
		t.Errorf("files payload = %+v", writer.payload)
	}

	if _, err := store.Update(map[string]any{
		"clipboard": map[string]any{"content": map[string]any{
			"copyThenHideWindow": true,
		}},
	}); err != nil {
		t.Fatal(err)
	}
	writer.payload = clipboard.WritePayload{}
	if _, _, err := model.Activate(ctx, fileID); err != nil {
		t.Fatal(err)
	}
	if writer.payload.Text != "" || len(writer.payload.Files) != 2 || writer.payload.Files[0] != "/tmp/a" || writer.payload.Files[1] != "/tmp/b" {
		t.Errorf("files payload with hide enabled = %+v", writer.payload)
	}
	hide, err = model.Copy(ctx, htmlID)
	if err != nil {
		t.Fatal(err)
	}
	if !hide {
		t.Error("copy did not request hiding the panel")
	}

	before, err := db.FindByID(ctx, htmlID)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := model.Activate(ctx, htmlID); err != nil {
		t.Fatal(err)
	}
	after, err := db.FindByID(ctx, htmlID)
	if err != nil {
		t.Fatal(err)
	}
	if after.UseCount != before.UseCount || !after.CreatedAt.Equal(before.CreatedAt) || !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Errorf("internal paste changed usage fields: before=%+v after=%+v", before, after)
	}

	if _, err := store.Update(map[string]any{
		"clipboard": map[string]any{"content": map[string]any{"copyPlain": true}},
	}); err != nil {
		t.Fatal(err)
	}
	writer.payload = clipboard.WritePayload{}
	if _, _, err := model.Activate(ctx, htmlID); err != nil {
		t.Fatal(err)
	}
	if writer.payload.Text != plain || writer.payload.HTML != "<b>rich text</b>" {
		t.Errorf("copy preference changed paste formatting = %+v", writer.payload)
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
		Kind:    database.KindText,
		Content: content,
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
		clicks := model.ClickBehavior()
		if clicks.SingleClick != tc.singleWrites || clicks.DoubleClick != tc.doubleWrites {
			t.Errorf("%s click behavior = %+v", tc.mode, clicks)
		}
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
