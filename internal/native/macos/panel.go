//go:build darwin

package macos

import (
	"fmt"
	"sync"

	"github.com/ebitengine/purego/objc"
	"github.com/nsgo-dev/nsgo/core"
)

// PanelComponent is the native AppKit view that fills the clipboard
// panel.
const PanelComponent = "ecopaste.panel"

// Panel is the AppKit content view of the clipboard panel: search, the
// native history list and a compact footer.
type Panel struct {
	cx        *core.NativeContext
	view      objc.ID
	search    objc.ID
	divider   objc.ID
	scroll    objc.ID
	table     objc.ID
	empty     objc.ID
	footer    objc.ID
	hint      objc.ID
	observer  objc.ID
	timer     objc.ID
	query     string
	rows      []HistoryRow
	total     int64
	loading   bool
	errText   string
	selection string
}

var (
	registerPanelOnce sync.Once
	searchTarget      objc.ID
	observerClass     objc.Class
	panelViewClass    objc.Class

	panelsBySearch   = map[objc.ID]*Panel{}
	panelsByObserver = map[objc.ID]*Panel{}
	panelsByView     = map[objc.ID]*Panel{}
)

// NewPanel makes the panel's AppKit view.
func NewPanel(cx *core.NativeContext) core.NativeComponent {
	registerPanelOnce.Do(registerPanelTargets)

	p := &Panel{cx: cx, view: objc.ID(panelViewClass).Send(sel("new"))}
	p.view.Send(sel("setAutoresizesSubviews:"), false)

	p.search = objc.ID(classNSSearchField).Send(sel("new"))
	p.search.Send(sel("setPlaceholderString:"), nsString("搜索剪贴板"))
	p.search.Send(sel("setFont:"), systemFont(13))
	p.search.Send(sel("setSendsSearchStringImmediately:"), true)
	p.search.Send(sel("setSendsWholeSearchString:"), false)
	p.search.Send(sel("setTarget:"), searchTarget)
	p.search.Send(sel("setAction:"), sel("searchChanged:"))

	p.divider = objc.ID(classNSBox).Send(sel("new"))
	p.divider.Send(sel("setBoxType:"), 2)
	p.divider.Send(sel("setTitlePosition:"), 0)

	p.empty = objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString("暂无剪贴板记录"))
	p.empty.Send(sel("setFont:"), systemFont(13))
	p.empty.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))

	p.footer = objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
	p.footer.Send(sel("setFont:"), systemFont(11))
	p.footer.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("tertiaryLabelColor")))

	p.hint = objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString("↵ 复制   esc 关闭"))
	p.hint.Send(sel("setFont:"), systemFont(11))
	p.hint.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("tertiaryLabelColor")))
	p.hint.Send(sel("setAlignment:"), int64(1))

	p.scroll = newHistoryTable(p)

	for _, v := range []objc.ID{p.search, p.divider, p.scroll, p.empty, p.footer, p.hint} {
		p.view.Send(sel("addSubview:"), v)
	}

	panelsBySearch[p.search] = p
	panelsByView[p.view] = p
	layoutPanel(p, 420, 560)

	p.observer = objc.ID(observerClass).Send(sel("new"))
	panelsByObserver[p.observer] = p
	center := objc.ID(class("NSNotificationCenter")).Send(sel("defaultCenter"))
	center.Send(sel("addObserver:selector:name:object:"), p.observer, sel("windowBecameKey:"),
		nsString(windowDidBecomeKey), objc.ID(0))
	p.timer = objc.ID(class("NSTimer")).Send(sel("scheduledTimerWithTimeInterval:target:selector:userInfo:repeats:"),
		0.1, p.observer, sel("pollSearch:"), objc.ID(0), true)

	return p
}

func registerPanelTargets() {
	var err error
	panelViewClass, err = objc.RegisterClass("EcoPastePanelView", class("NSView"), nil, nil,
		[]objc.MethodDef{{Cmd: sel("setFrameSize:"), Fn: func(self objc.ID, _ objc.SEL, size CGSize) {
			sendSuper[objc.ID](self, panelViewClass, sel("setFrameSize:"), size)
			if p := panelsByView[self]; p != nil {
				layoutPanel(p, size.Width, size.Height)
			}
		}}})
	if err != nil {
		panic(err)
	}

	target, err := objc.RegisterClass("EcoPasteSearchTarget", class("NSObject"), nil, nil,
		[]objc.MethodDef{{Cmd: sel("searchChanged:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			if p := panelsBySearch[sender]; p != nil {
				p.query = goString(sender.Send(sel("stringValue")))
				p.cx.Emit("query", p.query)
			}
		}}})
	if err != nil {
		panic(err)
	}
	searchTarget = objc.ID(target)

	observerClass, err = objc.RegisterClass("EcoPastePanelObserver", class("NSObject"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("windowBecameKey:"), Fn: func(self objc.ID, _ objc.SEL, note objc.ID) {
				p := panelsByObserver[self]
				if p == nil {
					return
				}
				window := p.view.Send(sel("window"))
				if window != 0 && window == objc.Send[objc.ID](note, sel("object")) {
					window.Send(sel("makeFirstResponder:"), p.search)
				}
			}},
			{Cmd: sel("pollSearch:"), Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				p := panelsByObserver[self]
				if p == nil {
					return
				}
				next := goString(p.search.Send(sel("stringValue")))
				if next != p.query {
					p.query = next
					p.cx.Emit("query", next)
				}
			}},
		})
	if err != nil {
		panic(err)
	}
	registerHistoryTargets()
}

func layoutPanel(p *Panel, width, height float64) {
	const pad = 12.0
	searchHeight := 28.0
	setFrame(p.search, pad, height-pad-searchHeight, width-2*pad, searchHeight)
	dividerY := height - pad - searchHeight - 10
	setFrame(p.divider, 0, dividerY, width, 1)

	footerHeight := 28.0
	footerWidth := width/2 - pad
	hintWidth := width/2 - pad
	if p.errText != "" {
		footerWidth = width - 2*pad
		hintWidth = 0
	}
	p.hint.Send(sel("setHidden:"), p.errText != "")
	setFrame(p.footer, pad, 8, footerWidth, 16)
	setFrame(p.hint, width/2, 8, hintWidth, 16)
	scrollY := footerHeight + 4
	scrollHeight := dividerY - scrollY - 4
	setFrame(p.scroll, 0, scrollY, width, scrollHeight)
	if p.table != 0 {
		setFrame(p.table, 0, 0, width, scrollHeight)
		if column := p.table.Send(sel("tableColumnWithIdentifier:"), nsString("history")); column != 0 {
			column.Send(sel("setWidth:"), width)
		}
	}

	p.empty.Send(sel("sizeToFit"))
	text := objc.Send[CGRect](p.empty, sel("frame"))
	emptyCenter := scrollY + scrollHeight/2
	setFrame(p.empty, (width-text.Size.Width)/2, emptyCenter-text.Size.Height/2, text.Size.Width, text.Size.Height)
}

// View implements core.NativeComponent.
func (p *Panel) View() any { return p.view }

// SetProp implements core.NativeComponent.
func (p *Panel) SetProp(name string, value any) {
	switch name {
	case "query":
		if s, ok := value.(string); ok && s != p.query {
			p.query = s
			p.search.Send(sel("setStringValue:"), nsString(s))
		}
	case "placeholder":
		if s, ok := value.(string); ok {
			p.search.Send(sel("setPlaceholderString:"), nsString(s))
		}
	case "rows":
		rows, _ := value.([]HistoryRow)
		p.rows = rows
		p.table.Send(sel("reloadData"))
		p.restoreSelection()
		p.updateStatus()
	case "total":
		if n, ok := int64Value(value); ok {
			p.total = n
			p.updateStatus()
		}
	case "loading":
		if b, ok := value.(bool); ok {
			p.loading = b
			p.updateStatus()
		}
	case "error":
		if s, ok := value.(string); ok {
			p.errText = s
			p.updateStatus()
		}
	}
}

// Command implements core.NativeComponent.
func (p *Panel) Command(name string, args any) {
	switch name {
	case "focusSearch":
		p.view.Send(sel("window")).Send(sel("makeFirstResponder:"), p.search)
	case "clearSearch":
		p.search.Send(sel("setStringValue:"), nsString(""))
		p.query = ""
	case "reload":
		p.table.Send(sel("reloadData"))
	}
}

// Close implements core.NativeComponent.
func (p *Panel) Close() {
	center := objc.ID(class("NSNotificationCenter")).Send(sel("defaultCenter"))
	center.Send(sel("removeObserver:"), p.observer)
	p.timer.Send(sel("invalidate"))
	delete(panelsByObserver, p.observer)
	delete(panelsBySearch, p.search)
	delete(panelsByView, p.view)
	delete(panelsByTable, p.table)
	p.observer.Send(sel("release"))
	p.hint.Send(sel("release"))
	p.footer.Send(sel("release"))
	p.empty.Send(sel("release"))
	p.divider.Send(sel("release"))
	p.scroll.Send(sel("release"))
	p.search.Send(sel("release"))
	p.view.Send(sel("release"))
}

func (p *Panel) updateStatus() {
	if p.errText != "" {
		if len(p.rows) == 0 {
			p.empty.Send(sel("setStringValue:"), nsString(p.errText))
			p.empty.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("systemRedColor")))
			p.empty.Send(sel("setHidden:"), false)
		} else {
			p.empty.Send(sel("setHidden:"), true)
		}
	} else if len(p.rows) == 0 {
		text := "暂无剪贴板记录"
		if p.query != "" {
			text = "没有匹配结果"
		}
		if p.loading {
			text = "正在加载…"
		}
		p.empty.Send(sel("setStringValue:"), nsString(text))
		p.empty.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
		p.empty.Send(sel("setHidden:"), false)
	} else {
		p.empty.Send(sel("setHidden:"), true)
	}
	if p.errText != "" {
		p.footer.Send(sel("setStringValue:"), nsString(p.errText))
		p.footer.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("systemRedColor")))
	} else if p.loading {
		p.footer.Send(sel("setStringValue:"), nsString("正在加载…"))
		p.footer.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("tertiaryLabelColor")))
	} else {
		p.footer.Send(sel("setStringValue:"), nsString(fmt.Sprintf("%d 条记录", p.total)))
		p.footer.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("tertiaryLabelColor")))
	}
	p.empty.Send(sel("sizeToFit"))
	frame := bounds(p.view)
	layoutPanel(p, frame.Size.Width, frame.Size.Height)
}

func (p *Panel) restoreSelection() {
	if p.selection == "" {
		return
	}
	for i, row := range p.rows {
		if row.ID == p.selection {
			indexes := objc.ID(class("NSIndexSet")).Send(sel("indexSetWithIndex:"), int64(i))
			p.table.Send(sel("selectRowIndexes:byExtendingSelection:"), indexes, false)
			return
		}
	}
	p.selection = ""
}

func (p *Panel) selectRow(index int) {
	if index < 0 || index >= len(p.rows) {
		return
	}
	p.selection = p.rows[index].ID
	indexes := objc.ID(class("NSIndexSet")).Send(sel("indexSetWithIndex:"), int64(index))
	p.table.Send(sel("selectRowIndexes:byExtendingSelection:"), indexes, false)
}

func int64Value(value any) (int64, bool) {
	switch n := value.(type) {
	case int64:
		return n, true
	case int:
		return int64(n), true
	case float64:
		return int64(n), true
	}
	return 0, false
}
