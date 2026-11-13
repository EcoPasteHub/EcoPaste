package clipboard

import (
	"context"
	"time"
)

// Watcher polls the pasteboard's change counter and reports new
// snapshots. Starting it never captures what was already on the
// pasteboard.
type Watcher struct {
	Source   Source
	Interval time.Duration
	Capture  func(ctx context.Context, payload Payload, source SourceApp)

	last int
}

// Run watches until ctx ends.
func (w *Watcher) Run(ctx context.Context) {
	interval := w.Interval
	if interval <= 0 {
		interval = 120 * time.Millisecond
	}
	w.last = w.Source.ChangeCount()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			count := w.Source.ChangeCount()
			if count == w.last {
				continue
			}
			w.last = count
			payload, source, err := w.Source.Read()
			if err != nil || w.Capture == nil {
				continue
			}
			w.Capture(ctx, payload, source)
		}
	}
}
