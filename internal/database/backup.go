package database

import (
	"context"
	"fmt"
)

// ItemCounts are the manifest totals of a backup.
type ItemCounts struct {
	Items  int64
	Text   int64
	Images int64
	Files  int64
}

// Snapshot writes a consistent copy of the database to path. SQLite's
// VACUUM INTO includes committed WAL frames and leaves the live database
// untouched.
func (d *DB) Snapshot(ctx context.Context, path string) error {
	if _, err := d.sql.ExecContext(ctx, "VACUUM INTO ?", path); err != nil {
		return fmt.Errorf("快照数据库: %w", err)
	}
	return nil
}

// Counts totals the clipboard rows a backup contains.
func (d *DB) Counts(ctx context.Context) (ItemCounts, error) {
	var counts ItemCounts
	row := d.sql.QueryRowContext(ctx, `
		SELECT COUNT(*),
			COALESCE(SUM(kind = 'text'), 0),
			COALESCE(SUM(kind = 'image'), 0),
			COALESCE(SUM(kind = 'files'), 0)
		FROM clipboard_items`)
	if err := row.Scan(&counts.Items, &counts.Text, &counts.Images, &counts.Files); err != nil {
		return counts, fmt.Errorf("统计备份条目: %w", err)
	}
	return counts, nil
}
