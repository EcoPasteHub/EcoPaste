package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/EcoPasteHub/EcoPaste/internal/storage"
)

const directoryFileIconKey = "<dir>"

// fileIconCache maps a file type to one cached Finder icon. Failed lookups
// are remembered in memory so scrolling does not repeatedly ask AppKit for
// a path that has no usable icon.
type fileIconCache struct {
	db    *database.DB
	store *storage.FileIconStore
	load  func(string) []byte

	mu    sync.Mutex
	files map[string]string
}

func newFileIconCache(db *database.DB, store *storage.FileIconStore) *fileIconCache {
	return &fileIconCache{
		db:    db,
		store: store,
		load:  macos.FileIconPNG,
		files: map[string]string{},
	}
}

// Path returns the cached icon for a path, extracting and persisting one
// on first use. Missing paths get no icon so stale items do not display a
// misleading generic document.
func (c *fileIconCache) Path(ctx context.Context, path string, isDir bool) string {
	if c == nil || c.store == nil || c.db == nil || path == "" {
		return ""
	}
	if ctx.Err() != nil {
		return ""
	}
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	key := fileIconCacheKey(path, isDir)

	c.mu.Lock()
	defer c.mu.Unlock()
	if name, ok := c.files[key]; ok {
		if name == "" || c.store.Exists(name) {
			return c.iconPath(name)
		}
		delete(c.files, key)
	}
	if name, ok, err := c.db.FileIcon(ctx, key); err == nil && ok {
		if c.store.Exists(name) {
			c.files[key] = name
			return c.iconPath(name)
		}
	}

	data := c.load(path)
	if len(data) > 0 {
		if resized, err := storage.DownscalePNG(data, storage.FileIconMaxEdge); err == nil {
			data = resized
		}
		if name, err := c.store.StorePNG(data); err == nil {
			if err := c.db.UpsertFileIcon(ctx, key, name); err == nil {
				c.files[key] = name
				return c.iconPath(name)
			}
		}
	}
	c.files[key] = ""
	return ""
}

func (c *fileIconCache) iconPath(name string) string {
	if name == "" {
		return ""
	}
	return c.store.Path(name)
}

func fileIconCacheKey(path string, isDir bool) string {
	if isDir {
		return directoryFileIconKey
	}
	ext := filepath.Ext(path)
	if ext == "" {
		return filepath.Base(path)
	}
	return strings.ToLower(ext)
}
