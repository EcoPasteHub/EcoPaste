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

func TestCleanupDue(t *testing.T) {
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		last  time.Time
		hours uint32
		want  bool
	}{
		{"disabled", now.Add(-24 * time.Hour), 0, false},
		{"not yet due", now.Add(-59 * time.Minute), 1, false},
		{"exactly due", now.Add(-time.Hour), 1, true},
		{"overdue", now.Add(-25 * time.Hour), 24, true},
		{"clock moved backwards", now.Add(time.Minute), 1, false},
		{"overflow is not due", now.Add(-24 * time.Hour), ^uint32(0), false},
	}
	for _, tt := range tests {
		if got := cleanupDue(now, tt.last, tt.hours); got != tt.want {
			t.Errorf("%s: cleanupDue() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestCleanupLoopRunsOnConfiguredInterval(t *testing.T) {
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
		"retention":            map[string]any{"value": 1, "unit": "hours"},
		"cleanupIntervalHours": 1,
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
	runtime := &clipboardRuntime{
		service: &clipboard.Service{
			DB:       db,
			Settings: func() settings.Clipboard { return store.Snapshot().Clipboard },
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
	runCtx, cancel := context.WithCancel(ctx)
	done := startCleanupLoop(runCtx, runtime, store, 10*time.Millisecond, clock)
	mu.Lock()
	now = base.Add(2 * time.Hour)
	mu.Unlock()

	select {
	case <-changed:
	case <-time.After(2 * time.Second):
		t.Fatal("periodic cleanup did not run")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cleanup loop did not stop")
	}
	found, err := db.FindByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if found != nil {
		t.Error("expired item was not removed by periodic cleanup")
	}
}
