package database

import (
	"context"
	"fmt"
	"time"
)

// App is one source application that appears on clipboard items.
type App struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	IconFile  *string   `json:"iconFile,omitempty"`
	Platform  string    `json:"platform"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

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

// FindApp returns one known source application.
func (d *DB) FindApp(ctx context.Context, id string) (*App, error) {
	const query = `SELECT id, name, icon_file, platform, created_at, updated_at
		FROM clipboard_apps WHERE id = ?`
	var (
		app      App
		iconFile *string
		created  string
		updated  string
	)
	err := d.sql.QueryRowContext(ctx, query, id).
		Scan(&app.ID, &app.Name, &iconFile, &app.Platform, &created, &updated)
	if err != nil {
		return nil, err
	}
	app.IconFile = iconFile
	app.CreatedAt = parseTime(created)
	app.UpdatedAt = parseTime(updated)
	return &app, nil
}

// ListApps returns every materialized source application ordered by name.
func (d *DB) ListApps(ctx context.Context) ([]App, error) {
	const query = `SELECT id, name, icon_file, platform, created_at, updated_at
		FROM clipboard_apps ORDER BY name COLLATE NOCASE ASC, id ASC`
	rows, err := d.sql.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list source apps: %w", err)
	}
	defer rows.Close()

	var apps []App
	for rows.Next() {
		var (
			app      App
			iconFile *string
			created  string
			updated  string
		)
		if err := rows.Scan(&app.ID, &app.Name, &iconFile, &app.Platform, &created, &updated); err != nil {
			return nil, fmt.Errorf("scan source app: %w", err)
		}
		app.IconFile = iconFile
		app.CreatedAt = parseTime(created)
		app.UpdatedAt = parseTime(updated)
		apps = append(apps, app)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read source apps: %w", err)
	}
	return apps, nil
}

// DeleteUnreferencedApps removes apps that no clipboard item points at and
// returns the ids that were actually deleted.
func (d *DB) DeleteUnreferencedApps(ctx context.Context, ids []string) ([]string, error) {
	var deleted []string
	for _, id := range ids {
		result, err := d.sql.ExecContext(ctx,
			`DELETE FROM clipboard_apps WHERE id = ?
			 AND NOT EXISTS (SELECT 1 FROM clipboard_items WHERE source_app_id = ? LIMIT 1)`,
			id, id)
		if err != nil {
			return deleted, fmt.Errorf("delete unreferenced source app: %w", err)
		}
		if affected, err := result.RowsAffected(); err == nil && affected > 0 {
			deleted = append(deleted, id)
		}
	}
	return deleted, nil
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
