package ui

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/nsgo-dev/nsgo/core"
)

// itemActions runs one row action off the main thread. It owns the error,
// loading and preview signals the panel shows, plus the reload and hide
// behavior the action results feed back into.
type itemActions struct {
	cx         *core.Scope
	model      HistoryModel
	ref        *core.Ref
	errorText  *core.Signal[string]
	loading    *core.Signal[bool]
	preview    *core.Signal[macos.PreviewContent]
	previewSeq atomic.Int64
	pinned     *bool
	hide       func()
	reload     func()
	schedule   func(time.Duration, func())
}

func (a *itemActions) copy(id string, forcePlain bool) {
	if id == "" || a.model == nil {
		return
	}
	ctx := a.cx.Context()
	go func() {
		copyAction := a.model.Copy
		if forcePlain {
			copyAction = a.model.CopyPlain
		}
		hideAfter, err := copyAction(ctx, id)
		a.cx.Post(func() {
			if err != nil {
				a.reportError(err)
				return
			}
			if hideAfterAction(*a.pinned, hideAfter) && a.hide != nil {
				a.hide()
			}
		})
	}()
}

func (a *itemActions) activate(id string) {
	if a.model == nil {
		return
	}
	a.activateWith(id, a.model.Activate)
}

func (a *itemActions) activateSingleClick(id string) {
	if a.model == nil {
		return
	}
	a.activateWith(id, a.model.ActivateSingleClick)
}

func (a *itemActions) activateDoubleClick(id string) {
	if a.model == nil {
		return
	}
	a.activateWith(id, a.model.ActivateDoubleClick)
}

func (a *itemActions) activateWith(id string, action func(context.Context, string) (bool, bool, error)) {
	if id == "" || a.model == nil {
		return
	}
	ctx := a.cx.Context()
	go func() {
		paste, hideAfter, err := action(ctx, id)
		a.cx.Post(func() {
			if err != nil {
				a.reportError(err)
				return
			}
			if hideAfterAction(*a.pinned, hideAfter) && a.hide != nil {
				a.hide()
			}
			if paste {
				if *a.pinned {
					a.ref.Command("resignKey", nil)
				}
				a.after(70*time.Millisecond, func() {
					if err := a.model.SendPaste(); err != nil {
						a.reportError(err)
						return
					}
					if *a.pinned {
						a.after(50*time.Millisecond, func() {
							a.ref.Command("makeKey", nil)
						})
					}
				})
				return
			}
		})
	}()
}

func (a *itemActions) after(delay time.Duration, fn func()) {
	if a.schedule != nil {
		a.schedule(delay, fn)
		return
	}
	a.cx.AfterFunc(delay, fn)
}

// run performs one item action. Actions that change the list ask for a
// reload once they succeed.
func (a *itemActions) run(id string, reloadAfter bool, run func(context.Context, string) error) {
	if id == "" || a.model == nil {
		return
	}
	ctx := a.cx.Context()
	go func() {
		err := run(ctx, id)
		a.cx.Post(func() {
			if err != nil {
				a.reportError(err)
				return
			}
			if reloadAfter {
				a.reload()
			}
		})
	}()
}

func (a *itemActions) showPreview(id string) {
	if id == "" || a.model == nil {
		return
	}
	// Serialize preview requests so rapid keyboard navigation lands on the
	// last selected row instead of a slower earlier response.
	seq := a.previewSeq.Add(1)
	ctx := a.cx.Context()
	go func() {
		content, err := a.model.Preview(ctx, id)
		a.cx.Post(func() {
			if a.previewSeq.Load() != seq {
				return
			}
			if err != nil {
				a.reportError(err)
				return
			}
			if !content.Enabled {
				return
			}
			a.preview.Set(macos.PreviewContent{
				Active:     true,
				Kind:       content.Kind,
				Title:      content.Title,
				Meta:       content.Meta,
				Text:       content.Text,
				RichText:   content.RichText,
				RichFormat: content.RichFormat,
				ImagePath:  content.ImagePath,
			})
		})
	}()
}

// clearPreview closes the preview and invalidates any in-flight preview
// request so a late response cannot reopen a dismissed popover.
func (a *itemActions) clearPreview() {
	a.previewSeq.Add(1)
	a.preview.Set(macos.PreviewContent{})
}

// reportError shows a failure and asks for the accessibility alert the
// first time macOS denies posting paste keystrokes.
func (a *itemActions) reportError(err error) {
	logHistoryFailure("item action", err)
	a.errorText.Set(err.Error())
	a.loading.Set(false)
	if a.model.NeedsAccessibility(err) {
		a.ref.Command("showAccessibilityAlert", nil)
	}
}

func logHistoryFailure(action string, err error) {
	slog.Error("history operation failed", "module", "ui.history", "action", action, "error", err)
}
