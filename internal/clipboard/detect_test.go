package clipboard

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
)

func TestDetectTextSubKind(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(file, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		text string
		want database.SubKind
		ok   bool
	}{
		{"https://example.com", database.SubURL, true},
		{"www.example.com", database.SubURL, true},
		{"ftp://host/file", database.SubURL, true},
		{"user@example.com", database.SubEmail, true},
		{"张三@example.com.cn", database.SubEmail, true},
		{"first.last+tag@example.com", database.SubEmail, true},
		{"#8B5CF6", database.SubColor, true},
		{"rgb(1, 2, 3)", database.SubColor, true},
		{"linear-gradient(90deg, #fff, #000)", database.SubColor, true},
		{file, database.SubPath, true},
		{"example.com", "", false},
		{"hello world", "", false},
		{"https://example.com/path with space", "", false},
		{"rgb(1,2,3);background:url(x)", "", false},
		{filepath.Join(dir, "missing.txt"), "", false},
		{"relative/path.txt", "", false},
	}
	for _, tt := range tests {
		got, ok := DetectTextSubKind(tt.text)
		if got != tt.want || ok != tt.ok {
			t.Errorf("DetectTextSubKind(%q) = (%q, %v), want (%q, %v)", tt.text, got, ok, tt.want, tt.ok)
		}
	}
}

func TestCaptureDetectsURLSubKind(t *testing.T) {
	cfg := settings.Default().Clipboard
	svc, db, _ := testService(t, cfg)
	id, _, err := svc.Capture(context.Background(), Payload{Text: "https://example.com"}, SourceApp{})
	if err != nil {
		t.Fatal(err)
	}
	item, err := db.FindByID(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if item.SubKind == nil || *item.SubKind != database.SubURL {
		t.Errorf("sub kind = %v, want url", item.SubKind)
	}
}
