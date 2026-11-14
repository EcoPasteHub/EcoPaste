package database

import (
	"context"
	"path/filepath"
	"testing"
)

func TestFileIconCacheRoundTrip(t *testing.T) {
	ctx := context.Background()
	db, err := Open(filepath.Join(t.TempDir(), "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, ok, err := db.FileIcon(ctx, ".pdf"); err != nil || ok {
		t.Fatalf("initial lookup: ok=%v err=%v", ok, err)
	}
	if err := db.UpsertFileIcon(ctx, ".pdf", "abc.png"); err != nil {
		t.Fatal(err)
	}
	file, ok, err := db.FileIcon(ctx, ".pdf")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || file != "abc.png" {
		t.Fatalf("file = %q, ok = %v", file, ok)
	}
	if err := db.UpsertFileIcon(ctx, ".pdf", "def.png"); err != nil {
		t.Fatal(err)
	}
	file, _, err = db.FileIcon(ctx, ".pdf")
	if err != nil {
		t.Fatal(err)
	}
	if file != "def.png" {
		t.Errorf("updated file = %q", file)
	}
	files, err := db.FileIconFiles(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := files["def.png"]; !ok || len(files) != 1 {
		t.Errorf("referenced files = %v", files)
	}
}
