package database

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

// DB owns the SQLite connection. One connection serializes writes, which
// keeps the watcher, cleanup and UI on a single writer without lock
// contention.
type DB struct {
	sql  *sql.DB
	path string
}

const schemaVersion = 1

// Open opens the clipboard database, initializing a new database when empty.
func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create database dir: %w", err)
	}
	dsn := databaseURI(path) +
		"?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=synchronous(NORMAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	d := &DB{sql: db, path: path}
	if err := d.ensureSchema(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable database WAL: %w", err)
	}
	return d, nil
}

// OpenSnapshot opens an existing backup database without changing its schema or contents.
func OpenSnapshot(ctx context.Context, path string) (*DB, error) {
	db, err := sql.Open("sqlite", databaseURI(path)+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("open database snapshot: %w", err)
	}
	var version int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		db.Close()
		return nil, fmt.Errorf("read database snapshot schema version: %w", err)
	}
	if version != schemaVersion {
		db.Close()
		return nil, fmt.Errorf("unsupported database schema version: %d", version)
	}
	return &DB{sql: db, path: path}, nil
}

func databaseURI(path string) string {
	return (&url.URL{Scheme: "file", Path: path}).String()
}

// Path returns the database file's location.
func (d *DB) Path() string { return d.path }

// Close flushes and closes the database.
func (d *DB) Close() error { return d.sql.Close() }

// SQL exposes the pool for modules that own their own queries, such as
// groups and source apps.
func (d *DB) SQL() *sql.DB { return d.sql }

// queryStringSet reads one text column into a set, collapsing duplicates.
func (d *DB) queryStringSet(ctx context.Context, query string, args ...any) (map[string]struct{}, error) {
	rows, err := d.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	values := map[string]struct{}{}
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		values[value] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func (d *DB) ensureSchema(ctx context.Context) error {
	tx, err := d.sql.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin database initialization: %w", err)
	}
	defer tx.Rollback()
	var version int
	if err := tx.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("read database schema version: %w", err)
	}
	if version == schemaVersion {
		return nil
	}
	if version != 0 {
		return fmt.Errorf("unsupported database schema version: %d", version)
	}
	var hasSchema bool
	if err := tx.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM sqlite_schema)").Scan(&hasSchema); err != nil {
		return fmt.Errorf("inspect database schema: %w", err)
	}
	if hasSchema {
		return fmt.Errorf("database schema has no version")
	}
	if _, err := tx.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("create database schema: %w", err)
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", schemaVersion)); err != nil {
		return fmt.Errorf("set database schema version: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit database schema: %w", err)
	}
	return nil
}

// Fixed-width UTC timestamps keep SQLite's text ordering chronological.
const timestampLayout = "2006-01-02T15:04:05.000000000Z"

func formatTime(value time.Time) string { return value.UTC().Format(timestampLayout) }

func now() string { return formatTime(time.Now()) }

func parseTime(value string) time.Time {
	parsed, _ := time.Parse(timestampLayout, value)
	return parsed
}
