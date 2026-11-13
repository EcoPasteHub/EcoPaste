package storage

import "path/filepath"

// AppIconStore keeps application icons as flat PNG files named by their
// content hash. Unlike clipboard images, the number of distinct icons is
// bounded by the applications the user copies from, so sharding would add
// complexity without a practical benefit.
type AppIconStore struct {
	*pngIconStore
}

// AppIconMaxEdge is the square edge used for cached application icons.
const AppIconMaxEdge = 128

// NewAppIconStore returns the app icon store under a resources directory.
func NewAppIconStore(resources string) *AppIconStore {
	return &AppIconStore{pngIconStore: newPNGIconStore(
		filepath.Join(resources, "app-icons"), "app icon")}
}
