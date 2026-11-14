package database

import (
	"context"
	"testing"
	"time"
)

func TestMergeBackupAddsMissingRowsAndPreservesExistingMetadata(t *testing.T) {
	current := openTestDB(t)
	backup := openTestDB(t)
	ctx := context.Background()
	timestamp := time.Now().UTC().Format(time.RFC3339Nano)

	if _, _, err := current.Upsert(ctx, testItem(KindText, "hello", "hello")); err != nil {
		t.Fatal(err)
	}
	if _, err := current.SQL().ExecContext(ctx, `INSERT INTO clipboard_groups
		(id, name, icon, is_hidden, sort_order, created_at, updated_at)
		VALUES ('group-shared', 'Current', 'folder', 0, 1, ?, ?)`, timestamp, timestamp); err != nil {
		t.Fatal(err)
	}
	if err := current.EnsureApp(ctx, "com.example.shared", "Current App", nil); err != nil {
		t.Fatal(err)
	}

	if _, _, err := backup.Upsert(ctx, testItem(KindText, "hello", "hello")); err != nil {
		t.Fatal(err)
	}
	if _, err := backup.SQL().ExecContext(ctx, `INSERT INTO clipboard_groups
		(id, name, icon, is_hidden, sort_order, created_at, updated_at)
		VALUES ('group-shared', 'Backup', 'folder', 0, 1, ?, ?)`, timestamp, timestamp); err != nil {
		t.Fatal(err)
	}
	group, err := backup.CreateGroup(ctx, "Imported", "tag")
	if err != nil {
		t.Fatal(err)
	}
	if err := backup.EnsureApp(ctx, "com.example.shared", "Backup App", nil); err != nil {
		t.Fatal(err)
	}
	if err := backup.EnsureApp(ctx, "com.example.imported", "Imported App", nil); err != nil {
		t.Fatal(err)
	}
	if err := backup.UpsertFileIcon(ctx, ".pdf", "pdf-icon.png"); err != nil {
		t.Fatal(err)
	}
	note := "imported note"
	importedAppID := "com.example.imported"
	itemID, imported, err := backup.Upsert(ctx, &Item{
		ID: "imported-text", Kind: KindText, GroupID: &group.ID,
		SourceAppID: &importedAppID,
		Content:     "imported text", Note: &note, IsFavorite: true, IsSensitive: true,
		Platform: "macos",
	})
	if err != nil {
		t.Fatal(err)
	}
	if imported {
		t.Fatal("backup item unexpectedly deduplicated")
	}
	if _, _, err := backup.Upsert(ctx, testItem(KindImage, "image.png", "")); err != nil {
		t.Fatal(err)
	}

	result, err := current.MergeBackup(ctx, backup)
	if err != nil {
		t.Fatal(err)
	}
	if result.Items != 2 || result.Skipped != 1 {
		t.Errorf("merge result = %+v", result)
	}
	counts, err := current.Counts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if counts.Items != 3 {
		t.Errorf("counts = %+v", counts)
	}
	item, err := current.FindByID(ctx, itemID)
	if err != nil {
		t.Fatal(err)
	}
	if item == nil || item.Note == nil || *item.Note != note ||
		!item.IsFavorite || !item.IsSensitive || item.GroupID == nil ||
		*item.GroupID != group.ID || item.SourceAppID == nil ||
		*item.SourceAppID != "com.example.imported" {
		t.Errorf("imported item = %+v", item)
	}
	sharedGroup, err := current.FindGroup(ctx, "group-shared")
	if err != nil {
		t.Fatal(err)
	}
	if sharedGroup == nil || sharedGroup.Name != "Current" {
		t.Errorf("shared group = %+v", sharedGroup)
	}
	sharedApp, err := current.FindApp(ctx, "com.example.shared")
	if err != nil {
		t.Fatal(err)
	}
	if sharedApp.Name != "Current App" {
		t.Errorf("shared app = %+v", sharedApp)
	}
	if _, ok, err := current.FileIcon(ctx, ".pdf"); err != nil || !ok {
		t.Errorf("merged file icon = %t, %v", ok, err)
	}
}
