package clipboard

import (
	"context"
	"errors"
	"testing"
)

type pollSource struct {
	count   int
	payload Payload
	err     error
}

func (f *pollSource) ChangeCount() int { return f.count }

func (f *pollSource) Read() (Payload, SourceApp, error) {
	return f.payload, SourceApp{ID: "com.example.editor"}, f.err
}

func TestWatcherPollReadsOnlyChanges(t *testing.T) {
	source := &pollSource{payload: Payload{Text: "first"}}
	var captured []string
	watcher := &Watcher{
		Source: source,
		Capture: func(_ context.Context, payload Payload, _ SourceApp) {
			captured = append(captured, payload.Text)
		},
	}
	ctx := context.Background()
	if read, err := watcher.Poll(ctx, false); err != nil || read {
		t.Fatalf("initial poll = %t, %v", read, err)
	}
	source.count = 1
	if read, err := watcher.Poll(ctx, false); err != nil || !read {
		t.Fatalf("changed poll = %t, %v", read, err)
	}
	if read, err := watcher.Poll(ctx, false); err != nil || read {
		t.Fatalf("repeat poll = %t, %v", read, err)
	}
	if read, err := watcher.Poll(ctx, true); err != nil || !read {
		t.Fatalf("forced poll = %t, %v", read, err)
	}
	if len(captured) != 2 || captured[0] != "first" || captured[1] != "first" {
		t.Errorf("captured = %q", captured)
	}
}

func TestWatcherPollBypassesPause(t *testing.T) {
	paused := true
	source := &pollSource{count: 3, payload: Payload{Text: "paused"}}
	var captured []Payload
	watcher := &Watcher{
		Source: source,
		Paused: func() bool { return paused },
		Capture: func(_ context.Context, payload Payload, _ SourceApp) {
			captured = append(captured, payload)
		},
	}
	ctx := context.Background()
	if read, err := watcher.Poll(ctx, false); err != nil || read {
		t.Fatalf("paused poll = %t, %v", read, err)
	}
	if read, err := watcher.Poll(ctx, true); err != nil || !read {
		t.Fatalf("manual poll = %t, %v", read, err)
	}
	paused = false
	if read, err := watcher.Poll(ctx, false); err != nil || read {
		t.Fatalf("resumed poll = %t, %v", read, err)
	}
	if len(captured) != 1 || captured[0].Text != "paused" {
		t.Errorf("captured = %+v", captured)
	}
}

func TestWatcherPollKeepsCountOnReadError(t *testing.T) {
	source := &pollSource{count: 2, err: errors.New("pasteboard busy")}
	watcher := &Watcher{Source: source, Capture: func(context.Context, Payload, SourceApp) {}}
	ctx := context.Background()
	if _, err := watcher.Poll(ctx, false); err == nil {
		t.Fatal("expected read error")
	}
	source.err = nil
	source.payload = Payload{Text: "retry"}
	if read, err := watcher.Poll(ctx, false); err != nil || !read {
		t.Fatalf("retry poll = %t, %v", read, err)
	}
}
