//go:build darwin

package macos

import (
	"os"
	"strings"
	"testing"
)

func TestResolveAppReadsBundleMetadata(t *testing.T) {
	path := "/System/Applications/Notes.app"
	if _, err := os.Stat(path); err != nil {
		t.Skip("Notes.app is not installed")
	}
	app, err := ResolveApp(path)
	if err != nil {
		t.Fatal(err)
	}
	if app.BundleID != "com.apple.Notes" {
		t.Errorf("bundle id = %q", app.BundleID)
	}
	if strings.TrimSpace(app.Name) == "" {
		t.Error("name is empty")
	}
	if app.Path != path {
		t.Errorf("path = %q", app.Path)
	}
}

func TestResolveAppRejectsNonBundle(t *testing.T) {
	file := t.TempDir() + "/plain.txt"
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveApp(file); err == nil {
		t.Fatal("non-app path was accepted")
	}
}

func TestAppIconForPath(t *testing.T) {
	path := "/System/Applications/Notes.app"
	if _, err := os.Stat(path); err != nil {
		t.Skip("Notes.app is not installed")
	}
	if png := AppIconForPath(path); len(png) == 0 {
		t.Fatal("icon is empty")
	}
}
