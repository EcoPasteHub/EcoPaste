package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/storage"
)

func TestFileIconCacheReusesType(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	firstPath := filepath.Join(dir, "report.PDF")
	secondPath := filepath.Join(dir, "notes.pdf")
	for _, path := range []string{firstPath, secondPath} {
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	store := storage.NewFileIconStore(filepath.Join(dir, "resources"))
	cache := newFileIconCache(db, store)
	calls := 0
	cache.load = func(string) []byte {
		calls++
		return testPNG(t)
	}

	first := cache.Path(ctx, firstPath, false)
	second := cache.Path(ctx, secondPath, false)
	if calls != 1 {
		t.Errorf("load called %d times, want 1", calls)
	}
	if first == "" || first != second {
		t.Errorf("icon paths = %q, %q", first, second)
	}
	if _, err := os.Stat(first); err != nil {
		t.Errorf("cached icon missing: %v", err)
	}
	iconFile, ok, err := db.FileIcon(ctx, ".pdf")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || iconFile != filepath.Base(first) {
		t.Errorf("database icon = %q, ok = %v", iconFile, ok)
	}
}

func TestFileIconCacheHandlesDirectoriesAndMissingFiles(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	store := storage.NewFileIconStore(filepath.Join(dir, "resources"))
	cache := newFileIconCache(db, store)
	calls := 0
	cache.load = func(string) []byte {
		calls++
		return testPNG(t)
	}
	subdirs := []string{filepath.Join(dir, "folder"), filepath.Join(dir, "other")}
	for _, subdir := range subdirs {
		if err := os.Mkdir(subdir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	first := cache.Path(ctx, subdirs[0], true)
	second := cache.Path(ctx, subdirs[1], true)
	if first == "" || first != second {
		t.Errorf("directory icon paths = %q, %q", first, second)
	}
	if calls != 1 {
		t.Errorf("load calls = %d, want 1", calls)
	}
	if got := cache.Path(ctx, filepath.Join(dir, "missing.pdf"), false); got != "" {
		t.Errorf("missing file icon = %q", got)
	}
	if calls != 1 {
		t.Errorf("missing path should not load an icon")
	}
}

func TestFileIconCacheKey(t *testing.T) {
	cases := []struct {
		path string
		dir  bool
		want string
	}{
		{"/tmp/Report.PDF", false, ".pdf"},
		{"/tmp/Makefile", false, "Makefile"},
		{"/tmp/folder", true, "<dir>"},
	}
	for _, tc := range cases {
		if got := fileIconCacheKey(tc.path, tc.dir); got != tc.want {
			t.Errorf("key(%q, %v) = %q, want %q", tc.path, tc.dir, got, tc.want)
		}
	}
}
