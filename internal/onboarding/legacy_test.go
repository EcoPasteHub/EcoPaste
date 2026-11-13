package onboarding

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
	"github.com/EcoPasteHub/EcoPaste/internal/storage"
	_ "modernc.org/sqlite"
)

type legacyFixture struct {
	dir     string
	image   []byte
	created time.Time
}

func newLegacyFixture(t *testing.T, dir string) legacyFixture {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "images"), 0o755); err != nil {
		t.Fatal(err)
	}
	var imageBytes bytes.Buffer
	source := image.NewRGBA(image.Rect(0, 0, 4, 3))
	for x := 0; x < 4; x++ {
		for y := 0; y < 3; y++ {
			source.Set(x, y, color.RGBA{R: 20, G: 40, B: 60, A: 255})
		}
	}
	if err := png.Encode(&imageBytes, source); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "images", "pic.png"), imageBytes.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite", filepath.Join(dir, "EcoPaste.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE history (
		id TEXT PRIMARY KEY,
		type TEXT,
		value TEXT,
		favorite INTEGER,
		createTime TEXT,
		note TEXT,
		width INTEGER,
		height INTEGER
	)`); err != nil {
		t.Fatal(err)
	}
	rows := []struct {
		id, kind, value string
		favorite        int
		created, note   string
	}{
		{"1", "text", "hello legacy", 0, "2024-01-02 03:04:05", "saved note"},
		{"2", "text", "favorite text", 1, "2024-01-03 03:04:05", ""},
		{"3", "files", `["/tmp/a.txt","/tmp/b.txt"]`, 0, "2024-01-04 03:04:05", ""},
		{"4", "image", "pic.png", 0, "2024-01-05 03:04:05", ""},
		{"5", "text", "   ", 0, "2024-01-06 03:04:05", ""},
		{"6", "text", "hello legacy", 0, "2024-01-07 03:04:05", ""},
		{"7", "rich", "unknown", 0, "2024-01-08 03:04:05", ""},
		{"8", "image", "missing.png", 0, "2024-01-09 03:04:05", ""},
	}
	for _, row := range rows {
		if _, err := db.Exec(
			"INSERT INTO history (id, type, value, favorite, createTime, note, width, height) VALUES (?, ?, ?, ?, ?, ?, 4, 3)",
			row.id, row.kind, row.value, row.favorite, row.created, row.note); err != nil {
			t.Fatal(err)
		}
	}
	created, err := time.ParseInLocation("2006-01-02 15:04:05", rows[0].created, time.Local)
	if err != nil {
		t.Fatal(err)
	}
	return legacyFixture{dir: dir, image: imageBytes.Bytes(), created: created.UTC()}
}

func newLegacyImporter(t *testing.T, base, mode string) (*LegacyImporter, *database.DB, *storage.ImageStore) {
	t.Helper()
	root := t.TempDir()
	db, err := database.Open(filepath.Join(root, "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	images := storage.NewImageStore(filepath.Join(root, "resources"))
	return &LegacyImporter{
		DB:       db,
		Images:   images,
		Settings: func() settings.Clipboard { return settings.Default().Clipboard },
		Base:     base,
		Mode:     mode,
	}, db, images
}

func TestLegacyDetectFindsLargestDatabase(t *testing.T) {
	base := t.TempDir()
	legacy := newLegacyFixture(t, filepath.Join(base, "com.ayangweb.EcoPaste"))
	custom := filepath.Join(t.TempDir(), "EcoPasteData")
	customFixture := newLegacyFixture(t, custom)
	customDB, err := sql.Open("sqlite", filepath.Join(customFixture.dir, "EcoPaste.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := customDB.Exec(`INSERT INTO history (id, type, value, favorite, createTime, note, width, height)
		VALUES ('9', 'text', 'custom only', 0, '2024-01-10 03:04:05', '', 0, 0)`); err != nil {
		t.Fatal(err)
	}
	customDB.Close()
	if _, err := os.Stat(legacy.dir); err != nil {
		t.Fatal(err)
	}
	store, err := os.Create(filepath.Join(base, "com.ayangweb.EcoPaste", ".store.json"))
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"globalStore": map[string]any{"env": map[string]any{"saveDataDir": customFixture.dir}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Write(payload); err != nil {
		t.Fatal(err)
	}
	store.Close()

	importer, _, _ := newLegacyImporter(t, base, storage.Prod)
	detection, err := importer.Detect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !detection.Found || detection.Path != customFixture.dir {
		t.Fatalf("detection = %+v", detection)
	}
	if detection.ImportableItemCount != 9 || detection.NormalItemCount != 8 || detection.FavoriteItemCount != 1 {
		t.Errorf("counts = %d/%d/%d", detection.ImportableItemCount, detection.NormalItemCount, detection.FavoriteItemCount)
	}
}

func TestLegacyImportNormalRows(t *testing.T) {
	base := t.TempDir()
	fixture := newLegacyFixture(t, filepath.Join(base, "com.ayangweb.EcoPaste"))
	importer, db, images := newLegacyImporter(t, base, storage.Prod)

	result, err := importer.Import(context.Background(), []LegacyImportSelection{LegacyNormal})
	if err != nil {
		t.Fatal(err)
	}
	if result.Imported != 3 || result.Skipped != 4 || result.ImportedNormal != 3 || result.ImportedFavorite != 0 {
		t.Fatalf("result = %+v", result)
	}

	var count int
	if err := db.SQL().QueryRow("SELECT COUNT(*) FROM clipboard_items").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Errorf("items = %d", count)
	}

	var note sql.NullString
	var isFavorite bool
	var createdAt string
	if err := db.SQL().QueryRow(
		"SELECT note, is_favorite, created_at FROM clipboard_items WHERE content = ?", "hello legacy").
		Scan(&note, &isFavorite, &createdAt); err != nil {
		t.Fatal(err)
	}
	if note.String != "saved note" || isFavorite {
		t.Errorf("note/favorite = %q/%v", note.String, isFavorite)
	}
	stored, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		t.Fatal(err)
	}
	if !stored.Equal(fixture.created) {
		t.Errorf("created at %s, want %s", stored, fixture.created)
	}

	var imageName string
	var width, height int64
	if err := db.SQL().QueryRow(
		"SELECT content, width, height FROM clipboard_items WHERE kind = ?", string(database.KindImage)).
		Scan(&imageName, &width, &height); err != nil {
		t.Fatal(err)
	}
	if width != 4 || height != 3 {
		t.Errorf("image size = %dx%d", width, height)
	}
	data, err := os.ReadFile(images.OriginPath(imageName))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, fixture.image) {
		t.Error("stored image differs from fixture")
	}
}

func TestLegacyImportFavoritesAndDedupe(t *testing.T) {
	base := t.TempDir()
	newLegacyFixture(t, filepath.Join(base, "EcoPaste"))
	importer, db, _ := newLegacyImporter(t, base, storage.Prod)

	result, err := importer.Import(context.Background(), []LegacyImportSelection{LegacyFavorite})
	if err != nil {
		t.Fatal(err)
	}
	if result.Imported != 1 || result.ImportedFavorite != 1 {
		t.Fatalf("result = %+v", result)
	}
	var favorite bool
	if err := db.SQL().QueryRow(
		"SELECT is_favorite FROM clipboard_items WHERE content = ?", "favorite text").Scan(&favorite); err != nil {
		t.Fatal(err)
	}
	if !favorite {
		t.Error("favorite flag not imported")
	}
	again, err := importer.Import(context.Background(), []LegacyImportSelection{LegacyFavorite})
	if err != nil {
		t.Fatal(err)
	}
	if again.Imported != 0 || again.Skipped != 1 {
		t.Errorf("second import = %+v", again)
	}
}

func TestLegacyImportRejectsEmptySelection(t *testing.T) {
	importer, _, _ := newLegacyImporter(t, t.TempDir(), storage.Prod)
	if _, err := importer.Import(context.Background(), nil); err == nil {
		t.Fatal("empty selection accepted")
	}
}
