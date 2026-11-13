package database

import (
	"context"
	"fmt"
)

// EnsureApp registers or refreshes a source application. A nil icon keeps
// any icon already stored for the application.
func (d *DB) EnsureApp(ctx context.Context, id, name string, iconFile *string) error {
	if id == "" {
		return nil
	}
	const query = `INSERT INTO clipboard_apps (id, name, icon_file, platform, created_at, updated_at)
		VALUES (?, ?, ?, 'macos', ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			icon_file = COALESCE(excluded.icon_file, clipboard_apps.icon_file),
			updated_at = excluded.updated_at
		WHERE clipboard_apps.name <> excluded.name
			OR (excluded.icon_file IS NOT NULL AND clipboard_apps.icon_file IS NOT excluded.icon_file)`
	timestamp := now()
	if _, err := d.sql.ExecContext(ctx, query, id, name, iconFile, timestamp, timestamp); err != nil {
		return fmt.Errorf("register source app: %w", err)
	}
	return nil
}

// AppIconFiles lists the icon file names referenced by known applications.
func (d *DB) AppIconFiles(ctx context.Context) (map[string]struct{}, error) {
	rows, err := d.sql.QueryContext(ctx,
		"SELECT icon_file FROM clipboard_apps WHERE icon_file IS NOT NULL AND icon_file <> ''")
	if err != nil {
		return nil, fmt.Errorf("list app icon files: %w", err)
	}
	defer rows.Close()

	files := map[string]struct{}{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan app icon file: %w", err)
		}
		files[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read app icon files: %w", err)
	}
	return files, nil
}
