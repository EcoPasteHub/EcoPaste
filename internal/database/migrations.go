package database

import (
	"context"
	"encoding/hex"
	"fmt"
)

const legacyMigrationChecksum = "bfa656aa68eed66f8dab5bacb9efa4bafeb11713fd417239e05f7b4c5757330fae5eafeeb138771c6d18785670c44b05"

const sqlxMigrationsSchema = `
CREATE TABLE IF NOT EXISTS _sqlx_migrations (
    version        BIGINT PRIMARY KEY,
    description    TEXT NOT NULL,
    installed_on   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    success        BOOLEAN NOT NULL,
    checksum       BLOB NOT NULL,
    execution_time BIGINT NOT NULL
);`

// ensureSQLxBaseline records the published 0001 migration in SQLx's table.
// Legacy databases predate the Go schema and new databases must remain
// readable by tools that expect the original migration metadata.
func (d *DB) ensureSQLxBaseline(ctx context.Context) error {
	if _, err := d.sql.ExecContext(ctx, sqlxMigrationsSchema); err != nil {
		return fmt.Errorf("create legacy migration table: %w", err)
	}
	checksum, err := hex.DecodeString(legacyMigrationChecksum)
	if err != nil {
		return fmt.Errorf("decode legacy migration checksum: %w", err)
	}
	if _, err := d.sql.ExecContext(ctx, `
INSERT OR IGNORE INTO _sqlx_migrations
    (version, description, success, checksum, execution_time)
VALUES (?, ?, ?, ?, ?)`,
		1, "init", true, checksum, int64(0)); err != nil {
		return fmt.Errorf("record legacy migration baseline: %w", err)
	}
	return nil
}
