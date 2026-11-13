package database

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "db", "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func testItem(kind Kind, content, search string) *Item {
	item := &Item{Kind: kind, Content: content, Platform: "macos"}
	if search != "" {
		item.SearchText = &search
		item.Summary = &search
	}
	return item
}

func TestUpsertDeduplicatesAndRefreshesUpdatedAt(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	item := testItem(KindText, "hello", "hello")
	item.CreatedAt = time.Now().Add(-time.Hour).UTC()
	item.UpdatedAt = item.CreatedAt

	id, dedup, err := db.Upsert(ctx, item)
	if err != nil {
		t.Fatal(err)
	}
	if dedup {
		t.Fatal("first insert reported a dedupe")
	}
	before, err := db.FindByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}

	again := testItem(KindText, "hello", "hello")
	secondID, dedup, err := db.Upsert(ctx, again)
	if err != nil {
		t.Fatal(err)
	}
	if !dedup || secondID != id {
		t.Fatalf("dedupe = %v, id = %q, want %q", dedup, secondID, id)
	}
	after, err := db.FindByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if after.UseCount != before.UseCount+1 {
		t.Errorf("use count = %d, want %d", after.UseCount, before.UseCount+1)
	}
	if !after.UpdatedAt.After(before.UpdatedAt) {
		t.Errorf("updated_at did not move: %v -> %v", before.UpdatedAt, after.UpdatedAt)
	}
}

func TestHashMatchesPublishedFormat(t *testing.T) {
	got := Hash(KindText, "hello")
	if len(got) != 64 {
		t.Fatalf("hash length = %d", len(got))
	}
	if got == Hash(KindFiles, "hello") {
		t.Fatal("kind is missing from the hash")
	}
	if got != Hash(KindText, "hello") {
		t.Fatal("hash is not stable")
	}
}

func TestPageStripsTextContent(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	if _, _, err := db.Upsert(ctx, testItem(KindText, "long body", "long body")); err != nil {
		t.Fatal(err)
	}
	page, err := db.Page(ctx, Query{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("items = %d", len(page.Items))
	}
	if page.Items[0].Content != "" {
		t.Errorf("list content = %q, want empty", page.Items[0].Content)
	}
	if page.Items[0].Summary == nil || *page.Items[0].Summary != "long body" {
		t.Errorf("summary = %v", page.Items[0].Summary)
	}
}

func TestSearchUsesFTSAndLikeFallback(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	for _, item := range []*Item{
		testItem(KindText, "hello world", "hello world"),
		testItem(KindText, "中文搜索测试", "中文搜索测试"),
		testItem(KindText, "he short", "he short"),
	} {
		if _, _, err := db.Upsert(ctx, item); err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct {
		keyword string
		want    string
	}{
		{"hel", "hello world"},
		{"he", "he short"},
		{"中文搜", "中文搜索测试"},
		{"中", "中文搜索测试"},
	}
	for _, tc := range cases {
		page, err := db.Page(ctx, Query{Keyword: tc.keyword})
		if err != nil {
			t.Fatalf("%q: %v", tc.keyword, err)
		}
		if len(page.Items) == 0 {
			t.Fatalf("%q: no result", tc.keyword)
		}
		found := false
		for _, item := range page.Items {
			if item.Summary != nil && *item.Summary == tc.want {
				found = true
			}
		}
		if !found {
			t.Errorf("%q: %q not in %+v", tc.keyword, tc.want, page.Items)
		}
	}
}

func TestMetadataUpdatesKeepUpdatedAt(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	id, _, err := db.Upsert(ctx, testItem(KindText, "pin me", "pin me"))
	if err != nil {
		t.Fatal(err)
	}
	before, err := db.FindByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ToggleFavorite(ctx, id); err != nil {
		t.Fatal(err)
	}
	if _, err := db.TogglePinned(ctx, id); err != nil {
		t.Fatal(err)
	}
	note := "a note"
	if err := db.SetNote(ctx, id, &note); err != nil {
		t.Fatal(err)
	}
	after, err := db.FindByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if !after.IsFavorite || !after.IsPinned {
		t.Errorf("flags = favorite %v pinned %v", after.IsFavorite, after.IsPinned)
	}
	if !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Errorf("updated_at moved: %v -> %v", before.UpdatedAt, after.UpdatedAt)
	}
}

func TestPagePaginationAndPinnedFirst(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	for i := 0; i < 4; i++ {
		if _, _, err := db.Upsert(ctx, testItem(KindText, string(rune('a'+i)), "text")); err != nil {
			t.Fatal(err)
		}
	}
	page, err := db.Page(ctx, Query{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 4 || len(page.Items) != 2 || !page.HasMore {
		t.Fatalf("page = %+v", page)
	}
	if _, err := db.TogglePinned(ctx, page.Items[0].ID); err != nil {
		t.Fatal(err)
	}
	next, err := db.Page(ctx, Query{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if !next.Items[0].IsPinned {
		t.Error("pinned item is not first")
	}
}

func TestKeywordWithFTSSyntaxCharacters(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	for _, text := range []string{`say "hi" now`, "100% done", "under_score"} {
		if _, _, err := db.Upsert(ctx, testItem(KindText, text, text)); err != nil {
			t.Fatal(err)
		}
	}
	for _, keyword := range []string{`"hi"`, "100%", "_"} {
		if _, err := db.Page(ctx, Query{Keyword: keyword}); err != nil {
			t.Errorf("%q: %v", keyword, err)
		}
	}
}

func TestReadsLegacyTimestampShapes(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	id := NewID()
	_, err := db.SQL().ExecContext(ctx,
		`INSERT INTO clipboard_items (id, kind, content, content_hash, summary, platform, created_at, updated_at)
		 VALUES (?, 'text', 'legacy', 'hash-legacy', 'legacy', 'macos', '2026-11-13 05:00:00+00:00', '2026-11-13 05:00:00+00:00')`,
		id)
	if err != nil {
		t.Fatal(err)
	}
	item, err := db.FindByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if item.CreatedAt.IsZero() {
		t.Fatal("legacy timestamp did not parse")
	}
	if item.CreatedAt.UTC().Format("2006-01-02 15:04") != "2026-11-13 05:00" {
		t.Errorf("created_at = %v", item.CreatedAt)
	}
}
