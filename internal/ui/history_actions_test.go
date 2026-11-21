package ui

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/nsgo-dev/nsgo/core"
)

// stubHistoryModel implements only the calls itemActions makes in tests.
// The embedded nil interface covers the rest of HistoryModel.
type stubHistoryModel struct {
	HistoryModel
	hideAfter   bool
	err         error
	preview     HistoryPreview
	previewErr  error
	pasteErr    error
	pastes      int
	accessError bool
}

func (s *stubHistoryModel) Copy(context.Context, string) (bool, error) {
	return s.hideAfter, s.err
}

func (s *stubHistoryModel) CopyPlain(context.Context, string) (bool, error) {
	return s.hideAfter, s.err
}

func (s *stubHistoryModel) Activate(context.Context, string) (bool, bool, error) {
	return true, s.hideAfter, s.err
}

func (s *stubHistoryModel) SendPaste() error {
	s.pastes++
	return s.pasteErr
}

func (s *stubHistoryModel) NeedsAccessibility(error) bool { return s.accessError }

func (s *stubHistoryModel) Preview(context.Context, string) (HistoryPreview, error) {
	return s.preview, s.previewErr
}

type actionsHarness struct {
	actions   *itemActions
	errorText *core.Signal[string]
	loading   *core.Signal[bool]
	preview   *core.Signal[macos.PreviewContent]
	reloads   chan struct{}
	posted    chan struct{}
	pending   *sync.WaitGroup
	hides     int
	pinned    bool
}

func newActionsHarness(t *testing.T, model HistoryModel) *actionsHarness {
	t.Helper()
	// Serialize posted callbacks on one goroutine, like the app's graph, so
	// the harness does not race with the test's own reads.
	queue := make(chan func())
	var posted sync.WaitGroup
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		for fn := range queue {
			fn()
		}
	}()
	postedCh := make(chan struct{}, 8)
	cx := core.NewRoot(core.RootOptions{Post: func(fn func()) {
		posted.Add(1)
		queue <- func() {
			defer posted.Done()
			fn()
		}
		select {
		case postedCh <- struct{}{}:
		default:
		}
	}})
	t.Cleanup(cx.Dispose)
	t.Cleanup(func() { close(queue); <-stopped; posted.Wait() })

	h := &actionsHarness{
		errorText: core.NewSignal(cx, ""),
		loading:   core.NewSignal(cx, true),
		preview:   core.NewSignal(cx, macos.PreviewContent{}),
		reloads:   make(chan struct{}, 8),
		posted:    postedCh,
		pending:   &posted,
	}
	h.actions = &itemActions{
		cx:        cx,
		model:     model,
		ref:       &core.Ref{},
		errorText: h.errorText,
		loading:   h.loading,
		preview:   h.preview,
		pinned:    &h.pinned,
		hide:      func() { h.hides++ },
		schedule:  func(_ time.Duration, fn func()) { fn() },
		reload: func() {
			select {
			case h.reloads <- struct{}{}:
			default:
			}
		},
	}
	return h
}

func (h *actionsHarness) waitReload(t *testing.T) {
	t.Helper()
	select {
	case <-h.reloads:
	case <-time.After(2 * time.Second):
		t.Fatal("action did not reload the list")
	}
}

// waitSettled waits for the action to post its result and for that result to
// run, so the test can read signals without racing the action goroutine.
func (h *actionsHarness) waitSettled(t *testing.T) {
	t.Helper()
	select {
	case <-h.posted:
	case <-time.After(2 * time.Second):
		t.Fatal("action posted no result")
	}
	h.pending.Wait()
}

func (h *actionsHarness) waitError(t *testing.T, want string) {
	t.Helper()
	h.waitSettled(t)
	if got := h.errorText.GetUntracked(); got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func (h *actionsHarness) expectNoReload(t *testing.T) {
	t.Helper()
	select {
	case <-h.reloads:
		t.Fatal("copy or paste reloaded the list")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestItemActionsCopyHidesWithoutReloading(t *testing.T) {
	model := &stubHistoryModel{hideAfter: true}
	h := newActionsHarness(t, model)
	h.actions.copy("1", true)
	h.waitSettled(t)
	h.expectNoReload(t)
	if h.hides != 1 {
		t.Errorf("hide calls = %d", h.hides)
	}
	if got := h.errorText.GetUntracked(); got != "" {
		t.Errorf("error = %q", got)
	}
}

func TestItemActionsCopyReportsFailureWithoutReloading(t *testing.T) {
	model := &stubHistoryModel{err: errors.New("copy failed")}
	h := newActionsHarness(t, model)
	h.actions.copy("1", false)
	h.waitError(t, "copy failed")
	h.expectNoReload(t)
	if h.hides != 0 {
		t.Errorf("hide calls = %d", h.hides)
	}
}

func TestItemActionsActivatePastesWithoutReloading(t *testing.T) {
	model := &stubHistoryModel{}
	h := newActionsHarness(t, model)
	h.actions.activate("1")
	h.waitSettled(t)
	h.expectNoReload(t)
	if model.pastes != 1 {
		t.Errorf("paste calls = %d", model.pastes)
	}
	if !h.loading.GetUntracked() {
		t.Error("activate cleared the loading state")
	}
}

func TestItemActionsActivateReportsPermissionFailure(t *testing.T) {
	model := &stubHistoryModel{err: errors.New("permission"), accessError: true}
	h := newActionsHarness(t, model)
	h.actions.activate("1")
	h.waitError(t, "permission")
	if h.loading.GetUntracked() {
		t.Error("loading stayed set after a failure")
	}
	h.expectNoReload(t)
}

func TestItemActionsPreviewPublishesEnabledContent(t *testing.T) {
	model := &stubHistoryModel{preview: HistoryPreview{Enabled: true, Title: "hello", Text: "body"}}
	h := newActionsHarness(t, model)
	h.actions.showPreview("1")
	h.waitSettled(t)
	content := h.preview.GetUntracked()
	if !content.Active || content.Title != "hello" || content.Text != "body" {
		t.Errorf("preview = %+v", content)
	}
}
