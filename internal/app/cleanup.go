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
	run := func() {
		result, err := runtime.service.Cleanup(ctx, time.Now())
		if err != nil {
			log.Printf("cleanup history: %v", err)
			return
		}
		if result.Removed > 0 {
			runtime.Signal()
		}
	}
	run()
	go func() {
		ticker := time.NewTicker(cleanupCheckInterval)
		defer ticker.Stop()
		last := time.Now()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				hours := store.Snapshot().Clipboard.History.CleanupIntervalHours
				if hours == 0 || time.Since(last) < time.Duration(hours)*time.Hour {
					continue
				}
				run()
				last = time.Now()
			}
		}
	}()
}
