package storage

import (
	"io/fs"
	"path/filepath"
)

// Usage is how many bytes the data root's parts take.
type Usage struct {
	Total    int64
	Database int64
	Resource int64
	Config   int64
}

// MeasureUsage sums the database, resources and config directories.
func MeasureUsage(paths *Paths) (Usage, error) {
	var usage Usage
	dirs := []struct {
		path string
		sum  *int64
	}{
		{paths.DB, &usage.Database},
		{paths.Resources, &usage.Resource},
		{paths.Config, &usage.Config},
	}
	for _, dir := range dirs {
		size, err := dirSize(dir.path)
		if err != nil {
			return usage, err
		}
		*dir.sum = size
	}
	total, err := dirSize(paths.Root)
	if err != nil {
		return usage, err
	}
	usage.Total = total
	return usage, nil
}

func dirSize(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			if path == root {
				return nil
			}
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	return total, err
}
