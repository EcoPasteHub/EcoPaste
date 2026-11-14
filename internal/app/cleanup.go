package app

import (
	"context"
	"log"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/settings"
)

const cleanupCheckInterval = time.Minute

// startCleanup runs one pass at launch, then honors the configured
// interval. A zero interval keeps the launch pass only.
func startCleanup(ctx context.Context, runtime *clipboardRuntime, store *settings.Store) {
	startCleanupLoop(ctx, runtime, store, cleanupCheckInterval, time.Now)
}

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
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(check)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				hours := store.Snapshot().Clipboard.History.CleanupIntervalHours
				if !cleanupDue(now(), last, hours) {
					continue
				}
				run()
				last = now()
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
