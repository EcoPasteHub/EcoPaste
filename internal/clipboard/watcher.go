package clipboard

import (
	"context"
	"sync"
	"time"
)

// Watcher polls the pasteboard's change counter and reports new
// snapshots. Starting it never captures what was already on the
// pasteboard.
type Watcher struct {
	Source   Source
	Interval time.Duration
	Capture  func(ctx context.Context, payload Payload, source SourceApp)
	Paused   func() bool

	mu   sync.Mutex
	last int
}

// Run watches until ctx ends.
func (w *Watcher) Run(ctx context.Context) {
	interval := w.Interval
	if interval <= 0 {
		interval = 120 * time.Millisecond
	}
	w.mu.Lock()
	w.last = w.Source.ChangeCount()
	w.mu.Unlock()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = w.Poll(ctx, false)
		}
	}
}

// Poll reads the pasteboard when its change counter moved past the last
// snapshot. force also reads an unchanged pasteboard, as the manual
// "read clipboard" command does.
func (w *Watcher) Poll(ctx context.Context, force bool) (bool, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !force && w.Paused != nil && w.Paused() {
		return false, nil
	}
	count := w.Source.ChangeCount()
	if !force && count == w.last {
		return false, nil
	}
	payload, source, err := w.Source.Read()
	if err != nil {
		return false, err
	}
	w.last = count
	if w.Capture != nil {
		w.Capture(ctx, payload, source)
	}
	return true, nil
}
