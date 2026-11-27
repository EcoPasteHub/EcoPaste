package database

import (
	"context"
	"path/filepath"
	"testing"
)

func TestSnapshotAndCounts(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	for _, item := range []*Item{
		testItem(KindText, "text", "text"),
		testItem(KindImage, "image", ""),
		testItem(KindFiles, "files", ""),
	} {
		if _, _, err := db.Upsert(ctx, item); err != nil {
			t.Fatal(err)
		}
	}
	counts, err := db.Counts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if counts != (ItemCounts{Items: 3, Text: 1, Images: 1, Files: 1}) {
		t.Errorf("counts = %+v", counts)
	}

	snapshot := filepath.Join(t.TempDir(), "snapshot.db")
	if err := db.Snapshot(ctx, snapshot); err != nil {
		t.Fatal(err)
	}
	copyDB, err := Open(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	defer copyDB.Close()
	copied, err := copyDB.Counts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if copied != counts {
		t.Errorf("snapshot counts = %+v, want %+v", copied, counts)
	}
}
