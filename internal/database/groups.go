package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Group is one user-created clipboard group.
type Group struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ListGroups returns groups in creation order.
func (d *DB) ListGroups(ctx context.Context) ([]Group, error) {
	rows, err := d.sql.QueryContext(ctx, `SELECT id, name, created_at, updated_at
		FROM clipboard_groups ORDER BY created_at, id`)
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
		if err := rows.Scan(&group.ID, &group.Name, &createdAt, &updatedAt); err != nil {
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
	err := d.sql.QueryRowContext(ctx, `SELECT id, name, created_at, updated_at
		FROM clipboard_groups WHERE id = ?`, id).
		Scan(&group.ID, &group.Name, &createdAt, &updatedAt)
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
func (d *DB) CreateGroup(ctx context.Context, name string) (Group, error) {
	created := time.Now().UTC()
	group := Group{ID: NewID(), Name: name, CreatedAt: created, UpdatedAt: created}
	timestamp := formatTime(created)
	if _, err := d.sql.ExecContext(ctx, `INSERT INTO clipboard_groups
		(id, name, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		group.ID, group.Name, timestamp, timestamp); err != nil {
		return Group{}, fmt.Errorf("insert clipboard group: %w", err)
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
