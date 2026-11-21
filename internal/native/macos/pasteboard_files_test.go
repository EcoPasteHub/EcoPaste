//go:build darwin

package macos

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestFileURLPathDecodesRealPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a b&c.txt")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	raw := (&url.URL{Scheme: "file", Path: path}).String()
	if got := fileURLPath(raw); got != path {
		t.Fatalf("fileURLPath(%q) = %q, want %q", raw, got, path)
	}
}

func TestFileURLPathRejectsNonFileURL(t *testing.T) {
	if got := fileURLPath("https://example.com/a"); got != "" {
		t.Fatalf("fileURLPath(non file) = %q, want empty", got)
	}
	if got := fileURLPath(""); got != "" {
		t.Fatalf("fileURLPath(empty) = %q, want empty", got)
	}
}
