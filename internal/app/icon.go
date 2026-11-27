package app

import (
	"github.com/EcoPasteHub/EcoPaste/internal/storage"
)

// iconStore persists one PNG icon and reports the file name it used.
type iconStore interface {
	StorePNG(data []byte) (string, error)
}

// storeIconPNG downscales an icon to maxEdge and persists it, returning nil
// when there is nothing to store.
func storeIconPNG(store iconStore, data []byte, maxEdge int) *string {
	if len(data) == 0 {
		return nil
	}
	if resized, err := storage.DownscalePNG(data, maxEdge); err == nil {
		data = resized
	}
	name, err := store.StorePNG(data)
	if err != nil {
		return nil
	}
	return &name
}
