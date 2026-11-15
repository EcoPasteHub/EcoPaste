package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

func (p *Panel) setEmptyState(state emptyState) {
	visible := state.Title != ""
	p.empty.Send(sel("setStringValue:"), nsString(state.Title))
	p.empty.Send(sel("setHidden:"), !visible)
	p.emptyDetail.Send(sel("setStringValue:"), nsString(state.Detail))
	p.emptyDetail.Send(sel("setHidden:"), !visible || state.Detail == "")
	p.emptyIcon.Send(sel("setHidden:"), !visible)
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
	p.emptyIcon.Send(sel("setImage:"), systemSymbol(state.Symbol))
	p.emptyIcon.Send(sel("setContentTintColor:"), iconColor)
}

type emptyState struct {
	Symbol string
	Title  string
	Detail string
	Error  bool
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
			Symbol: "doc.on.clipboard", Title: i18n.T(i18n.PanelEmpty),
			Detail: i18n.T(i18n.PanelEmptyDetail),
		}
	}
}
