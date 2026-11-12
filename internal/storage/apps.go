package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/domain"
)

// UpsertApp refreshes name/icon while preserving created_at.
func (d *Database) UpsertApp(ctx context.Context, app domain.App) error {
	now := formatTimestamp(time.Now())
	result, err := d.sql.ExecContext(ctx,
		"UPDATE clipboard_apps SET name = ?, icon_file = ?, updated_at = ? WHERE id = ?",
		app.Name, nullable(app.IconFile), now, app.ID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected > 0 {
		return nil
	}
	if app.CreatedAt.IsZero() {
		app.CreatedAt = time.Now()
	}
	_, err = d.sql.ExecContext(ctx, `INSERT INTO clipboard_apps
		(id, name, icon_file, platform, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		app.ID, app.Name, nullable(app.IconFile), app.Platform,
		formatTimestamp(app.CreatedAt), now)
	return err
}

// FindApp returns one source application.
func (d *Database) FindApp(ctx context.Context, id string) (domain.App, error) {
	var app domain.App
	var icon sql.NullString
	var createdAt, updatedAt string
	err := d.sql.QueryRowContext(ctx, `SELECT id, name, icon_file, platform,
		created_at, updated_at FROM clipboard_apps WHERE id = ?`, id,
	).Scan(&app.ID, &app.Name, &icon, &app.Platform, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return app, err
	}
	if err != nil {
		return app, err
	}
	app.IconFile = icon.String
	app.CreatedAt, err = parseTimestamp(createdAt)
	if err != nil {
		return app, err
	}
	app.UpdatedAt, err = parseTimestamp(updatedAt)
	return app, err
}

// ListApps returns all source applications ordered case-insensitively.
func (d *Database) ListApps(ctx context.Context) ([]domain.App, error) {
	rows, err := d.sql.QueryContext(ctx, `SELECT id, name, icon_file, platform,
		created_at, updated_at FROM clipboard_apps
		ORDER BY name COLLATE NOCASE ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []domain.App
	for rows.Next() {
		var app domain.App
		var icon sql.NullString
		var createdAt, updatedAt string
		if err := rows.Scan(&app.ID, &app.Name, &icon, &app.Platform,
			&createdAt, &updatedAt); err != nil {
			return nil, err
		}
		app.IconFile = icon.String
		app.CreatedAt, err = parseTimestamp(createdAt)
		if err != nil {
			return nil, err
		}
		app.UpdatedAt, err = parseTimestamp(updatedAt)
		if err != nil {
			return nil, err
		}
		apps = append(apps, app)
	}
	return apps, rows.Err()
}

// DeleteUnreferencedApps removes only applications no item references.
func (d *Database) DeleteUnreferencedApps(ctx context.Context, ids []string) ([]string, error) {
	deleted := make([]string, 0, len(ids))
	for _, id := range ids {
		result, err := d.sql.ExecContext(ctx, `DELETE FROM clipboard_apps
			WHERE id = ? AND NOT EXISTS (
				SELECT 1 FROM clipboard_items WHERE source_app_id = ? LIMIT 1
			)`, id, id)
		if err != nil {
			return nil, err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return nil, err
		}
		if affected > 0 {
			deleted = append(deleted, id)
		}
	}
	return deleted, nil
}
