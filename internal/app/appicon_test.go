package app

import (
	"path/filepath"
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/storage"
)

func TestAppIconCacheLoadsOnce(t *testing.T) {
	store := storage.NewAppIconStore(filepath.Join(t.TempDir(), "resources"))
	cache := newAppIconCache(store)
	calls := 0
	cache.load = func(bundleID string) []byte {
		calls++
		if bundleID != "com.example.app" {
			t.Errorf("bundle = %q", bundleID)
		}
		return testPNG(t)
	}

	first := cache.File("com.example.app")
	second := cache.File("com.example.app")
	if calls != 1 {
		t.Errorf("load called %d times, want 1", calls)
	}
	if first == nil || second == nil || *first != *second {
		t.Errorf("icon files = %v, %v", first, second)
	}
	if !store.Exists(*first) {
		t.Error("cached icon was not stored")
	}
}

func TestAppIconCacheKeepsMissingIcon(t *testing.T) {
	store := storage.NewAppIconStore(t.TempDir())
	cache := newAppIconCache(store)
	calls := 0
	cache.load = func(string) []byte {
		calls++
		return nil
	}
	if file := cache.File("com.example.missing"); file != nil {
		t.Errorf("file = %v", file)
	}
	cache.File("com.example.missing")
	if calls != 1 {
		t.Errorf("load called %d times, want 1", calls)
	}
}
