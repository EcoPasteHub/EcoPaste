package storage

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// Usage is how many bytes the data root's parts take.
type Usage struct {
	Total    int64
	Database int64
	Resource int64
	Config   int64
}

// MeasureUsage sums the database, resources, config and complete data root.
func MeasureUsage(paths *Paths) (Usage, error) {
	var usage Usage
	databasePrefix := usagePathPrefix(paths.DB)
	resourcePrefix := usagePathPrefix(paths.Resources)
	configPrefix := usagePathPrefix(paths.Config)
	err := filepath.WalkDir(paths.Root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			if path == paths.Root {
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
		size := info.Size()
		usage.Total += size
		if strings.HasPrefix(path, databasePrefix) {
			usage.Database += size
		}
		if strings.HasPrefix(path, resourcePrefix) {
			usage.Resource += size
		}
		if strings.HasPrefix(path, configPrefix) {
			usage.Config += size
		}
		return nil
	})
	return usage, err
}

func usagePathPrefix(root string) string {
	prefix := filepath.Clean(root)
	if strings.HasSuffix(prefix, string(filepath.Separator)) {
		return prefix
	}
	return prefix + string(filepath.Separator)
}
