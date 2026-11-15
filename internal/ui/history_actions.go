package ui

import (
	"context"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/nsgo-dev/nsgo/core"
)

// itemActions runs one row action off the main thread. It owns the error,
// loading and preview signals the panel shows, plus the reload and hide
// behavior the action results feed back into.
type itemActions struct {
	cx        *core.Scope
	model     HistoryModel
	ref       *core.Ref
	errorText *core.Signal[string]
	loading   *core.Signal[bool]
	preview   *core.Signal[macos.PreviewContent]
	pinned    *bool
	hide      func()
	reload    func()
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
				a.errorText.Set(err.Error())
				return
			}
			if hideAfterAction(*a.pinned, hideAfter) && a.hide != nil {
				a.hide()
			}
			a.reload()
		})
	}()
}

func (a *itemActions) activate(id string, forcePlain bool) {
	if a.model == nil {
		return
	}
	action := a.model.Activate
	if forcePlain {
		action = a.model.ActivatePlain
	}
	a.activateWith(id, action)
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
				a.cx.AfterFunc(70*time.Millisecond, func() {
					if err := a.model.SendPaste(); err != nil {
						a.reportError(err)
						return
					}
					if *a.pinned {
						a.cx.AfterFunc(50*time.Millisecond, func() {
							a.ref.Command("makeKey", nil)
						})
					}
					a.reload()
				})
				return
			}
			a.reload()
		})
	}()
}

func (a *itemActions) middleClick(id string) {
	if id == "" || a.model == nil {
		return
	}
	ctx := a.cx.Context()
	go func() {
		hideAfter, err := a.model.MiddleClick(ctx, id)
		a.cx.Post(func() {
			if err != nil {
				a.reportError(err)
				return
			}
			if hideAfterAction(*a.pinned, hideAfter) && a.hide != nil {
				a.hide()
			}
			a.reload()
		})
	}()
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
				a.errorText.Set(err.Error())
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
	ctx := a.cx.Context()
	go func() {
		content, err := a.model.Preview(ctx, id)
		a.cx.Post(func() {
			if err != nil {
				a.errorText.Set(err.Error())
				return
			}
			if !content.Enabled {
				return
			}
			a.preview.Set(macos.PreviewContent{
				Active:     true,
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

// reportError shows a failure and asks for the accessibility alert the
// first time macOS denies posting paste keystrokes.
func (a *itemActions) reportError(err error) {
	a.errorText.Set(err.Error())
	a.loading.Set(false)
	if a.model.NeedsAccessibility(err) {
		a.ref.Command("showAccessibilityAlert", nil)
	}
}
