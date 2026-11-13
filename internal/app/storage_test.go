package app

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/storage"
)

func testPNGBytes(t *testing.T, width, height int, shade uint8) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.Set(x, y, color.RGBA{R: shade, G: shade, B: shade, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestFormatBytes(t *testing.T) {
	cases := map[int64]string{
		512:                    "512 B",
		2048:                   "2.0 KB",
		1536 * 1024:            "1.5 MB",
		3 * 1024 * 1024 * 1024: "3.0 GB",
	}
	for size, want := range cases {
		if got := formatBytes(size); got != want {
			t.Errorf("formatBytes(%d) = %q, want %q", size, got, want)
		}
	}
}

func TestCleanCacheKeepsReferencedFiles(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	images := storage.NewImageStore(dir)
	icons := storage.NewAppIconStore(dir)
	fileIcons := storage.NewFileIconStore(dir)

	kept, err := images.StorePNG(testPNGBytes(t, 32, 32, 10))
	if err != nil {
		t.Fatal(err)
	}
	orphan, err := images.StorePNG(testPNGBytes(t, 24, 24, 20))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := db.Upsert(ctx, &database.Item{Kind: database.KindImage, Content: kept.FileName}); err != nil {
		t.Fatal(err)
	}
	keptIcon, err := icons.StorePNG(testPNGBytes(t, 16, 16, 30))
	if err != nil {
		t.Fatal(err)
	}
	orphanIcon, err := icons.StorePNG(testPNGBytes(t, 16, 16, 40))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.EnsureApp(ctx, "com.example.app", "Example", &keptIcon); err != nil {
		t.Fatal(err)
	}
	keptFileIcon, err := fileIcons.StorePNG(testPNGBytes(t, 16, 16, 50))
	if err != nil {
		t.Fatal(err)
	}
	orphanFileIcon, err := fileIcons.StorePNG(testPNGBytes(t, 16, 16, 60))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertFileIcon(ctx, ".pdf", keptFileIcon); err != nil {
		t.Fatal(err)
	}

	runtime := &clipboardRuntime{db: db, images: images, appIcons: icons, fileIcons: fileIcons}
	files, freed, err := runtime.cleanCache(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if files != 4 || freed <= 0 {
		t.Fatalf("removed %d files, %d bytes", files, freed)
	}
	if _, err := os.Stat(images.OriginPath(kept.FileName)); err != nil {
		t.Errorf("referenced image removed: %v", err)
	}
	if _, err := os.Stat(images.OriginPath(orphan.FileName)); !os.IsNotExist(err) {
		t.Errorf("orphan image kept: %v", err)
	}
	if !icons.Exists(keptIcon) {
		t.Error("referenced icon removed")
	}
	if icons.Exists(orphanIcon) {
		t.Error("orphan icon kept")
	}
	if !fileIcons.Exists(keptFileIcon) {
		t.Error("referenced file icon removed")
	}
	if fileIcons.Exists(orphanFileIcon) {
		t.Error("orphan file icon kept")
	}
}

func TestChangeDataDirCopiesLiveData(t *testing.T) {
	ctx := context.Background()
	base := t.TempDir()
	paths, err := storage.Resolve(base, storage.Dev)
	if err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(filepath.Join(paths.DB, "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, _, err := db.Upsert(ctx, &database.Item{
		Kind: database.KindText, Content: "relocate smoke", Platform: "macos",
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(paths.Config, "settings.json"), []byte(`{"general":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(paths.Resources, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(paths.Resources, "a.png"), []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(paths.State, "window-state.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(paths.State, "instance.lock"), []byte("123"), 0o644); err != nil {
		t.Fatal(err)
	}

	runtime := &clipboardRuntime{db: db, paths: paths}
	parent := t.TempDir()
	moved, err := runtime.changeDataDir(ctx, parent)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(parent, customDataDirName, storage.Dev)
	if moved.Root != want {
		t.Fatalf("root = %q, want %q", moved.Root, want)
	}
	resolved, err := storage.Resolve(base, storage.Dev)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Root != want {
		t.Errorf("manifest root = %q, want %q", resolved.Root, want)
	}
	if _, err := os.Stat(filepath.Join(moved.Resources, "a.png")); err != nil {
		t.Errorf("resource not copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(moved.Config, "settings.json")); err != nil {
		t.Errorf("settings not copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(moved.State, "window-state.json")); err != nil {
		t.Errorf("window state not copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(moved.State, "instance.lock")); !os.IsNotExist(err) {
		t.Errorf("instance lock was copied: %v", err)
	}
	copied, err := database.Open(filepath.Join(moved.DB, "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer copied.Close()
	counts, err := copied.Counts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if counts.Items != 1 {
		t.Errorf("copied counts = %+v", counts)
	}
	if _, err := runtime.changeDataDir(ctx, parent); err == nil {
		t.Error("existing target directory was accepted")
	}
}

func TestChangeDataDirRemovesTargetOnFailure(t *testing.T) {
	base := t.TempDir()
	paths, err := storage.Resolve(base, storage.Dev)
	if err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(filepath.Join(paths.DB, "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	runtime := &clipboardRuntime{db: db, paths: paths}
	parent := t.TempDir()
	if _, err := runtime.changeDataDir(context.Background(), parent); err == nil {
		t.Fatal("closed database snapshot was accepted")
	}
	target := filepath.Join(parent, customDataDirName, storage.Dev)
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Errorf("failed target was kept: %v", err)
	}
	resolved, err := storage.Resolve(base, storage.Dev)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Root != paths.Root {
		t.Errorf("manifest moved after failure: %q", resolved.Root)
	}
}
