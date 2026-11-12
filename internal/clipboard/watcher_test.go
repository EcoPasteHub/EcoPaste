package clipboard

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/settings"
)

type fakeReader struct {
	mu      sync.Mutex
	count   int64
	reads   int
	fail    int
	payload Payload
}

func (f *fakeReader) ChangeCount() int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.count
}

func (f *fakeReader) Read(settings.Capture) (Payload, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	if f.fail > 0 {
		f.fail--
		return Payload{}, false, errors.New("temporary read failure")
	}
	return f.payload, true, nil
}

func TestWatcherEmitsDistinctRevisionsWithSource(t *testing.T) {
	reader := &fakeReader{
		count:   1,
		fail:    2,
		payload: Payload{Kind: "text", Text: Text{Plain: "hello"}},
	}
	events := make(chan Captured, 1)
	watcher := NewWatcher(
		reader,
		func() SourceApp { return SourceApp{ID: "com.example.app", Name: "Example"} },
		func() settings.Capture { return settings.Default().Clipboard.Capture },
		func(_ context.Context, captured Captured) { events <- captured },
	)
	watcher.interval = 2 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go watcher.Run(ctx)

	time.Sleep(5 * time.Millisecond)
	reader.mu.Lock()
	reader.count = 2
	reader.mu.Unlock()

	select {
	case captured := <-events:
		if captured.Payload.Text.Plain != "hello" {
			t.Fatalf("payload = %+v", captured.Payload)
		}
		if captured.App.ID != "com.example.app" {
			t.Fatalf("source app = %+v", captured.App)
		}
	case <-time.After(time.Second):
		t.Fatal("watcher did not emit")
	}

	reader.mu.Lock()
	defer reader.mu.Unlock()
	if reader.reads < 3 {
		t.Fatalf("reads = %d, retries were not used", reader.reads)
	}
}

func TestGuardConsumesOnceAndExpires(t *testing.T) {
	guard := NewWritebackGuard()
	now := time.Now()
	guard.now = func() time.Time { return now }
	guard.Suppress("a")
	if !guard.Consume("a") {
		t.Fatal("first consume must match")
	}
	if guard.Consume("a") {
		t.Fatal("guard must consume once")
	}

	guard.Suppress("b")
	now = now.Add(3 * time.Second)
	if guard.Consume("b") {
		t.Fatal("expired guard must not match")
	}
}
