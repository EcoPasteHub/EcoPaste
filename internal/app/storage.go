package app

import (
	"context"
	"fmt"

	"github.com/EcoPasteHub/EcoPaste/internal/storage"
)

// storageUsage reports how much the data root and its main parts take.
func (r *clipboardRuntime) storageUsage() (storage.Usage, error) {
	return storage.MeasureUsage(r.paths)
}

// cleanCache removes images and app icons no history row references. It
// returns how many files and bytes it freed.
func (r *clipboardRuntime) cleanCache(ctx context.Context) (int, int64, error) {
	imageRefs, err := r.db.ImageFiles(ctx)
	if err != nil {
		return 0, 0, err
	}
	imageFiles, imageBytes, err := r.images.RemoveUnreferenced(imageRefs)
	if err != nil {
		return imageFiles, imageBytes, err
	}
	iconRefs, err := r.db.AppIconFiles(ctx)
	if err != nil {
		return imageFiles, imageBytes, err
	}
	iconFiles, iconBytes, err := r.appIcons.RemoveUnreferenced(iconRefs)
	if err != nil {
		return imageFiles + iconFiles, imageBytes + iconBytes, err
	}
	if r.fileIcons == nil {
		return imageFiles + iconFiles, imageBytes + iconBytes, nil
	}
	fileIconRefs, err := r.db.FileIconFiles(ctx)
	if err != nil {
		return imageFiles + iconFiles, imageBytes + iconBytes, err
	}
	fileIconFiles, fileIconBytes, err := r.fileIcons.RemoveUnreferenced(fileIconRefs)
	return imageFiles + iconFiles + fileIconFiles, imageBytes + iconBytes + fileIconBytes, err
}

func formatBytes(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	value := float64(size)
	for _, unit := range []string{"KB", "MB", "GB", "TB"} {
		value /= 1024
		if value < 1024 {
			return fmt.Sprintf("%.1f %s", value, unit)
		}
	}
	return fmt.Sprintf("%.1f PB", value/1024)
}
