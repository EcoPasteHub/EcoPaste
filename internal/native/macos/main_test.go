//go:build darwin

package macos

import (
	"os"
	"testing"

	"github.com/ebitengine/purego/objc"
)

type mainJob struct {
	fn   func()
	done chan struct{}
}

func TestMain(m *testing.M) {
	// AppKit views belong on the main thread. Keep that thread draining a
	// queue of callbacks while the tests run elsewhere.
	app := objc.ID(class("NSApplication")).Send(sel("sharedApplication"))
	app.Send(sel("finishLaunching"))
	jobs := make(chan mainJob)
	runOnMain = func(fn func()) {
		done := make(chan struct{})
		jobs <- mainJob{fn: fn, done: done}
		<-done
	}
	code := 0
	go func() {
		code = m.Run()
		close(jobs)
	}()
	for job := range jobs {
		job.fn()
		close(job.done)
	}
	os.Exit(code)
}
