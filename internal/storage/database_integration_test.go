package storage

import (
	"context"
	"os"
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/domain"
)

func TestRealDatabaseReadOnlyCompatibility(t *testing.T) {
	path := os.Getenv("ECOPASTE_TEST_DB")
	if path == "" {
		t.Skip("set ECOPASTE_TEST_DB to a real EcoPaste clipboard.db")
	}

	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	probe, err := db.Probe(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if probe.SQLxVersion == 0 || probe.SQLxSuccess != 1 {
		t.Fatalf("unexpected sqlx migration state: %+v", probe)
	}
	if probe.JournalMode != "wal" {
		t.Fatalf("journal_mode = %q", probe.JournalMode)
	}

	page, err := db.QueryPage(ctx, domain.Query{Sort: domain.SortUpdatedAt, Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) == 0 {
		t.Fatal("expected items")
	}
	if page.Items[0].CreatedAt.IsZero() || page.Items[0].UpdatedAt.IsZero() {
		t.Fatal("timestamps did not parse")
	}

	for _, item := range page.Items {
		if item.Kind != domain.KindText {
			continue
		}
		fts, err := db.QueryPage(ctx, domain.Query{
			Keyword: item.Summary,
			Sort:    domain.SortUpdatedAt,
			Limit:   5,
		})
		if err != nil {
			t.Fatalf("FTS query failed: %v", err)
		}
		t.Logf("FTS query %q returned %d/%d", item.Summary, len(fts.Items), fts.Total)
		break
	}
}
