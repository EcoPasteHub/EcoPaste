package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/domain"
)

// ListGroups returns visible and hidden groups in stable display order.
func (d *Database) ListGroups(ctx context.Context) ([]domain.Group, error) {
	rows, err := d.sql.QueryContext(ctx, `SELECT id, name, icon, is_hidden,
		sort_order, created_at, updated_at FROM clipboard_groups
		ORDER BY sort_order ASC, created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []domain.Group
	for rows.Next() {
		var group domain.Group
		var hidden int
		var createdAt, updatedAt string
		if err := rows.Scan(&group.ID, &group.Name, &group.Icon, &hidden,
			&group.SortOrder, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		group.IsHidden = hidden != 0
		group.CreatedAt, err = parseTimestamp(createdAt)
		if err != nil {
			return nil, err
		}
		group.UpdatedAt, err = parseTimestamp(updatedAt)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, rows.Err()
}

// NextGroupSortOrder returns the next append position.
func (d *Database) NextGroupSortOrder(ctx context.Context) (int64, error) {
	var value sql.NullInt64
	err := d.sql.QueryRowContext(ctx,
		"SELECT MAX(sort_order) FROM clipboard_groups").Scan(&value)
	if err != nil {
		return 0, err
	}
	if !value.Valid {
		return 0, nil
	}
	return value.Int64 + 1, nil
}

// InsertGroup creates a group.
func (d *Database) InsertGroup(ctx context.Context, group domain.Group) error {
	_, err := d.sql.ExecContext(ctx, `INSERT INTO clipboard_groups
		(id, name, icon, is_hidden, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		group.ID, group.Name, group.Icon, boolInt(group.IsHidden), group.SortOrder,
		formatTimestamp(group.CreatedAt), formatTimestamp(group.UpdatedAt))
	return err
}

// UpdateGroup changes editable metadata.
func (d *Database) UpdateGroup(ctx context.Context, id, name, icon string, hidden bool) error {
	_, err := d.sql.ExecContext(ctx, `UPDATE clipboard_groups
		SET name = ?, icon = ?, is_hidden = ?, updated_at = ? WHERE id = ?`,
		name, icon, boolInt(hidden), formatTimestamp(time.Now()), id)
	return err
}

// UpdateGroupLayout rewrites order and visibility in one transaction.
func (d *Database) UpdateGroupLayout(ctx context.Context, orderedIDs, visibleIDs []string) error {
	tx, err := d.sql.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	visible := make(map[string]bool, len(visibleIDs))
	for _, id := range visibleIDs {
		visible[id] = true
	}
	now := formatTimestamp(time.Now())
	for index, id := range orderedIDs {
		if _, err := tx.ExecContext(ctx, `UPDATE clipboard_groups
			SET sort_order = ?, is_hidden = ?, updated_at = ? WHERE id = ?`,
			index, boolInt(!visible[id]), now, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// DeleteGroup relies on ON DELETE SET NULL for item.group_id.
func (d *Database) DeleteGroup(ctx context.Context, id string) error {
	_, err := d.sql.ExecContext(ctx,
		"DELETE FROM clipboard_groups WHERE id = ?", id)
	return err
}

// FindGroup returns one group or sql.ErrNoRows.
func (d *Database) FindGroup(ctx context.Context, id string) (domain.Group, error) {
	var group domain.Group
	var hidden int
	var createdAt, updatedAt string
	err := d.sql.QueryRowContext(ctx, `SELECT id, name, icon, is_hidden,
		sort_order, created_at, updated_at FROM clipboard_groups WHERE id = ?`, id,
	).Scan(&group.ID, &group.Name, &group.Icon, &hidden, &group.SortOrder,
		&createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return group, err
	}
	if err != nil {
		return group, err
	}
	group.IsHidden = hidden != 0
	group.CreatedAt, err = parseTimestamp(createdAt)
	group.UpdatedAt, err = parseTimestamp(updatedAt)
	return group, err
}
