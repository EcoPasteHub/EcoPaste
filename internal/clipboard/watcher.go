package clipboard

import (
	"context"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/settings"
)

const (
	defaultPollInterval = 120 * time.Millisecond
)

var readRetryDelays = []time.Duration{
	15 * time.Millisecond,
	35 * time.Millisecond,
	75 * time.Millisecond,
}

// Reader is the platform pasteboard port used by the watcher.
type Reader interface {
	ChangeCount() int64
	Read(settings.Capture) (Payload, bool, error)
}

// SourceApp is captured before persistence so later app switches do not
// change the recorded source.
type SourceApp struct {
	ID       string
	Name     string
	Platform string
}

// Captured is one pasteboard revision plus its source app.
type Captured struct {
	Payload     Payload
	App         SourceApp
	ChangeCount int64
}

// Watcher polls the pasteboard and forwards distinct revisions.
type Watcher struct {
	reader    Reader
	frontmost func() SourceApp
	capture   func() settings.Capture
	onCapture func(context.Context, Captured)
	interval  time.Duration
}

// NewWatcher wires the pasteboard port, current settings and handler.
func NewWatcher(
	reader Reader,
	frontmost func() SourceApp,
	capture func() settings.Capture,
	onCapture func(context.Context, Captured),
) *Watcher {
	return &Watcher{
		reader:    reader,
		frontmost: frontmost,
		capture:   capture,
		onCapture: onCapture,
		interval:  defaultPollInterval,
	}
}

// Run polls until ctx is cancelled.
func (w *Watcher) Run(ctx context.Context) {
	last := w.reader.ChangeCount()
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			current := w.reader.ChangeCount()
			if current == last {
				continue
			}
			last = current

			app := w.frontmost()
			payload, ok, err := readWithRetry(w.reader, w.capture())
			if err != nil || !ok {
				continue
			}
			w.onCapture(ctx, Captured{Payload: payload, App: app, ChangeCount: current})
		}
	}
}

func readWithRetry(reader Reader, capture settings.Capture) (Payload, bool, error) {
	payload, ok, err := reader.Read(capture)
	for _, delay := range readRetryDelays {
		if err == nil {
			return payload, ok, nil
		}
		time.Sleep(delay)
		payload, ok, err = reader.Read(capture)
	}
	return payload, ok, err
}
