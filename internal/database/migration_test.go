package database

import (
	"context"
	"database/sql"
	"encoding/hex"
	"path/filepath"
	"testing"
)

func TestOpenRecordsLegacyMigrationBaseline(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	var (
		version     int64
		description string
		success     bool
		checksum    []byte
		execution   int64
	)
	err = db.SQL().QueryRowContext(context.Background(), `
SELECT version, description, success, checksum, execution_time
FROM _sqlx_migrations WHERE version = 1`).
		Scan(&version, &description, &success, &checksum, &execution)
	if err != nil {
		t.Fatal(err)
	}
	want, err := hex.DecodeString(legacyMigrationChecksum)
	if err != nil {
		t.Fatal(err)
	}
	if version != 1 || description != "init" || !success || execution != 0 {
		t.Errorf("baseline = %d, %q, %v, %d", version, description, success, execution)
	}
	if string(checksum) != string(want) {
		t.Errorf("checksum = %x", checksum)
	}
}

func TestOpenAddsBaselineToDatabaseWithoutMigrationTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "clipboard.db")
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`CREATE TABLE clipboard_items (id TEXT PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}

	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	var count int
	if err := db.SQL().QueryRow(`SELECT COUNT(*) FROM _sqlx_migrations WHERE version = 1 AND success = 1`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("baseline rows = %d, want 1", count)
	}
}

func TestOpenKeepsExistingMigrationMetadata(t *testing.T) {
	path := filepath.Join(t.TempDir(), "clipboard.db")
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`CREATE TABLE clipboard_items (id TEXT PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(sqlxMigrationsSchema); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`
INSERT INTO _sqlx_migrations (version, description, success, checksum, execution_time)
VALUES (1, 'init', 1, X'0102', 42)`); err != nil {
		t.Fatal(err)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}

	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	var (
		checksum  []byte
		execution int64
	)
	if err := db.SQL().QueryRow(`SELECT checksum, execution_time FROM _sqlx_migrations WHERE version = 1`).
		Scan(&checksum, &execution); err != nil {
		t.Fatal(err)
	}
	if string(checksum) != "\x01\x02" || execution != 42 {
		t.Errorf("metadata = %x, %d", checksum, execution)
	}
}
