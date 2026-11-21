package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// CleanupResult reports what one cleanup pass removed.
type CleanupResult struct {
	Removed    int64
	ImageFiles []string
}

// CleanupHistory deletes unprotected items older than olderThan and, when
// maxCount is positive, keeps only the newest maxCount unprotected items.
func (d *DB) CleanupHistory(ctx context.Context, olderThan *time.Time, maxCount int) (CleanupResult, error) {
	var out CleanupResult
	if olderThan != nil {
		result, err := d.deleteReturning(ctx,
			`DELETE FROM clipboard_items
			 WHERE is_pinned = 0 AND is_favorite = 0 AND created_at < ?`,
			formatTime(*olderThan))
		if err != nil {
			return out, fmt.Errorf("cleanup history by retention: %w", err)
		}
		out.merge(result)
	}
	if maxCount > 0 {
		result, err := d.deleteReturning(ctx,
			`DELETE FROM clipboard_items WHERE id IN (
				SELECT id FROM clipboard_items
				WHERE is_pinned = 0 AND is_favorite = 0
				ORDER BY created_at DESC
				LIMIT -1 OFFSET ?
			 )`, int64(maxCount))
		if err != nil {
			return out, fmt.Errorf("cleanup history by max count: %w", err)
		}
		out.merge(result)
	}
	return out, nil
}

// ClearHistory deletes all items except the categories the caller keeps.
func (d *DB) ClearHistory(ctx context.Context, deleteFavorites, deletePinned bool) (CleanupResult, error) {
	var where []string
	if !deleteFavorites {
		where = append(where, "is_favorite = 0")
	}
	if !deletePinned {
		where = append(where, "is_pinned = 0")
	}
	query := "DELETE FROM clipboard_items"
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	result, err := d.deleteReturning(ctx, query)
	if err != nil {
		return CleanupResult{}, fmt.Errorf("clear history: %w", err)
	}
	return result, nil
}

func (d *DB) deleteReturning(ctx context.Context, query string, args ...any) (CleanupResult, error) {
	// Copy only image file names into SQLite's RETURNING buffer. A text or
	// file row may contain large content that cleanup never uses.
	rows, err := d.sql.QueryContext(ctx, query+" RETURNING CASE WHEN kind = 'image' THEN content END", args...)
	if err != nil {
		return CleanupResult{}, err
	}
	defer rows.Close()

	var out CleanupResult
	for rows.Next() {
		var image sql.NullString
		if err := rows.Scan(&image); err != nil {
			return out, err
		}
		out.Removed++
		if image.Valid {
			out.ImageFiles = append(out.ImageFiles, image.String)
		}
	}
	return out, rows.Err()
}

func (r *CleanupResult) merge(other CleanupResult) {
	r.Removed += other.Removed
	r.ImageFiles = append(r.ImageFiles, other.ImageFiles...)
}
