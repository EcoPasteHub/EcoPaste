package database

import (
	"context"
	"fmt"
)

// EnsureApp registers or refreshes a source application.
func (d *DB) EnsureApp(ctx context.Context, id, name string) error {
	if id == "" {
		return nil
	}
	const query = `INSERT INTO clipboard_apps (id, name, icon_file, platform, created_at, updated_at)
		VALUES (?, ?, NULL, 'macos', ?, ?)
		ON CONFLICT(id) DO UPDATE SET name = excluded.name, updated_at = excluded.updated_at
		WHERE clipboard_apps.name <> excluded.name`
	timestamp := now()
	if _, err := d.sql.ExecContext(ctx, query, id, name, timestamp, timestamp); err != nil {
		return fmt.Errorf("register source app: %w", err)
	}
	return nil
}
