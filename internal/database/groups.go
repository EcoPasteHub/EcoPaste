package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Group is one user-created clipboard group. The icon and hidden fields
// remain part of the published schema even when the native UI does not
// expose them yet.
type Group struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Icon      string    `json:"icon"`
	Hidden    bool      `json:"isHidden"`
	SortOrder int64     `json:"sortOrder"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ListGroups returns groups in their published display order.
func (d *DB) ListGroups(ctx context.Context) ([]Group, error) {
	rows, err := d.sql.QueryContext(ctx, `SELECT id, name, icon, is_hidden, sort_order,
		created_at, updated_at FROM clipboard_groups
		ORDER BY sort_order ASC, created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("list clipboard groups: %w", err)
	}
	defer rows.Close()

	groups := make([]Group, 0)
	for rows.Next() {
		var (
			group     Group
			createdAt string
			updatedAt string
		)
		if err := rows.Scan(&group.ID, &group.Name, &group.Icon, &group.Hidden,
			&group.SortOrder, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan clipboard group: %w", err)
		}
		group.CreatedAt = parseTime(createdAt)
		group.UpdatedAt = parseTime(updatedAt)
		groups = append(groups, group)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read clipboard groups: %w", err)
	}
	return groups, nil
}

// FindGroup returns one group by id.
func (d *DB) FindGroup(ctx context.Context, id string) (*Group, error) {
	var (
		group     Group
		createdAt string
		updatedAt string
	)
	err := d.sql.QueryRowContext(ctx, `SELECT id, name, icon, is_hidden, sort_order,
		created_at, updated_at FROM clipboard_groups WHERE id = ?`, id).
		Scan(&group.ID, &group.Name, &group.Icon, &group.Hidden, &group.SortOrder,
			&createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find clipboard group: %w", err)
	}
	group.CreatedAt = parseTime(createdAt)
	group.UpdatedAt = parseTime(updatedAt)
	return &group, nil
}

// CreateGroup appends a group after every existing group.
func (d *DB) CreateGroup(ctx context.Context, name, icon string) (Group, error) {
	tx, err := d.sql.BeginTx(ctx, nil)
	if err != nil {
		return Group{}, fmt.Errorf("begin create group: %w", err)
	}
	defer tx.Rollback()

	var maxOrder sql.NullInt64
	if err := tx.QueryRowContext(ctx, "SELECT MAX(sort_order) FROM clipboard_groups").Scan(&maxOrder); err != nil {
		return Group{}, fmt.Errorf("read group sort order: %w", err)
	}
	group := Group{
		ID:        NewID(),
		Name:      name,
		Icon:      icon,
		SortOrder: 0,
		CreatedAt: time.Now().UTC(),
	}
	if maxOrder.Valid {
		group.SortOrder = maxOrder.Int64 + 1
	}
	group.UpdatedAt = group.CreatedAt
	if _, err := tx.ExecContext(ctx, `INSERT INTO clipboard_groups
		(id, name, icon, is_hidden, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		group.ID, group.Name, group.Icon, group.Hidden, group.SortOrder,
		group.CreatedAt.Format(time.RFC3339Nano), group.UpdatedAt.Format(time.RFC3339Nano)); err != nil {
		return Group{}, fmt.Errorf("insert clipboard group: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Group{}, fmt.Errorf("commit clipboard group: %w", err)
	}
	return group, nil
}

// RenameGroup updates a group's visible name without touching its items.
func (d *DB) RenameGroup(ctx context.Context, id, name string) error {
	result, err := d.sql.ExecContext(ctx,
		"UPDATE clipboard_groups SET name = ?, updated_at = ? WHERE id = ?",
		name, now(), id)
	if err != nil {
		return fmt.Errorf("rename clipboard group: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check renamed clipboard group: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("clipboard group not found")
	}
	return nil
}

// DeleteGroup removes a group. The foreign key clears group_id on its
// items, so clipboard history survives.
func (d *DB) DeleteGroup(ctx context.Context, id string) error {
	if _, err := d.sql.ExecContext(ctx, "DELETE FROM clipboard_groups WHERE id = ?", id); err != nil {
		return fmt.Errorf("delete clipboard group: %w", err)
	}
	return nil
}
