package app

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
)

func TestRuntimeOperationsCancelWithTheirOwner(t *testing.T) {
	for _, values := range []bool{false, true} {
		t.Run(map[bool]string{false: "operation", true: "value"}[values], func(t *testing.T) {
			parent, cancel := context.WithCancel(context.Background())
			defer cancel()
			started := make(chan struct{})
			done := make(chan error, 1)
			wait := func(ctx context.Context, _ *clipboardRuntime) error {
				close(started)
				<-ctx.Done()
				return ctx.Err()
			}
			go func() {
				runtime := &clipboardRuntime{ctx: parent}
				if values {
					_, err := runWithTimeoutValue(runtime, time.Hour, func(ctx context.Context, rt *clipboardRuntime) (int, error) {
						return 0, wait(ctx, rt)
					})
					done <- err
					return
				}
				done <- runWithTimeout(runtime, time.Hour, wait)
			}()
			select {
			case <-started:
			case <-time.After(time.Second):
				t.Fatal("operation did not start")
			}
			cancel()
			select {
			case err := <-done:
				if err != context.Canceled {
					t.Fatalf("operation returned %v, want cancellation", err)
				}
			case <-time.After(time.Second):
				t.Fatal("operation outlived its canceled owner")
			}
		})
	}
}

func TestSettingsWorkerUsesCapturedRuntimeContext(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	runtime := &clipboardRuntime{ctx: parent, db: db}
	// No graph is available on this worker. Context ownership was resolved
	// when the runtime was created, before settings started background work.
	wiring := &settingsWiring{runtime: func() *clipboardRuntime { return runtime }}
	done := make(chan error, 1)
	go func() {
		if _, err := wiring.listGroups(); err != nil {
			done <- err
			return
		}
		cancel()
		_, err := wiring.listGroups()
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("settings worker returned %v, want owner cancellation", err)
		}
	case <-time.After(time.Second):
		t.Fatal("settings worker did not finish")
	}
}
