package clipboard

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type countingSource struct {
	count atomic.Int64
	polls chan struct{}
}

func (s *countingSource) ChangeCount() int {
	s.polls <- struct{}{}
	return int(s.count.Load())
}

func (*countingSource) Read() (Payload, SourceApp, error) {
	return Payload{Text: "latest"}, SourceApp{}, nil
}

func TestWatcherStopsPollingWhilePausedAndResumes(t *testing.T) {
	const interval = 10 * time.Millisecond
	source := &countingSource{polls: make(chan struct{}, 100)}
	captures := make(chan Payload, 10)
	watcher := &Watcher{Source: source, Interval: interval, Capture: func(_ context.Context, payload Payload, _ SourceApp) {
		captures <- payload
	}}
	watcher.SetPaused(true)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		watcher.Run(ctx)
	}()
	t.Cleanup(func() { cancel(); <-done })
	select {
	case <-source.polls:
	case <-time.After(time.Second):
		t.Fatal("watcher did not initialize")
	}
	expectIdle := func() {
		t.Helper()
		select {
		case <-source.polls:
			t.Fatal("paused watcher polled the pasteboard")
		case <-time.After(5 * interval):
		}
	}
	expectIdle()
	source.count.Store(1)
	watcher.SetPaused(false)
	select {
	case <-captures:
	case <-time.After(time.Second):
		t.Fatal("resumed watcher did not capture the changed pasteboard")
	}
	watcher.SetPaused(true)
	// Discard counter reads that started before SetPaused took effect.
	for len(source.polls) > 0 {
		<-source.polls
	}
	expectIdle()
	if _, err := watcher.Poll(ctx, true); err != nil {
		t.Fatal(err)
	}
	if len(captures) != 1 {
		t.Fatal("manual capture did not work while paused")
	}
}

func TestCanceledPollDoesNotReadPasteboard(t *testing.T) {
	source := &countingSource{polls: make(chan struct{}, 1)}
	watcher := &Watcher{Source: source}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if read, err := watcher.Poll(ctx, true); read || err != context.Canceled {
		t.Fatalf("canceled poll = %t, %v", read, err)
	}
	if len(source.polls) != 0 {
		t.Fatal("canceled poll read the pasteboard")
	}
}
