package app

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/keystroke"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
	"github.com/EcoPasteHub/EcoPaste/internal/storage"
	"github.com/EcoPasteHub/EcoPaste/internal/ui"
)

func TestNeedsAccessibility(t *testing.T) {
	model := &historyModel{}
	if !model.NeedsAccessibility(keystroke.ErrPermission) {
		t.Error("permission error was not recognized")
	}
	if model.NeedsAccessibility(nil) {
		t.Error("nil error asked for permission")
	}
}

func TestDeletePolicyHonorsProtectionSettings(t *testing.T) {
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
	ctx := context.Background()
	favorite, _, err := db.Upsert(ctx, &database.Item{Kind: database.KindText, Content: "favorite", IsFavorite: true})
	if err != nil {
		t.Fatal(err)
	}
	pinned, _, err := db.Upsert(ctx, &database.Item{Kind: database.KindText, Content: "pinned", IsPinned: true})
	if err != nil {
		t.Fatal(err)
	}
	normal, _, err := db.Upsert(ctx, &database.Item{Kind: database.KindText, Content: "normal"})
	if err != nil {
		t.Fatal(err)
	}
	model := &historyModel{runtime: &clipboardRuntime{db: db}, store: store}

	page, err := model.List(ctx, ui.HistoryQuery{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	rows := map[string]ui.HistoryItem{}
	for _, row := range page.Items {
		rows[row.ID] = row
	}
	if rows[favorite].CanDelete || !rows[favorite].DeleteConfirm {
		t.Errorf("favorite row = %+v", rows[favorite])
	}
	if rows[pinned].CanDelete || !rows[pinned].DeleteConfirm {
		t.Errorf("pinned row = %+v", rows[pinned])
	}
	if !rows[normal].CanDelete || !rows[normal].DeleteConfirm {
		t.Errorf("normal row = %+v", rows[normal])
	}
	if err := model.Delete(ctx, favorite); err == nil {
		t.Error("protected favorite was deleted")
	}

	if _, err := store.Update(map[string]any{"clipboard": map[string]any{"content": map[string]any{
		"deleteFavoriteItems":   true,
		"deletePinnedItems":     true,
		"deleteFavoriteConfirm": false,
	}}}); err != nil {
		t.Fatal(err)
	}
	page, err = model.List(ctx, ui.HistoryQuery{Filter: "favorite", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range page.Items {
		if row.ID == favorite && (!row.CanDelete || row.DeleteConfirm) {
			t.Errorf("favorite in favorite group = %+v", row)
		}
	}
	page, err = model.List(ctx, ui.HistoryQuery{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range page.Items {
		if row.ID == favorite && row.CanDelete {
			t.Errorf("favorite outside favorite group = %+v", row)
		}
	}
}

func TestHistoryItemRedactsSensitiveSummary(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	summary := "sk-" + strings.Repeat("z", 32)
	model := &historyModel{store: store}
	row := model.historyItem(context.Background(), database.Item{Kind: database.KindText, Summary: &summary, IsSensitive: true})
	if row.Title == summary || !strings.Contains(row.Title, "*") {
		t.Errorf("title was not redacted: %q", row.Title)
	}

	if _, err := store.Update(map[string]any{
		"clipboard": map[string]any{"sensitive": map[string]any{"redactSecrets": false}},
	}); err != nil {
		t.Fatal(err)
	}
	row = model.historyItem(context.Background(), database.Item{Kind: database.KindText, Summary: &summary, IsSensitive: true})
	if row.Title != summary {
		t.Errorf("title = %q, want full summary", row.Title)
	}
}

func TestHistoryItemIncludesSourceAppIcon(t *testing.T) {
	root := t.TempDir()
	db, err := database.Open(filepath.Join(root, "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	icons := storage.NewAppIconStore(filepath.Join(root, "resources"))
	iconFile, err := icons.StorePNG(testPNG(t))
	if err != nil {
		t.Fatal(err)
	}
	source := "com.example.editor"
	if err := db.EnsureApp(context.Background(), source, "Editor", &iconFile); err != nil {
		t.Fatal(err)
	}
	summary := "hello"
	if _, _, err := db.Upsert(context.Background(), &database.Item{
		Kind:        database.KindText,
		Content:     summary,
		Summary:     &summary,
		SourceAppID: &source,
		Platform:    "macos",
	}); err != nil {
		t.Fatal(err)
	}
	page, err := db.Page(context.Background(), database.Query{})
	if err != nil {
		t.Fatal(err)
	}
	model := &historyModel{runtime: &clipboardRuntime{db: db, appIcons: icons}}
	row := model.historyItem(context.Background(), page.Items[0])
	if row.SourceAppIcon != icons.Path(iconFile) {
		t.Errorf("source icon = %q", row.SourceAppIcon)
	}
}

func TestPreviewRedactsSensitiveText(t *testing.T) {
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
	secret := "sk-" + strings.Repeat("z", 32)
	item := &database.Item{
		Kind:        database.KindText,
		Content:     "token: " + secret,
		Summary:     &secret,
		IsSensitive: true,
		Platform:    "macos",
	}
	id, _, err := db.Upsert(context.Background(), item)
	if err != nil {
		t.Fatal(err)
	}
	model := &historyModel{
		runtime: &clipboardRuntime{db: db, images: storage.NewImageStore(filepath.Join(root, "resources"))},
		store:   store,
	}
	preview, err := model.Preview(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.Enabled {
		t.Fatal("preview was disabled")
	}
	if strings.Contains(preview.Text, secret) || strings.Contains(preview.Title, secret) {
		t.Errorf("preview leaked secret: %+v", preview)
	}

	if _, err := store.Update(map[string]any{
		"clipboard": map[string]any{"preview": map[string]any{"spaceEnabled": false}},
	}); err != nil {
		t.Fatal(err)
	}
	preview, err = model.Preview(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Enabled {
		t.Error("preview stayed enabled after the setting was turned off")
	}
}

func TestDeleteRemovesImageFile(t *testing.T) {
	root := t.TempDir()
	db, err := database.Open(filepath.Join(root, "db", "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	images := storage.NewImageStore(filepath.Join(root, "resources"))
	stored, err := images.StorePNG(testPNG(t))
	if err != nil {
		t.Fatal(err)
	}
	item := &database.Item{
		ID:       database.NewID(),
		Kind:     database.KindImage,
		Content:  stored.FileName,
		Platform: "macos",
	}
	id, _, err := db.Upsert(context.Background(), item)
	if err != nil {
		t.Fatal(err)
	}

	model := &historyModel{runtime: &clipboardRuntime{db: db, images: images}}
	if err := model.Delete(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(images.OriginPath(stored.FileName)); !os.IsNotExist(err) {
		t.Errorf("image still exists: %v", err)
	}
	found, err := db.FindByID(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if found != nil {
		t.Errorf("item %s still exists", id)
	}
}

func TestDeleteUnknownItemIsNoop(t *testing.T) {
	root := t.TempDir()
	db, err := database.Open(filepath.Join(root, "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	model := &historyModel{runtime: &clipboardRuntime{db: db, images: storage.NewImageStore(root)}}
	if err := model.Delete(context.Background(), "missing"); err != nil {
		t.Fatalf("delete missing item: %v", err)
	}
}

func testPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
