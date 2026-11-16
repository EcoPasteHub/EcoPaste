package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
	"github.com/nsgo-dev/nsgo"
	kubai "github.com/nsgo-dev/nsgo-icons-kubai"
	"github.com/nsgo-dev/nsgo/core"
)

func (p *Panel) setEmptyState(state emptyState) {
	visible := state.Title != ""
	p.empty.Send(sel("setStringValue:"), nsString(state.Title))
	p.empty.Send(sel("setHidden:"), !visible)
	p.emptyDetail.Send(sel("setStringValue:"), nsString(state.Detail))
	p.emptyDetail.Send(sel("setHidden:"), !visible || state.Detail == "")
	useKubai := visible && state.Kubai
	p.emptyIcon.Send(sel("setHidden:"), !visible || useKubai)
	if p.emptyKubai != 0 {
		p.emptyKubai.Send(sel("setHidden:"), !useKubai)
	}
	if !visible {
		return
	}
	color := objc.ID(classNSColor).Send(sel("secondaryLabelColor"))
	iconColor := objc.ID(classNSColor).Send(sel("tertiaryLabelColor"))
	if state.Error {
		color = objc.ID(classNSColor).Send(sel("systemRedColor"))
		iconColor = color
	}
	p.empty.Send(sel("setTextColor:"), color)
	if useKubai {
		p.ensureEmptyKubai()
		return
	}
	p.emptyIcon.Send(sel("setImage:"), systemSymbol(state.Symbol))
	p.emptyIcon.Send(sel("setContentTintColor:"), iconColor)
}

type emptyState struct {
	Symbol string
	Title  string
	Detail string
	Error  bool
	Kubai  bool
}

func emptyStateFor(errText string, rows int, query string, loading bool) emptyState {
	switch {
	case rows > 0:
		return emptyState{}
	case errText != "":
		return emptyState{Symbol: "exclamationmark.triangle", Title: errText, Error: true}
	case loading:
		return emptyState{Symbol: "clock", Title: i18n.T(i18n.PanelLoading)}
	case query != "":
		return emptyState{
			Symbol: "magnifyingglass", Title: i18n.T(i18n.PanelNoMatches),
			Detail: i18n.T(i18n.PanelNoMatchesDetail),
		}
	default:
		return emptyState{
			Title: i18n.T(i18n.PanelEmpty), Kubai: true,
			Detail: i18n.T(i18n.PanelEmptyDetail),
		}
	}
}

// emptyKubaiIconSize is the decorative illustration above the empty title.
const emptyKubaiIconSize = 44.0

// ensureEmptyKubai embeds the Kubai empty illustration on first use. The
// NSGo view follows the panel's effective appearance.
func (p *Panel) ensureEmptyKubai() {
	if p.emptyKubai != 0 {
		return
	}
	icon := nsgo.NewView(func(*core.Scope) core.Node {
		return core.View{
			Style: core.Style{
				Width:          core.Pt(emptyKubaiIconSize),
				Height:         core.Pt(emptyKubaiIconSize),
				AlignItems:     core.AlignCenter,
				JustifyContent: core.JustifyCenter,
			},
			Children: []core.Node{
				kubai.Icon{Symbol: kubai.Clipboard(), Size: emptyKubaiIconSize},
			},
		}
	})
	p.emptyKubaiIcon = icon
	p.emptyKubai = objc.ID(icon.Handle())
	p.view.Send(sel("addSubview:"), p.emptyKubai)
}
