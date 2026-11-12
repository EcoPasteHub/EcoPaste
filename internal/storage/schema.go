package storage

import (
	"context"
	"crypto/sha512"
	_ "embed"
	"encoding/hex"
	"fmt"
	"time"
)

//go:embed schema.sql
var schemaSQL string

// sqlxChecksum is the SHA-384 checksum of src-tauri/migrations/0001_init.sql.
// Keeping it lets a Rust build recognize a database first created by Go.
const sqlxChecksum = "bfa656aa68eed66f8dab5bacb9efa4bafeb11713fd417239e05f7b4c5757330fae5eafeeb138771c6d18785670c44b05"

func (d *Database) createSchema(ctx context.Context) error {
	sum := sha512.Sum384([]byte(schemaSQL))
	if hex.EncodeToString(sum[:]) != sqlxChecksum {
		return fmt.Errorf("embedded schema differs from the sqlx migration checksum")
	}

	tx, err := d.sql.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, schemaSQL); err != nil {
		return fmt.Errorf("create EcoPaste schema: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `CREATE TABLE _sqlx_migrations (
		version BIGINT PRIMARY KEY,
		description TEXT NOT NULL,
		installed_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		success BOOLEAN NOT NULL,
		checksum BLOB NOT NULL,
		execution_time BIGINT NOT NULL
	)`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO _sqlx_migrations (version, description, installed_on, success, checksum, execution_time) VALUES (1, 'init', ?, 1, ?, 0)",
		time.Now().UTC().Format("2006-01-02 15:04:05"), sum[:],
	); err != nil {
		return err
	}
	return tx.Commit()
}
