package clipboard

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Watcher polls the pasteboard's change counter and reports new
// snapshots. Starting it never captures what was already on the
// pasteboard.
type Watcher struct {
	Source   Source
	Interval time.Duration
	Capture  func(ctx context.Context, payload Payload, source SourceApp)
	Guard    *Guard

	mu           sync.Mutex
	last         int
	paused       atomic.Bool
	pauseOnce    sync.Once
	pauseChanges chan struct{}
}

func (w *Watcher) Paused() bool { return w.paused.Load() }

// SetPaused wakes the watcher so pausing also stops its polling timer.
// Manual Poll calls remain available while automatic capture is paused.
func (w *Watcher) SetPaused(paused bool) {
	changes := w.pauseNotifications()
	if w.paused.Swap(paused) == paused {
		return
	}
	select {
	case changes <- struct{}{}:
	default:
	}
}

func (w *Watcher) pauseNotifications() chan struct{} {
	w.pauseOnce.Do(func() { w.pauseChanges = make(chan struct{}, 1) })
	return w.pauseChanges
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
	changes := w.pauseNotifications()
	var ticker *time.Ticker
	var ticks <-chan time.Time
	stop := func() {
		if ticker != nil {
			ticker.Stop()
			ticker, ticks = nil, nil
		}
	}
	defer stop()
	configure := func() {
		if w.Paused() {
			stop()
		} else if ticker == nil {
			ticker = time.NewTicker(interval)
			ticks = ticker.C
		}
	}
	configure()
	for {
		select {
		case <-ctx.Done():
			return
		case <-changes:
			configure()
		case <-ticks:
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
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if !force && w.Paused() {
		return false, nil
	}
	var payload Payload
	var source SourceApp
	readSnapshot := false
	read := func(isSelfWrite func(int) bool) error {
		for attempt := 0; attempt < 3; attempt++ {
			count := w.Source.ChangeCount()
			if !force && count == w.last {
				return nil
			}
			if isSelfWrite(count) {
				w.last = count
				return nil
			}
			var err error
			payload, source, err = w.Source.Read()
			if err != nil {
				return err
			}
			after := w.Source.ChangeCount()
			if after != count {
				continue
			}
			w.last = after
			readSnapshot = true
			return nil
		}
		return nil
	}
	var err error
	if w.Guard != nil {
		err = w.Guard.Synchronize(read)
	} else {
		err = read(func(int) bool { return false })
	}
	if err != nil {
		return false, err
	}
	if !readSnapshot {
		return false, nil
	}
	if w.Capture != nil {
		w.Capture(ctx, payload, source)
	}
	return true, nil
}
