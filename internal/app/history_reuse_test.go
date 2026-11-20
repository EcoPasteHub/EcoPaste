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

func TestReuseUpdatesUsageAndRespectsSort(t *testing.T) {
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
			if after.UseCount != before.UseCount+3 || !after.UpdatedAt.After(before.UpdatedAt) || !after.CreatedAt.Equal(created) {
				t.Fatalf("usage before=%+v after=%+v", before, after)
			}
			page, err := model.List(ctx, ui.HistoryQuery{Limit: 10})
			if err != nil {
				t.Fatal(err)
			}
			wantFirst := id
			if sort == settings.SortCreatedAt {
				wantFirst = newerID
			}
			if len(page.Items) != 2 || page.Items[0].ID != wantFirst {
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
			if failed.UseCount != after.UseCount || !failed.UpdatedAt.Equal(after.UpdatedAt) {
				t.Fatal("failed write updated usage")
			}
		})
	}
}
