package app

import (
	"context"
	"log"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/settings"
)

// startCleanup runs one pass at launch, then schedules the next pass from the
// configured interval. A zero interval keeps the launch pass only.
func startCleanup(ctx context.Context, runtime *clipboardRuntime, store *settings.Store) {
	startCleanupLoop(ctx, runtime, store, 0, time.Now)
}

// startCleanupLoop waits until the configured interval has elapsed. check is an
// optional cap on a single wait, for tests that use a controllable clock; zero
// waits exactly until the next pass is due, and settings changes wake it early.
func startCleanupLoop(
	ctx context.Context,
	runtime *clipboardRuntime,
	store *settings.Store,
	check time.Duration,
	now func() time.Time,
) <-chan struct{} {
	run := func() {
		result, err := runtime.service.Cleanup(ctx, now())
		if err != nil {
			log.Printf("cleanup history: %v", err)
			return
		}
		if result.Removed > 0 {
			runtime.Signal()
		}
	}
	run()
	last := now()
	changes, stopChanges := store.Changes()
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer stopChanges()
		for {
			hours := store.Snapshot().Clipboard.History.CleanupIntervalHours
			interval := time.Duration(hours) * time.Hour
			if interval <= 0 {
				// Disabled, or a value so large the duration overflowed: no
				// periodic pass is due. Wait for a settings change rather than
				// waking on a timer that can never fire.
				select {
				case <-ctx.Done():
					return
				case <-changes:
					continue
				}
			}
			due := last.Add(interval)
			if !due.After(now()) {
				run()
				last = now()
				continue
			}
			wait := due.Sub(now())
			if check > 0 && wait > check {
				wait = check
			}
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-changes:
				timer.Stop()
			case <-timer.C:
				if cleanupDue(now(), last, store.Snapshot().Clipboard.History.CleanupIntervalHours) {
					run()
					last = now()
				}
			}
		}
	}()
	return done
}

func cleanupDue(now, last time.Time, hours uint32) bool {
	if hours == 0 {
		return false
	}
	interval := time.Duration(hours) * time.Hour
	return interval > 0 && now.Sub(last) >= interval
}
