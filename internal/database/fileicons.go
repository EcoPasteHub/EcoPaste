package database

import (
	"context"
	"database/sql"
	"fmt"
)

// FileIcon returns the cached icon file name for a file type, if present.
func (d *DB) FileIcon(ctx context.Context, cacheKey string) (string, bool, error) {
	var iconFile string
	err := d.sql.QueryRowContext(ctx,
		"SELECT icon_file FROM file_type_icons WHERE cache_key = ? AND platform = 'macos'",
		cacheKey).Scan(&iconFile)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read file icon: %w", err)
	}
	return iconFile, true, nil
}

// UpsertFileIcon records the file name for a file-type cache key.
func (d *DB) UpsertFileIcon(ctx context.Context, cacheKey, iconFile string) error {
	timestamp := now()
	_, err := d.sql.ExecContext(ctx, `INSERT INTO file_type_icons
		(cache_key, platform, icon_file, created_at, updated_at)
		VALUES (?, 'macos', ?, ?, ?)
		ON CONFLICT(cache_key, platform) DO UPDATE SET
			icon_file = excluded.icon_file,
			updated_at = excluded.updated_at`,
		cacheKey, iconFile, timestamp, timestamp)
	if err != nil {
		return fmt.Errorf("store file icon: %w", err)
	}
	return nil
}

// FileIconFiles lists icon file names still referenced by the cache table.
func (d *DB) FileIconFiles(ctx context.Context) (map[string]struct{}, error) {
	files, err := d.queryStringSet(ctx, "SELECT icon_file FROM file_type_icons")
	if err != nil {
		return nil, fmt.Errorf("list file icon files: %w", err)
	}
	return files, nil
}
