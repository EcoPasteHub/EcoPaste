package database

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func BenchmarkUpsertRepeatedRichText(b *testing.B) {
	db, err := Open(filepath.Join(b.TempDir(), "clipboard.db"))
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()
	item := testItem(KindText, strings.Repeat("<b>formatted</b>", 16*1024), "formatted")
	ctx := context.Background()
	if _, _, err := db.Upsert(ctx, item); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, deduplicated, err := db.Upsert(ctx, item); err != nil || !deduplicated {
			b.Fatalf("deduplicated = %v, err = %v", deduplicated, err)
		}
	}
}
