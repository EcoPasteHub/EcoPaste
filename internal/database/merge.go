package database

import (
	"context"
	"database/sql"
	"fmt"
)

// MergeResult reports how a backup merge changed the current database.
type MergeResult struct {
	Items   int64
	Skipped int64
}

// MergeBackup inserts backup rows that are missing from the current
// database. Existing IDs and (kind, content_hash) pairs keep their current
// metadata.
func (d *DB) MergeBackup(ctx context.Context, backup *DB) (MergeResult, error) {
	var result MergeResult
	if backup == nil {
		return result, fmt.Errorf("backup database is nil")
	}
	tx, err := d.sql.BeginTx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("begin backup merge: %w", err)
	}
	defer tx.Rollback()

	if err := mergeGroups(ctx, tx, backup); err != nil {
		return result, err
	}
	if err := mergeApps(ctx, tx, backup); err != nil {
		return result, err
	}
	if err := mergeFileIcons(ctx, tx, backup); err != nil {
		return result, err
	}
	if err := mergeItems(ctx, tx, backup, &result); err != nil {
		return result, err
	}
	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("commit backup merge: %w", err)
	}
	return result, nil
}

func mergeGroups(ctx context.Context, tx *sql.Tx, backup *DB) error {
	rows, err := backup.sql.QueryContext(ctx, `SELECT id, name, icon, is_hidden, sort_order,
		created_at, updated_at FROM clipboard_groups`)
	if err != nil {
		return fmt.Errorf("read backup groups: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			id, name, icon, createdAt, updatedAt string
			hidden                               bool
			sortOrder                            int64
		)
		if err := rows.Scan(&id, &name, &icon, &hidden, &sortOrder, &createdAt, &updatedAt); err != nil {
			return fmt.Errorf("scan backup group: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO clipboard_groups
			(id, name, icon, is_hidden, sort_order, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			id, name, icon, hidden, sortOrder, createdAt, updatedAt); err != nil {
			return fmt.Errorf("merge backup group: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read backup groups: %w", err)
	}
	return nil
}

func mergeApps(ctx context.Context, tx *sql.Tx, backup *DB) error {
	rows, err := backup.sql.QueryContext(ctx, `SELECT id, name, icon_file, platform,
		created_at, updated_at FROM clipboard_apps`)
	if err != nil {
		return fmt.Errorf("read backup apps: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			id, name, platform, createdAt, updatedAt string
			iconFile                                 sql.NullString
		)
		if err := rows.Scan(&id, &name, &iconFile, &platform, &createdAt, &updatedAt); err != nil {
			return fmt.Errorf("scan backup app: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO clipboard_apps
			(id, name, icon_file, platform, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)`,
			id, name, iconFile, platform, createdAt, updatedAt); err != nil {
			return fmt.Errorf("merge backup app: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read backup apps: %w", err)
	}
	return nil
}

func mergeFileIcons(ctx context.Context, tx *sql.Tx, backup *DB) error {
	rows, err := backup.sql.QueryContext(ctx, `SELECT cache_key, platform, icon_file,
		created_at, updated_at FROM file_type_icons`)
	if err != nil {
		return fmt.Errorf("read backup file icons: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var cacheKey, platform, iconFile, createdAt, updatedAt string
		if err := rows.Scan(&cacheKey, &platform, &iconFile, &createdAt, &updatedAt); err != nil {
			return fmt.Errorf("scan backup file icon: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO file_type_icons
			(cache_key, platform, icon_file, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?)`,
			cacheKey, platform, iconFile, createdAt, updatedAt); err != nil {
			return fmt.Errorf("merge backup file icon: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read backup file icons: %w", err)
	}
	return nil
}

func mergeItems(ctx context.Context, tx *sql.Tx, backup *DB, result *MergeResult) error {
	rows, err := backup.sql.QueryContext(ctx,
		"SELECT "+itemColumns+" FROM clipboard_items ORDER BY created_at ASC")
	if err != nil {
		return fmt.Errorf("read backup items: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return fmt.Errorf("scan backup item: %w", err)
		}
		var existing int
		err = tx.QueryRowContext(ctx,
			"SELECT 1 FROM clipboard_items WHERE kind = ? AND content_hash = ? LIMIT 1",
			item.Kind, item.ContentHash).Scan(&existing)
		switch {
		case err == nil:
			result.Skipped++
			continue
		case err != sql.ErrNoRows:
			return fmt.Errorf("check backup item: %w", err)
		}
		inserted, err := tx.ExecContext(ctx, insertItemSQL, itemArgs(item)...)
		if err != nil {
			return fmt.Errorf("merge backup item: %w", err)
		}
		affected, err := inserted.RowsAffected()
		if err != nil {
			return fmt.Errorf("check merged item: %w", err)
		}
		if affected == 0 {
			result.Skipped++
		} else {
			result.Items++
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read backup items: %w", err)
	}
	return nil
}
