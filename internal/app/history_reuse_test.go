package app

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/clipboard"
	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
	"github.com/EcoPasteHub/EcoPaste/internal/ui"
)

func TestInternalReuseDoesNotUpdateUsageOrOrder(t *testing.T) {
	root := t.TempDir()
	db, err := database.Open(filepath.Join(root, "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	store, err := settings.Open(filepath.Join(root, settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	previousTrusted := inputTrusted
	inputTrusted = func() bool { return true }
	t.Cleanup(func() { inputTrusted = previousTrusted })
	writer := &recordingWriter{}
	model := newHistoryModel(&clipboardRuntime{db: db, service: &clipboard.Service{DB: db, Writer: writer}}, store)
	ctx := context.Background()
	created := time.Now().UTC().Add(-2 * time.Hour)
	id, _, err := db.Upsert(ctx, &database.Item{Kind: database.KindText, Content: "older", CreatedAt: created})
	if err != nil {
		t.Fatal(err)
	}
	newerID, _, err := db.Upsert(ctx, &database.Item{Kind: database.KindText, Content: "newer", CreatedAt: created.Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	for _, sort := range []settings.Sort{settings.SortUpdatedAt, settings.SortCreatedAt, settings.SortUseCount} {
		t.Run(string(sort), func(t *testing.T) {
			if _, err := store.Update(map[string]any{"clipboard": map[string]any{"content": map[string]any{"sort": sort}}}); err != nil {
				t.Fatal(err)
			}
			before, err := db.FindByID(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := model.Copy(ctx, id); err != nil {
				t.Fatal(err)
			}
			if _, err := model.CopyPlain(ctx, id); err != nil {
				t.Fatal(err)
			}
			if _, _, err := model.Activate(ctx, id); err != nil {
				t.Fatal(err)
			}
			after, err := db.FindByID(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			if after.UseCount != before.UseCount || !after.UpdatedAt.Equal(before.UpdatedAt) || !after.CreatedAt.Equal(before.CreatedAt) {
				t.Fatalf("usage before=%+v after=%+v", before, after)
			}
			page, err := model.List(ctx, ui.HistoryQuery{Limit: 10})
			if err != nil {
				t.Fatal(err)
			}
			if len(page.Items) != 2 || page.Items[0].ID != newerID {
				t.Fatalf("sort %s = %+v", sort, page.Items)
			}
			writer.err = errors.New("write failed")
			if _, err := model.Copy(ctx, id); err == nil {
				t.Fatal("failed write succeeded")
			}
			writer.err = nil
			failed, err := db.FindByID(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			if failed.UseCount != after.UseCount || !failed.UpdatedAt.Equal(after.UpdatedAt) || !failed.CreatedAt.Equal(after.CreatedAt) {
				t.Fatal("failed write updated usage")
			}
		})
	}
}

func TestShouldRefreshHistoryForDuplicateCapture(t *testing.T) {
	for _, tc := range []struct {
		sort         settings.Sort
		deduplicated bool
		wantRefresh  bool
	}{
		{settings.SortCreatedAt, false, true},
		{settings.SortCreatedAt, true, false},
		{settings.SortUpdatedAt, true, true},
		{settings.SortUseCount, true, true},
	} {
		if got := shouldRefreshHistory(tc.sort, tc.deduplicated); got != tc.wantRefresh {
			t.Errorf("shouldRefreshHistory(%q, %t) = %t, want %t", tc.sort, tc.deduplicated, got, tc.wantRefresh)
		}
	}
}

func TestExternalDuplicateMovesOnlyForUsageBasedSorts(t *testing.T) {
	for _, sort := range []settings.Sort{settings.SortUpdatedAt, settings.SortUseCount, settings.SortCreatedAt} {
		t.Run(string(sort), func(t *testing.T) {
			root := t.TempDir()
			db, err := database.Open(filepath.Join(root, "clipboard.db"))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { db.Close() })
			store, err := settings.Open(filepath.Join(root, settings.FileName))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := store.Update(map[string]any{"clipboard": map[string]any{"content": map[string]any{"sort": sort}}}); err != nil {
				t.Fatal(err)
			}
			cfg := store.Snapshot().Clipboard
			service := &clipboard.Service{DB: db, Settings: func() settings.Clipboard { return cfg }}
			model := newHistoryModel(&clipboardRuntime{db: db, service: service}, store)
			ctx := context.Background()
			created := time.Now().UTC().Add(-2 * time.Hour)
			olderID, _, err := db.Upsert(ctx, &database.Item{Kind: database.KindText, Content: "older", CreatedAt: created})
			if err != nil {
				t.Fatal(err)
			}
			newerID, _, err := db.Upsert(ctx, &database.Item{Kind: database.KindText, Content: "newer", CreatedAt: created.Add(time.Hour)})
			if err != nil {
				t.Fatal(err)
			}
			if id, changed, err := service.Capture(ctx, clipboard.Payload{Text: "older"}, clipboard.SourceApp{}); err != nil || !changed || id != olderID {
				t.Fatalf("duplicate capture id=%q changed=%t err=%v", id, changed, err)
			}
			page, err := model.List(ctx, ui.HistoryQuery{Limit: 10})
			if err != nil {
				t.Fatal(err)
			}
			wantFirst := olderID
			if sort == settings.SortCreatedAt {
				wantFirst = newerID
			}
			if len(page.Items) != 2 || page.Items[0].ID != wantFirst {
				t.Fatalf("sort %s after external duplicate = %+v", sort, page.Items)
			}
		})
	}
}
