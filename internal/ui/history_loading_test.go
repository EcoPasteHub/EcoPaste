package ui

import (
	"context"
	"testing"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
)

type historyListCall struct {
	ctx    context.Context
	query  HistoryQuery
	result chan HistoryPage
}

type loadingModel struct {
	behaviorModel
	calls chan historyListCall
}

func (m *loadingModel) Groups(context.Context) ([]GroupOption, error) { return nil, nil }
func (m *loadingModel) Updates() <-chan struct{}                      { return nil }

func (m *loadingModel) List(ctx context.Context, query HistoryQuery) (HistoryPage, error) {
	call := historyListCall{ctx: ctx, query: query, result: make(chan HistoryPage, 1)}
	m.calls <- call
	select {
	case page := <-call.result:
		return page, nil
	case <-ctx.Done():
		return HistoryPage{}, ctx.Err()
	}
}

func newLoadingPanel(t *testing.T) (*historyPanel, <-chan historyListCall, chan func(), *core.ManualClock) {
	t.Helper()
	posts := make(chan func(), 32)
	clock := core.NewManualClock(time.Now())
	cx := core.NewRoot(core.RootOptions{Post: func(fn func()) { posts <- fn }, Clock: clock})
	t.Cleanup(cx.Dispose)
	model := &loadingModel{calls: make(chan historyListCall, 32)}
	return newHistoryPanel(cx, model, nil), model.calls, posts, clock
}

func nextListCall(t *testing.T, calls <-chan historyListCall) historyListCall {
	t.Helper()
	select {
	case call := <-calls:
		return call
	case <-time.After(2 * time.Second):
		t.Fatal("history query did not start")
		return historyListCall{}
	}
}

func runHistoryPost(t *testing.T, posts <-chan func()) {
	t.Helper()
	select {
	case fn := <-posts:
		fn()
	case <-time.After(2 * time.Second):
		t.Fatal("history query posted no result")
	}
}

func TestHiddenHistoryDefersRefreshUntilOpen(t *testing.T) {
	p, calls, posts, _ := newLoadingPanel(t)
	for range 100 {
		p.reload()
	}
	if p.request.Load() != 0 {
		t.Fatal("hidden panel started a history request")
	}
	p.setVisible(true)
	call := nextListCall(t, calls)
	call.result <- HistoryPage{Items: []HistoryItem{{ID: "first", Title: "first"}}, Total: 1}
	runHistoryPost(t, posts)
	if got := p.items.GetUntracked(); len(got) != 1 || got[0].ID != "first" {
		t.Fatalf("loaded items = %+v", got)
	}
	p.setVisible(false)
	serial := p.request.Load()
	for range 100 {
		p.reload()
	}
	if p.request.Load() != serial {
		t.Fatal("hidden updates started history requests")
	}
	p.setVisible(true)
	call = nextListCall(t, calls)
	call.result <- HistoryPage{Items: []HistoryItem{{ID: "latest"}}, Total: 1}
	runHistoryPost(t, posts)
	if got := p.items.GetUntracked(); len(got) != 1 || got[0].ID != "latest" {
		t.Fatalf("reopened items = %+v", got)
	}
	select {
	case <-calls:
		t.Fatal("opening started duplicate requests")
	default:
	}
}

func TestOpeningHistoryWithoutServiceShowsError(t *testing.T) {
	cx := core.NewRoot(core.RootOptions{})
	t.Cleanup(cx.Dispose)
	p := newHistoryPanel(cx, nil, nil)
	p.setVisible(true)
	if p.loading.GetUntracked() || p.errorText.GetUntracked() != i18n.T(i18n.DatabaseOpenFailed) {
		t.Fatalf("unavailable service: loading=%t, error=%q", p.loading.GetUntracked(), p.errorText.GetUntracked())
	}
}

func TestHistoryFollowsWindowVisibilityWithoutNativeNotifications(t *testing.T) {
	posts := make(chan func(), 32)
	cx := core.NewRoot(core.RootOptions{
		Post:    func(fn func()) { posts <- fn },
		Windows: &core.WindowOptions{Width: 420, Height: 560},
	})
	t.Cleanup(cx.Dispose)
	model := &loadingModel{calls: make(chan historyListCall, 32)}
	var p *historyPanel
	var window *desktop.Window
	tree := core.NewTree(cx, core.ApproxTextEngine{})
	tree.Mount(core.AppRoot(func(app *core.Scope) {
		window = desktop.OpenPanel(app, desktop.PanelOptions{}, func(panel *core.Scope) core.Node {
			p = newHistoryPanel(panel, model, nil)
			p.start()
			return p.node()
		})
	}))
	commit := func() {
		tree.Commit()
		for _, surface := range tree.Surfaces() {
			surface.Commit()
		}
	}
	commit()
	if p.visible || p.request.Load() != 0 {
		t.Fatal("hidden window started a history query")
	}
	window.Show()
	commit()
	first := nextListCall(t, model.calls)
	serial := p.request.Load()
	p.setVisible(true)
	if p.request.Load() != serial {
		t.Fatal("unchanged visibility duplicated the history refresh")
	}
	window.Hide()
	commit()
	if p.visible || first.ctx.Err() != context.Canceled {
		t.Fatal("hidden window left its history query active")
	}
	window.Show()
	commit()
	second := nextListCall(t, model.calls)
	if second.ctx.Err() != nil {
		t.Fatal("reopened window did not start a fresh query")
	}
}

func TestSearchCancelsOldQueryAndKeepsLatestTextOnHide(t *testing.T) {
	p, calls, posts, clock := newLoadingPanel(t)
	p.setVisible(true)
	first := nextListCall(t, calls)
	p.handleListEvent("query", "new search")
	if first.ctx.Err() != context.Canceled {
		t.Fatal("new search left old query running")
	}
	runHistoryPost(t, posts)
	if p.errorText.GetUntracked() != "" {
		t.Fatal("canceled request displayed an error")
	}
	p.setVisible(false)
	clock.Advance(time.Second)
	if len(posts) != 0 {
		t.Fatal("hidden search retained its debounce timer")
	}
	p.setVisible(true)
	latest := nextListCall(t, calls)
	if latest.query.Keyword != "new search" {
		t.Fatalf("reopened keyword = %q", latest.query.Keyword)
	}
	latest.result <- HistoryPage{}
	runHistoryPost(t, posts)
}

func TestFilterConsumesPendingSearchWithoutDuplicateOrMixedPages(t *testing.T) {
	p, calls, posts, clock := newLoadingPanel(t)
	p.setVisible(true)
	first := nextListCall(t, calls)
	first.result <- HistoryPage{Items: []HistoryItem{{ID: "old"}}, Total: 100, HasMore: true}
	runHistoryPost(t, posts)
	p.handleListEvent("query", "latest")
	serial := p.request.Load()
	p.handleListEvent("loadMore", nil)
	if p.request.Load() != serial {
		t.Fatal("pending search appended a page to the old results")
	}
	p.handleListEvent("filter", "favorite")
	filtered := nextListCall(t, calls)
	if filtered.query.Keyword != "latest" || filtered.query.Filter != "favorite" || filtered.query.Offset != 0 {
		t.Fatalf("filtered query = %+v", filtered.query)
	}
	clock.Advance(time.Second)
	if len(posts) != 0 {
		t.Fatal("filter change left a duplicate search timer")
	}
	serial = p.request.Load()
	p.handleListEvent("filter", "favorite")
	if p.request.Load() != serial {
		t.Fatal("selecting the current filter repeated its query")
	}
	filtered.result <- HistoryPage{}
	runHistoryPost(t, posts)
}

func TestNewLoadCancelsPreviousRequestAndDisposeCancelsLast(t *testing.T) {
	p, calls, posts, _ := newLoadingPanel(t)
	p.visible = true
	p.reload()
	first := nextListCall(t, calls)
	p.reload()
	second := nextListCall(t, calls)
	if first.ctx.Err() != context.Canceled {
		t.Fatal("superseded history request was not canceled")
	}
	runHistoryPost(t, posts)
	if !p.loading.GetUntracked() || p.errorText.GetUntracked() != "" {
		t.Fatal("old result changed the new request's state")
	}
	p.cx.Dispose()
	if second.ctx.Err() != context.Canceled {
		t.Fatal("disposed panel left its query running")
	}
}

func TestHistoryRowsAreMemoizedAcrossUnrelatedProps(t *testing.T) {
	p, _, _, _ := newLoadingPanel(t)
	p.items.Set([]HistoryItem{{ID: "first", Kind: "text", Title: "content"}})
	node := p.node().(core.Native)
	first := node.PropsFunc()["rows"].([]macos.HistoryRow)
	p.loading.Set(false)
	p.preview.Set(macos.PreviewContent{Active: true})
	second := node.PropsFunc()["rows"].([]macos.HistoryRow)
	if &first[0] != &second[0] {
		t.Fatal("unrelated state rebuilt all history rows")
	}
	i18n.SetLanguage("en-US")
	t.Cleanup(func() { i18n.SetLanguage("zh-CN") })
	// The test root is not bound to the global language signal; a new item
	// snapshot still has to invalidate the presentation memo.
	p.items.Set([]HistoryItem{{ID: "second", Kind: "text", Title: "new"}})
	third := node.PropsFunc()["rows"].([]macos.HistoryRow)
	if third[0].ID != "second" || third[0].Detail != "Text" {
		t.Fatalf("updated presentation = %+v", third)
	}
}
