package storage

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/domain"
)

func newTestDatabase(t *testing.T) *Database {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	return db
}

func sampleItem(id, content string) domain.Item {
	now := time.Date(2026, 11, 12, 13, 43, 15, 414023000, time.UTC)
	return domain.Item{
		ID:         id,
		Kind:       domain.KindText,
		Content:    content,
		SearchText: content,
		Summary:    content,
		UseCount:   1,
		Platform:   "macos",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func TestInitializeCreatesRustCompatibleSchema(t *testing.T) {
	db := newTestDatabase(t)
	probe, err := db.Probe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if probe.SQLxVersion != 1 || probe.SQLxSuccess != 1 {
		t.Fatalf("unexpected sqlx ledger: %+v", probe)
	}
	if probe.JournalMode != "wal" {
		t.Fatalf("journal_mode = %q", probe.JournalMode)
	}
	var checksum string
	if err := db.SQL().QueryRow("SELECT hex(checksum) FROM _sqlx_migrations").Scan(&checksum); err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(checksum, sqlxChecksum) {
		t.Fatalf("checksum = %s", checksum)
	}
}

func TestInitializeRejectsLegacyHistoryOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	if _, err := raw.Exec("CREATE TABLE history (id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}

	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.Initialize(context.Background()); !errors.Is(err, ErrLegacySchema) {
		t.Fatalf("Initialize() = %v, want ErrLegacySchema", err)
	}
}

func TestUpsertDeduplicatesAndRefreshesUpdatedAt(t *testing.T) {
	db := newTestDatabase(t)
	ctx := context.Background()
	item := sampleItem("a", "hello")

	first, err := db.UpsertItem(ctx, &item)
	if err != nil {
		t.Fatal(err)
	}
	if first.Deduplicated {
		t.Fatal("first insert must not deduplicate")
	}

	original, err := db.Find(ctx, "a")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond)

	duplicate := sampleItem("b", "hello")
	second, err := db.UpsertItem(ctx, &duplicate)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Deduplicated || second.ID != "a" {
		t.Fatalf("second upsert = %+v", second)
	}

	after, err := db.Find(ctx, "a")
	if err != nil {
		t.Fatal(err)
	}
	if after.UseCount != 2 {
		t.Fatalf("use_count = %d", after.UseCount)
	}
	if !after.UpdatedAt.After(original.UpdatedAt) {
		t.Fatalf("updated_at did not refresh: %v -> %v", original.UpdatedAt, after.UpdatedAt)
	}
}

func TestItemMetadataUpdatesDoNotRefreshUpdatedAt(t *testing.T) {
	db := newTestDatabase(t)
	ctx := context.Background()
	item := sampleItem("a", "hello")
	if _, err := db.UpsertItem(ctx, &item); err != nil {
		t.Fatal(err)
	}
	before, err := db.Find(ctx, "a")
	if err != nil {
		t.Fatal(err)
	}

	note := "note"
	if err := db.SetNote(ctx, "a", &note); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ToggleFavorite(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.TogglePinned(ctx, "a"); err != nil {
		t.Fatal(err)
	}

	after, err := db.Find(ctx, "a")
	if err != nil {
		t.Fatal(err)
	}
	if !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatalf("metadata changed updated_at: %v -> %v", before.UpdatedAt, after.UpdatedAt)
	}
	if !after.IsFavorite || !after.IsPinned || after.Note != note {
		t.Fatalf("metadata not persisted: %+v", after)
	}
}

func TestGroupDeleteNullsItemGroup(t *testing.T) {
	db := newTestDatabase(t)
	ctx := context.Background()
	now := time.Now().UTC()
	if err := db.InsertGroup(ctx, domain.Group{
		ID: "g", Name: "Work", Icon: "folder", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	item := sampleItem("a", "hello")
	item.GroupID = "g"
	if _, err := db.UpsertItem(ctx, &item); err != nil {
		t.Fatal(err)
	}
	if err := db.DeleteGroup(ctx, "g"); err != nil {
		t.Fatal(err)
	}
	after, err := db.Find(ctx, "a")
	if err != nil {
		t.Fatal(err)
	}
	if after.GroupID != "" {
		t.Fatalf("group_id = %q", after.GroupID)
	}
}

func TestTimestampRoundTripUsesMicrosecondsAndOffset(t *testing.T) {
	db := newTestDatabase(t)
	ctx := context.Background()
	item := sampleItem("a", "hello")
	if _, err := db.UpsertItem(ctx, &item); err != nil {
		t.Fatal(err)
	}
	var raw string
	if err := db.SQL().QueryRow("SELECT created_at FROM clipboard_items WHERE id = 'a'").Scan(&raw); err != nil {
		t.Fatal(err)
	}
	if raw != "2026-11-12T13:43:15.414023+00:00" {
		t.Fatalf("created_at = %q", raw)
	}
}
