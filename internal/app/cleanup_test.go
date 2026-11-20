package app

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/clipboard"
	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
)

func TestHistoryCleanupEnabled(t *testing.T) {
	for _, tc := range []struct {
		name    string
		history settings.History
		want    bool
	}{
		{"defaults", settings.Default().Clipboard.History, false},
		{"zero age", settings.History{Retention: settings.Retention{Unit: settings.RetentionHours}}, false},
		{"forever", settings.History{Retention: settings.Retention{Value: 1, Unit: settings.RetentionForever}}, false},
		{"age only", settings.History{Retention: settings.Retention{Value: 1, Unit: settings.RetentionHours}}, true},
		{"count only", settings.History{MaxCount: 1}, true},
		{"both", settings.History{Retention: settings.Retention{Value: 1, Unit: settings.RetentionDays}, MaxCount: 1}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := historyCleanupEnabled(tc.history); got != tc.want {
				t.Fatalf("enabled = %t, want %t", got, tc.want)
			}
		})
	}
}

func startTestCleanup(t *testing.T, runtime *clipboardRuntime, store *settings.Store, interval time.Duration, now func() time.Time) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := startCleanupLoop(ctx, runtime, store, interval, now)
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("cleanup loop did not stop")
		}
	})
}

func TestCleanupLoopRunsOnBuiltInInterval(t *testing.T) {
	ctx := context.Background()
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
	if _, err := store.Update(map[string]any{"clipboard": map[string]any{"history": map[string]any{
		"retention": map[string]any{"value": 1, "unit": "hours"},
	}}}); err != nil {
		t.Fatal(err)
	}

	base := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	item := &database.Item{
		Kind: database.KindText, Content: "expires later", Platform: "macos",
		CreatedAt: base.Add(-30 * time.Minute),
	}
	item.UpdatedAt = item.CreatedAt
	id, _, err := db.Upsert(ctx, item)
	if err != nil {
		t.Fatal(err)
	}
	changed := make(chan struct{}, 1)
	passes := make(chan struct{}, 8)
	runtime := &clipboardRuntime{
		service: &clipboard.Service{
			DB: db,
			Settings: func() settings.Clipboard {
				select {
				case passes <- struct{}{}:
				default:
				}
				return store.Snapshot().Clipboard
			},
		},
		changed: changed,
	}

	var mu sync.Mutex
	now := base
	clock := func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return now
	}
	startTestCleanup(t, runtime, store, 10*time.Millisecond, clock)
	select {
	case <-passes:
	case <-time.After(time.Second):
		t.Fatal("startup cleanup did not run")
	}
	mu.Lock()
	now = base.Add(2 * time.Hour)
	mu.Unlock()

	select {
	case <-changed:
	case <-time.After(2 * time.Second):
		t.Fatal("periodic cleanup did not run")
	}
	found, err := db.FindByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if found != nil {
		t.Error("expired item was not removed by periodic cleanup")
	}
}

func TestRetentionSettingCleansHistoryImmediately(t *testing.T) {
	ctx := context.Background()
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
	runtime := &clipboardRuntime{
		db: db,
		service: &clipboard.Service{
			DB:       db,
			Settings: func() settings.Clipboard { return store.Snapshot().Clipboard },
		},
		changed: make(chan struct{}, 1),
	}
	model := newSettingsModel(store)
	startTestCleanup(t, runtime, store, time.Hour, time.Now)
	old := &database.Item{Kind: database.KindText, Content: "old", CreatedAt: time.Now().Add(-2 * time.Hour)}
	oldID, _, err := db.Upsert(ctx, old)
	if err != nil {
		t.Fatal(err)
	}
	fresh := &database.Item{Kind: database.KindText, Content: "fresh"}
	freshID, _, err := db.Upsert(ctx, fresh)
	if err != nil {
		t.Fatal(err)
	}

	if err := model.Apply("history.retention", "1h"); err != nil {
		t.Fatal(err)
	}
	waitCleanupChange(t, runtime.changed)
	removed, err := db.FindByID(ctx, oldID)
	if err != nil {
		t.Fatal(err)
	}
	if removed != nil {
		t.Error("expired item survived the retention change")
	}
	kept, err := db.FindByID(ctx, freshID)
	if err != nil {
		t.Fatal(err)
	}
	if kept == nil {
		t.Error("fresh item was removed")
	}
}

func TestMaxCountSettingCleansHistoryImmediately(t *testing.T) {
	ctx := context.Background()
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
	runtime := &clipboardRuntime{
		db: db,
		service: &clipboard.Service{
			DB:       db,
			Settings: func() settings.Clipboard { return store.Snapshot().Clipboard },
		},
		changed: make(chan struct{}, 1),
	}
	model := newSettingsModel(store)
	startTestCleanup(t, runtime, store, time.Hour, time.Now)
	for index, content := range []string{"one", "two", "three"} {
		item := &database.Item{
			Kind: database.KindText, Content: content,
			CreatedAt: time.Now().Add(time.Duration(index) * time.Minute),
		}
		if _, _, err := db.Upsert(ctx, item); err != nil {
			t.Fatal(err)
		}
	}
	if err := model.Apply("history.maxCount", uint32(1)); err != nil {
		t.Fatal(err)
	}
	waitCleanupChange(t, runtime.changed)
	page, err := db.Page(ctx, database.Query{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 {
		t.Errorf("items = %d, want 1", len(page.Items))
	}
}

func waitCleanupChange(t *testing.T, changed <-chan struct{}) {
	t.Helper()
	select {
	case <-changed:
	case <-time.After(2 * time.Second):
		t.Fatal("cleanup did not refresh history")
	}
}
