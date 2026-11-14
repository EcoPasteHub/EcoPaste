package database

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
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

// Open opens the clipboard database, creating the published schema when
// the file has none. An existing legacy database is used as is.
func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create database dir: %w", err)
	}
	dsn := "file:" + path +
		"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=synchronous(NORMAL)"
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
	return d, nil
}

// Path returns the database file's location.
func (d *DB) Path() string { return d.path }

// Close flushes and closes the database.
func (d *DB) Close() error { return d.sql.Close() }

// SQL exposes the pool for modules that own their own queries, such as
// groups and source apps.
func (d *DB) SQL() *sql.DB { return d.sql }

func (d *DB) ensureSchema(ctx context.Context) error {
	var name string
	err := d.sql.QueryRowContext(ctx,
		"SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'clipboard_items'").Scan(&name)
	if err == nil {
		return d.ensureSQLxBaseline(ctx)
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("inspect database schema: %w", err)
	}
	if _, err := d.sql.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("create database schema: %w", err)
	}
	return d.ensureSQLxBaseline(ctx)
}

func now() string { return time.Now().UTC().Format(time.RFC3339Nano) }

func parseTime(value string) time.Time {
	for _, layout := range []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02T15:04:05.999999999",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, value); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}
