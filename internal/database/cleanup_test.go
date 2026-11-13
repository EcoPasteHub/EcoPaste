package database

import (
	"context"
	"testing"
	"time"
)

func insertAt(t *testing.T, db *DB, content string, createdAt time.Time) string {
	t.Helper()
	item := testItem(KindText, content, content)
	item.CreatedAt = createdAt
	item.UpdatedAt = createdAt
	id, _, err := db.Upsert(context.Background(), item)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestCleanupHistoryByRetentionKeepsProtectedItems(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	old := now.Add(-48 * time.Hour)
	cutoff := now.Add(-24 * time.Hour)

	plain := insertAt(t, db, "old plain", old)
	favorite := insertAt(t, db, "old favorite", old)
	pinned := insertAt(t, db, "old pinned", old)
	recent := insertAt(t, db, "recent", now)
	if _, err := db.ToggleFavorite(ctx, favorite); err != nil {
		t.Fatal(err)
	}
	if _, err := db.TogglePinned(ctx, pinned); err != nil {
		t.Fatal(err)
	}

	result, err := db.CleanupHistory(ctx, &cutoff, 0)
	if err != nil {
		t.Fatal(err)
	}
	if result.Removed != 1 {
		t.Errorf("removed = %d, want 1", result.Removed)
	}
	for id, want := range map[string]bool{plain: false, favorite: true, pinned: true, recent: true} {
		item, err := db.FindByID(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if (item != nil) != want {
			t.Errorf("item %s exists = %v, want %v", id, item != nil, want)
		}
	}
}

func TestCleanupHistoryByMaxCount(t *testing.T) {
	db := openTestDB(t)
	now := time.Now().UTC()
	ids := []string{
		insertAt(t, db, "one", now.Add(-3*time.Hour)),
		insertAt(t, db, "two", now.Add(-2*time.Hour)),
		insertAt(t, db, "three", now.Add(-time.Hour)),
	}
	result, err := db.CleanupHistory(context.Background(), nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	if result.Removed != 2 {
		t.Errorf("removed = %d, want 2", result.Removed)
	}
	item, err := db.FindByID(context.Background(), ids[2])
	if err != nil {
		t.Fatal(err)
	}
	if item == nil {
		t.Error("newest item was removed")
	}
}

func TestClearHistoryOptionallyKeepsFavoritesAndPinned(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	plain := insertAt(t, db, "plain", now)
	favorite := insertAt(t, db, "favorite", now)
	pinned := insertAt(t, db, "pinned", now)
	if _, err := db.ToggleFavorite(ctx, favorite); err != nil {
		t.Fatal(err)
	}
	if _, err := db.TogglePinned(ctx, pinned); err != nil {
		t.Fatal(err)
	}

	result, err := db.ClearHistory(ctx, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Removed != 1 {
		t.Errorf("removed = %d, want 1", result.Removed)
	}
	for id, want := range map[string]bool{plain: false, favorite: true, pinned: true} {
		item, err := db.FindByID(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if (item != nil) != want {
			t.Errorf("item %s exists = %v, want %v", id, item != nil, want)
		}
	}

	if _, err := db.ClearHistory(ctx, true, true); err != nil {
		t.Fatal(err)
	}
	page, err := db.Page(ctx, Query{})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 0 {
		t.Errorf("remaining = %d, want 0", page.Total)
	}
}

func TestCleanupReturnsImageFiles(t *testing.T) {
	db := openTestDB(t)
	now := time.Now().UTC()
	item := &Item{Kind: KindImage, Content: "abc123.png", Platform: "macos", CreatedAt: now.Add(-48 * time.Hour)}
	item.UpdatedAt = item.CreatedAt
	if _, _, err := db.Upsert(context.Background(), item); err != nil {
		t.Fatal(err)
	}
	cutoff := now.Add(-24 * time.Hour)
	result, err := db.CleanupHistory(context.Background(), &cutoff, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.ImageFiles) != 1 || result.ImageFiles[0] != "abc123.png" {
		t.Errorf("image files = %v", result.ImageFiles)
	}
}
