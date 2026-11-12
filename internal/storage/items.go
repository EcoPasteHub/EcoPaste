package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zeebo/blake3"

	"github.com/EcoPasteHub/EcoPaste/internal/domain"
)

// UpsertResult reports whether an incoming payload inserted or reused a row.
type UpsertResult struct {
	ID           string
	Deduplicated bool
}

// ContentHash is the stable deduplication fingerprint.
func ContentHash(kind domain.Kind, content string) string {
	sum := blake3.Sum256([]byte(string(kind) + ":" + content))
	return fmt.Sprintf("%x", sum[:])
}

// UpsertItem deduplicates by content_hash and otherwise inserts item.
func (d *Database) UpsertItem(ctx context.Context, item *domain.Item) (UpsertResult, error) {
	if item.ContentHash == "" {
		item.ContentHash = ContentHash(item.Kind, item.Content)
	}

	tx, err := d.sql.BeginTx(ctx, nil)
	if err != nil {
		return UpsertResult{}, err
	}
	defer tx.Rollback()

	var existing string
	err = tx.QueryRowContext(ctx,
		"SELECT id FROM clipboard_items WHERE content_hash = ? ORDER BY created_at DESC LIMIT 1",
		item.ContentHash,
	).Scan(&existing)
	if err == nil {
		if _, err := tx.ExecContext(ctx,
			"UPDATE clipboard_items SET use_count = use_count + 1, updated_at = ? WHERE id = ?",
			formatTimestamp(time.Now()), existing,
		); err != nil {
			return UpsertResult{}, err
		}
		if err := tx.Commit(); err != nil {
			return UpsertResult{}, err
		}
		return UpsertResult{ID: existing, Deduplicated: true}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return UpsertResult{}, err
	}

	if item.ID == "" {
		return UpsertResult{}, fmt.Errorf("clipboard item id is required")
	}
	if item.UseCount == 0 {
		item.UseCount = 1
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}
	if item.UpdatedAt.IsZero() {
		item.UpdatedAt = item.CreatedAt
	}
	if err := insertItem(ctx, tx, item); err != nil {
		return UpsertResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return UpsertResult{}, err
	}
	return UpsertResult{ID: item.ID}, nil
}

// InsertItem inserts an item without deduplication.
func (d *Database) InsertItem(ctx context.Context, item *domain.Item) error {
	if item.ContentHash == "" {
		item.ContentHash = ContentHash(item.Kind, item.Content)
	}
	return insertItem(ctx, d.sql, item)
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func insertItem(ctx context.Context, db execer, item *domain.Item) error {
	_, err := db.ExecContext(ctx, `INSERT INTO clipboard_items
		(id, kind, sub_kind, group_id, source_app_id, content, content_hash, search_text,
		 summary, file_types, size, width, height, use_count, is_favorite, is_pinned,
		 is_sensitive, platform, note, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.Kind, nullable(item.SubKind), nullable(item.GroupID),
		nullable(item.SourceAppID), item.Content, item.ContentHash,
		nullable(item.SearchText), nullable(item.Summary), nullable(item.FileTypes),
		nullableInt(item.Size), nullableInt(item.Width), nullableInt(item.Height),
		item.UseCount, boolInt(item.IsFavorite), boolInt(item.IsPinned),
		boolInt(item.IsSensitive), item.Platform, nullable(item.Note),
		formatTimestamp(item.CreatedAt), formatTimestamp(item.UpdatedAt),
	)
	return err
}

// ToggleFavorite flips the favorite flag and returns its new value.
func (d *Database) ToggleFavorite(ctx context.Context, id string) (bool, error) {
	var value bool
	err := d.sql.QueryRowContext(ctx,
		"UPDATE clipboard_items SET is_favorite = NOT is_favorite WHERE id = ? RETURNING is_favorite",
		id,
	).Scan(&value)
	return value, err
}

// TogglePinned flips the pinned flag and returns its new value.
func (d *Database) TogglePinned(ctx context.Context, id string) (bool, error) {
	var value bool
	err := d.sql.QueryRowContext(ctx,
		"UPDATE clipboard_items SET is_pinned = NOT is_pinned WHERE id = ? RETURNING is_pinned",
		id,
	).Scan(&value)
	return value, err
}

// SetNote updates a note without refreshing updated_at.
func (d *Database) SetNote(ctx context.Context, id string, note *string) error {
	_, err := d.sql.ExecContext(ctx,
		"UPDATE clipboard_items SET note = ? WHERE id = ?", nullablePtr(note), id)
	return err
}

// SetGroup updates grouping without refreshing updated_at.
func (d *Database) SetGroup(ctx context.Context, id string, groupID *string) error {
	_, err := d.sql.ExecContext(ctx,
		"UPDATE clipboard_items SET group_id = ? WHERE id = ?", nullablePtr(groupID), id)
	return err
}

// IncrementUseCount refreshes reuse metadata.
func (d *Database) IncrementUseCount(ctx context.Context, id string) error {
	_, err := d.sql.ExecContext(ctx,
		"UPDATE clipboard_items SET use_count = use_count + 1, updated_at = ? WHERE id = ?",
		formatTimestamp(time.Now()), id)
	return err
}

// DeleteItem removes one row and returns an orphaned image file name.
func (d *Database) DeleteItem(ctx context.Context, id string) (string, error) {
	var kind string
	var content string
	err := d.sql.QueryRowContext(ctx,
		"DELETE FROM clipboard_items WHERE id = ? RETURNING kind, content", id,
	).Scan(&kind, &content)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if domain.Kind(kind) != domain.KindImage {
		return "", nil
	}
	return content, nil
}

func formatTimestamp(value time.Time) string {
	return value.UTC().Format("2006-01-02T15:04:05.000000+00:00")
}

func nullable(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullablePtr(value *string) any {
	if value == nil || *value == "" {
		return nil
	}
	return *value
}

func nullableInt(value int64) any {
	if value == 0 {
		return nil
	}
	return value
}
