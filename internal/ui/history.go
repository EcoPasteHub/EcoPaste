package ui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/nsgo-dev/nsgo/core"
)

// HistoryItem is one row the list can show without knowing how history
// is stored.
type HistoryItem struct {
	ID        string
	Kind      string
	Title     string
	SourceApp string
	Note      string
	ImagePath string
	CreatedAt time.Time
	Favorite  bool
	Pinned    bool
}

// HistoryPage is one page of history rows.
type HistoryPage struct {
	Items   []HistoryItem
	Total   int64
	HasMore bool
}

// HistoryQuery selects a page of history.
type HistoryQuery struct {
	Keyword string
	Limit   int
	Offset  int
}

// HistoryModel is the app service the history panel consumes. Copy
// reports whether the panel should close after copying.
type HistoryModel interface {
	List(context.Context, HistoryQuery) (HistoryPage, error)
	Copy(context.Context, string) (hide bool, err error)
	Activate(context.Context, string) (paste bool, hide bool, err error)
	SendPaste() error
	NeedsAccessibility(error) bool
	ToggleFavorite(context.Context, string) (bool, error)
	TogglePinned(context.Context, string) (bool, error)
	Delete(context.Context, string) error
	Updates() <-chan struct{}
}

// ClipboardPanel builds the history panel. hide closes the panel after
// an action that the user configured to close it.
func ClipboardPanel(model HistoryModel, hide func()) func(*core.Scope) core.Node {
	return func(cx *core.Scope) core.Node {
		rows := core.NewSignalFunc(cx, []macos.HistoryRow(nil), nil)
		total := core.NewSignal(cx, int64(0))
		loading := core.NewSignal(cx, true)
		errorText := core.NewSignal(cx, "")
		hasMore := core.NewSignal(cx, false)
		ref := &core.Ref{}

		var (
			query      string
			selectedID string
			pageSize   = 80
			request    atomic.Int64
		)

		load := func(keyword string, appendRows bool) {
			if model == nil {
				errorText.Set("无法打开剪贴板数据库")
				loading.Set(false)
				return
			}
			loading.Set(true)
			errorText.Set("")
			offset := 0
			if appendRows {
				offset = len(rows.GetUntracked())
			}
			serial := request.Add(1)
			ctx := cx.Context()
			go func(keyword string, offset int, appendRows bool, serial int64) {
				page, err := model.List(ctx, HistoryQuery{
					Keyword: keyword,
					Limit:   pageSize,
					Offset:  offset,
				})
				cx.Post(func() {
					if request.Load() != serial {
						return
					}
					if err != nil {
						errorText.Set(err.Error())
						loading.Set(false)
						return
					}
					next := make([]macos.HistoryRow, 0, len(page.Items)+offset)
					if appendRows {
						next = append(next, rows.GetUntracked()...)
					}
					for _, item := range page.Items {
						next = append(next, historyRow(item))
					}
					rows.Set(next)
					total.Set(page.Total)
					hasMore.Set(page.HasMore)
					loading.Set(false)
				})
			}(keyword, offset, appendRows, serial)
		}

		copyItem := func(id string) {
			if id == "" || model == nil {
				return
			}
			ctx := cx.Context()
			go func() {
				hideAfter, err := model.Copy(ctx, id)
				cx.Post(func() {
					if err != nil {
						errorText.Set(err.Error())
						return
					}
					if hideAfter && hide != nil {
						hide()
					}
					load(query, false)
				})
			}()
		}

		activateItem := func(id string) {
			if id == "" || model == nil {
				return
			}
			ctx := cx.Context()
			go func() {
				paste, hideAfter, err := model.Activate(ctx, id)
				cx.Post(func() {
					if err != nil {
						errorText.Set(err.Error())
						loading.Set(false)
						if model.NeedsAccessibility(err) {
							ref.Command("showAccessibilityAlert", nil)
						}
						return
					}
					if hideAfter && hide != nil {
						hide()
					}
					if paste {
						cx.AfterFunc(70*time.Millisecond, func() {
							if err := model.SendPaste(); err != nil {
								errorText.Set(err.Error())
								if model.NeedsAccessibility(err) {
									ref.Command("showAccessibilityAlert", nil)
								}
								return
							}
							load(query, false)
						})
					} else {
						load(query, false)
					}
				})
			}()
		}

		applyAction := func(id string, run func(context.Context, string) error) {
			if id == "" || model == nil {
				return
			}
			ctx := cx.Context()
			go func() {
				err := run(ctx, id)
				cx.Post(func() {
					if err != nil {
						errorText.Set(err.Error())
						return
					}
					load(query, false)
				})
			}()
		}

		var pendingSearch func() bool
		cx.Cleanup(func() {
			if pendingSearch != nil {
				pendingSearch()
			}
		})

		load("", false)
		if model != nil {
			if updates := model.Updates(); updates != nil {
				ctx := cx.Context()
				go func() {
					for {
						select {
						case <-ctx.Done():
							return
						case <-updates:
							cx.Post(func() { load(query, false) })
						}
					}
				}()
			}
		}
		return core.Native{
			Component: macos.PanelComponent,
			Ref:       ref,
			Style:     core.Style{Width: core.Pct(100), Height: core.Pct(100)},
			PropsFunc: func() map[string]any {
				return map[string]any{
					"rows":    rows.Get(),
					"total":   total.Get(),
					"loading": loading.Get(),
					"error":   errorText.Get(),
				}
			},
			OnEvent: func(name string, data any) {
				switch name {
				case "query":
					text, _ := data.(string)
					if pendingSearch != nil {
						pendingSearch()
					}
					pendingSearch = cx.AfterFunc(120*time.Millisecond, func() {
						query = text
						load(query, false)
					})
				case "select":
					selectedID, _ = data.(string)
				case "activate":
					id := selectedID
					if value, ok := data.(string); ok && value != "" {
						id = value
					}
					activateItem(id)
				case "copy":
					id := selectedID
					if value, ok := data.(string); ok && value != "" {
						id = value
					}
					copyItem(id)
				case "toggleFavorite":
					applyAction(selectedID, func(ctx context.Context, id string) error {
						_, err := model.ToggleFavorite(ctx, id)
						return err
					})
				case "togglePinned":
					applyAction(selectedID, func(ctx context.Context, id string) error {
						_, err := model.TogglePinned(ctx, id)
						return err
					})
				case "delete":
					if selectedID == "" || model == nil {
						return
					}
					id := selectedID
					ctx := cx.Context()
					go func() {
						err := model.Delete(ctx, id)
						cx.Post(func() {
							if err != nil {
								errorText.Set(err.Error())
								return
							}
							if selectedID == id {
								selectedID = ""
							}
							load(query, false)
						})
					}()
				case "loadMore":
					if hasMore.GetUntracked() && !loading.GetUntracked() {
						load(query, true)
					}
				case "hide":
					if hide != nil {
						hide()
					}
				}
			},
		}
	}
}

func historyRow(item HistoryItem) macos.HistoryRow {
	title := strings.TrimSpace(item.Title)
	if title == "" {
		title = kindName(item.Kind)
	}
	detail := make([]string, 0, 4)
	if item.SourceApp != "" {
		detail = append(detail, item.SourceApp)
	}
	if !item.CreatedAt.IsZero() {
		detail = append(detail, relativeTime(time.Now(), item.CreatedAt))
	}
	if item.Note != "" {
		detail = append(detail, strings.TrimSpace(item.Note))
	}
	if item.Pinned {
		detail = append(detail, "已固定")
	}
	if item.Favorite {
		detail = append(detail, "收藏")
	}
	return macos.HistoryRow{
		ID:        item.ID,
		Kind:      item.Kind,
		Title:     title,
		Detail:    strings.Join(detail, " · "),
		ImagePath: item.ImagePath,
		Favorite:  item.Favorite,
		Pinned:    item.Pinned,
	}
}

func kindName(kind string) string {
	switch kind {
	case "image":
		return "图片"
	case "files":
		return "文件"
	default:
		return "文本"
	}
}

func relativeTime(now, then time.Time) string {
	if then.IsZero() {
		return ""
	}
	diff := now.Sub(then)
	switch {
	case diff < time.Minute:
		return "刚刚"
	case diff < time.Hour:
		return fmt.Sprintf("%d 分钟前", int(diff/time.Minute))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%d 小时前", int(diff/time.Hour))
	case diff < 48*time.Hour:
		return "昨天"
	case diff < 7*24*time.Hour:
		return fmt.Sprintf("%d 天前", int(diff/(24*time.Hour)))
	default:
		return then.Local().Format("01-02")
	}
}

// FileTitle summarizes paths by their base names, as the list shows
// them.
func FileTitle(paths []string, max int) string {
	if len(paths) == 0 {
		return ""
	}
	names := make([]string, 0, min(len(paths), max))
	for i, path := range paths {
		if max > 0 && i == max {
			break
		}
		names = append(names, filepath.Base(path))
	}
	title := strings.Join(names, "、")
	if max > 0 && len(paths) > max {
		title += fmt.Sprintf(" 等 %d 个", len(paths))
	}
	return title
}
