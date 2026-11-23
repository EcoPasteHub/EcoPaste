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
	store    *storage.AppIconStore
	load     func(string) []byte
	loadPath func(string) []byte

	mu    sync.Mutex
	files map[string]*string
}

func newAppIconCache(store *storage.AppIconStore) *appIconCache {
	return &appIconCache{
		store:    store,
		load:     macos.AppIconPNG,
		loadPath: macos.AppIconForPath,
		files:    map[string]*string{},
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
	file := storeIconPNG(c.store, c.load(bundleID), storage.AppIconMaxEdge)
	c.files[bundleID] = file
	return file
}

// FileAtPath resolves an icon for an application that may not be running, using
// its bundle path. The result is cached under the bundle identifier.
func (c *appIconCache) FileAtPath(bundleID, path string) *string {
	if c == nil || c.store == nil || c.loadPath == nil || bundleID == "" || path == "" {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if file, ok := c.files[bundleID]; ok {
		return file
	}
	file := storeIconPNG(c.store, c.loadPath(path), storage.AppIconMaxEdge)
	c.files[bundleID] = file
	return file
}

func (c *appIconCache) Forget(bundleID string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	delete(c.files, bundleID)
	c.mu.Unlock()
}
