package app

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/clipboard"
	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
)

func TestCleanupLoopFollowsEnabledLimits(t *testing.T) {
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
	passes := make(chan struct{}, 32)
	runtime := &clipboardRuntime{service: &clipboard.Service{
		DB: db,
		Settings: func() settings.Clipboard {
			select {
			case passes <- struct{}{}:
			default:
			}
			return store.Snapshot().Clipboard
		},
	}}
	interval := 20 * time.Millisecond
	startTestCleanup(t, runtime, store, interval, time.Now)
	noPass := func() {
		t.Helper()
		select {
		case <-passes:
			t.Fatal("cleanup ran while both limits were disabled")
		case <-time.After(3 * interval):
		}
	}
	waitPass := func() {
		t.Helper()
		select {
		case <-passes:
		case <-time.After(time.Second):
			t.Fatal("enabled cleanup did not run")
		}
	}
	apply := func(history map[string]any) {
		t.Helper()
		if _, err := store.Update(map[string]any{"clipboard": map[string]any{"history": history}}); err != nil {
			t.Fatal(err)
		}
	}
	noPass()
	if _, err := store.Update(map[string]any{"appearance": map[string]any{"theme": "dark"}}); err != nil {
		t.Fatal(err)
	}
	noPass()
	apply(map[string]any{"maxCount": 10})
	waitPass()
	waitPass()
	apply(map[string]any{"maxCount": 0})
	// Allow a pass already in flight before disabling to finish.
	<-time.After(2 * interval)
	for len(passes) > 0 {
		<-passes
	}
	noPass()
	apply(map[string]any{"retention": map[string]any{"value": 1, "unit": "hours"}})
	waitPass()
	waitPass()
	apply(map[string]any{"retention": map[string]any{"value": 0, "unit": "forever"}})
	<-time.After(2 * interval)
	for len(passes) > 0 {
		<-passes
	}
	noPass()
}

func TestUnrelatedSettingsDoNotTriggerCleanup(t *testing.T) {
	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(map[string]any{"clipboard": map[string]any{"history": map[string]any{"maxCount": 10}}}); err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(filepath.Join(t.TempDir(), "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	passes := make(chan struct{}, 2)
	runtime := &clipboardRuntime{service: &clipboard.Service{DB: db, Settings: func() settings.Clipboard {
		select {
		case passes <- struct{}{}:
		default:
		}
		return store.Snapshot().Clipboard
	}}}
	startTestCleanup(t, runtime, store, time.Hour, time.Now)
	select {
	case <-passes:
	case <-time.After(time.Second):
		t.Fatal("startup cleanup did not run")
	}
	if _, err := store.Update(map[string]any{"appearance": map[string]any{"theme": "dark"}}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-passes:
		t.Fatal("unrelated setting triggered cleanup")
	case <-time.After(60 * time.Millisecond):
	}
}
