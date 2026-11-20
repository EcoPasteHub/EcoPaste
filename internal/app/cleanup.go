package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/clipboard"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
)

const historyCleanupInterval = 15 * time.Minute

// startCleanup observes policy changes; disabled limits allocate no timer and
// perform no cleanup. Enabled limits run at startup, on change and periodically.
func startCleanup(ctx context.Context, runtime *clipboardRuntime, store *settings.Store) <-chan struct{} {
	return startCleanupLoop(ctx, runtime, store, historyCleanupInterval, time.Now)
}

func startCleanupLoop(
	ctx context.Context,
	runtime *clipboardRuntime,
	store *settings.Store,
	interval time.Duration,
	now func() time.Time,
) <-chan struct{} {
	changes, stopChanges := store.Changes()
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer stopChanges()
		var timer *time.Timer
		var ticks <-chan time.Time
		defer func() {
			if timer != nil {
				timer.Stop()
			}
		}()
		run := func() {
			if ctx.Err() != nil || !historyCleanupEnabled(store.Snapshot().Clipboard.History) {
				return
			}
			result, err := runtime.service.Cleanup(ctx, now())
			if err != nil {
				if ctx.Err() == nil {
					slog.Error("clean clipboard history", "module", "app.cleanup", "error", err)
				}
				return
			}
			if result.Removed > 0 {
				runtime.Signal()
			}
		}
		configure := func(policy settings.History) {
			if timer != nil {
				timer.Stop()
				timer, ticks = nil, nil
			}
			if !historyCleanupEnabled(policy) || ctx.Err() != nil {
				return
			}
			run()
			timer = time.NewTimer(interval)
			ticks = timer.C
		}
		policy := store.Snapshot().Clipboard.History
		configure(policy)
		for {
			select {
			case <-ctx.Done():
				return
			case <-changes:
				next := store.Snapshot().Clipboard.History
				if next != policy {
					policy = next
					configure(policy)
				}
			case <-ticks:
				next := store.Snapshot().Clipboard.History
				if next != policy {
					policy = next
					configure(policy)
					continue
				}
				run()
				timer.Reset(interval)
			}
		}
	}()
	return done
}

func historyCleanupEnabled(policy settings.History) bool {
	return policy.MaxCount > 0 || clipboard.RetentionCutoff(policy.Retention, time.Time{}) != nil
}
