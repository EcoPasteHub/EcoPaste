//go:build darwin

package macos

import (
	"sync"

	"github.com/ebitengine/purego/objc"
	"github.com/nsgo-dev/nsgo/core"
)

// PanelComponent is the native AppKit view that fills the clipboard
// panel.
const PanelComponent = "ecopaste.panel"

// Panel is the AppKit content view of the clipboard panel: a search
// field, a separator and, for now, an empty state.
type Panel struct {
	cx       *core.NativeContext
	view     objc.ID
	search   objc.ID
	divider  objc.ID
	empty    objc.ID
	observer objc.ID
	query    string
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

	for _, v := range []objc.ID{p.search, p.divider, p.empty} {
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
		[]objc.MethodDef{{Cmd: sel("windowBecameKey:"), Fn: func(self objc.ID, _ objc.SEL, note objc.ID) {
			p := panelsByObserver[self]
			if p == nil {
				return
			}
			window := p.view.Send(sel("window"))
			if window != 0 && window == objc.Send[objc.ID](note, sel("object")) {
				window.Send(sel("makeFirstResponder:"), p.search)
			}
		}}})
	if err != nil {
		panic(err)
	}
}

func layoutPanel(p *Panel, width, height float64) {
	const pad = 12.0
	searchHeight := 28.0
	setFrame(p.search, pad, height-pad-searchHeight, width-2*pad, searchHeight)
	dividerY := height - pad - searchHeight - 10
	setFrame(p.divider, 0, dividerY, width, 1)
	p.empty.Send(sel("sizeToFit"))
	text := objc.Send[CGRect](p.empty, sel("frame"))
	setFrame(p.empty, (width-text.Size.Width)/2, dividerY/2-text.Size.Height/2, text.Size.Width, text.Size.Height)
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
	}
}

// Close implements core.NativeComponent.
func (p *Panel) Close() {
	center := objc.ID(class("NSNotificationCenter")).Send(sel("defaultCenter"))
	center.Send(sel("removeObserver:"), p.observer)
	delete(panelsByObserver, p.observer)
	delete(panelsBySearch, p.search)
	delete(panelsByView, p.view)
	p.observer.Send(sel("release"))
	p.empty.Send(sel("release"))
	p.divider.Send(sel("release"))
	p.search.Send(sel("release"))
	p.view.Send(sel("release"))
}
