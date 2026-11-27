package database

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

func TestListAppsOrdersByNameAndRefreshesMetadata(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	icon := "first.png"

	if err := db.EnsureApp(ctx, "com.example.zebra", "Zebra", nil); err != nil {
		t.Fatal(err)
	}
	if err := db.EnsureApp(ctx, "com.example.apple", "apple", &icon); err != nil {
		t.Fatal(err)
	}
	apps, err := db.ListApps(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 2 || apps[0].ID != "com.example.apple" || apps[1].ID != "com.example.zebra" {
		t.Fatalf("apps = %+v", apps)
	}

	created := apps[0].CreatedAt
	updatedIcon := "second.png"
	if err := db.EnsureApp(ctx, "com.example.apple", "Apple", &updatedIcon); err != nil {
		t.Fatal(err)
	}
	app, err := db.FindApp(ctx, "com.example.apple")
	if err != nil {
		t.Fatal(err)
	}
	if app.Name != "Apple" || app.IconFile == nil || *app.IconFile != updatedIcon {
		t.Errorf("app = %+v", app)
	}
	if !app.CreatedAt.Equal(created) {
		t.Errorf("created at = %v, want %v", app.CreatedAt, created)
	}
}

func TestFindAppMissing(t *testing.T) {
	db := openTestDB(t)
	if _, err := db.FindApp(context.Background(), "com.example.missing"); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("error = %v", err)
	}
}

func TestDeleteUnreferencedAppsKeepsReferencedRows(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	if err := db.EnsureApp(ctx, "com.example.kept", "Kept", nil); err != nil {
		t.Fatal(err)
	}
	if err := db.EnsureApp(ctx, "com.example.removed", "Removed", nil); err != nil {
		t.Fatal(err)
	}
	source := "com.example.kept"
	item := testItem(KindText, "hello", "hello")
	item.SourceAppID = &source
	if _, _, err := db.Upsert(ctx, item); err != nil {
		t.Fatal(err)
	}

	deleted, err := db.DeleteUnreferencedApps(ctx, []string{"com.example.kept", "com.example.removed"})
	if err != nil {
		t.Fatal(err)
	}
	if len(deleted) != 1 || deleted[0] != "com.example.removed" {
		t.Errorf("deleted = %v", deleted)
	}
	if _, err := db.FindApp(ctx, "com.example.kept"); err != nil {
		t.Errorf("referenced app was removed: %v", err)
	}
}
