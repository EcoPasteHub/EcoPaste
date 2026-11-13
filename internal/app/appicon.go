package app

import (
	"sync"

	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/EcoPasteHub/EcoPaste/internal/storage"
)

// appIconCache resolves an application icon once and stores it in the app
// icon directory. Missing icons are cached as nil so a failed AppKit lookup
// does not repeat on every pasteboard change.
type appIconCache struct {
	store *storage.AppIconStore
	load  func(string) []byte

	mu    sync.Mutex
	files map[string]*string
}

func newAppIconCache(store *storage.AppIconStore) *appIconCache {
	return &appIconCache{
		store: store,
		load:  macos.AppIconPNG,
		files: map[string]*string{},
	}
}

func (c *appIconCache) File(bundleID string) *string {
	if c == nil || c.store == nil || bundleID == "" {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if file, ok := c.files[bundleID]; ok {
		return file
	}
	var file *string
	if data := c.load(bundleID); len(data) > 0 {
		if resized, err := storage.DownscalePNG(data, storage.AppIconMaxEdge); err == nil {
			data = resized
		}
		if name, err := c.store.StorePNG(data); err == nil {
			file = &name
		}
	}
	c.files[bundleID] = file
	return file
}
