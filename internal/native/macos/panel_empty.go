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
	p.empty.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	p.emptyDetail.Send(sel("setStringValue:"), nsString(state.Detail))
	p.emptyDetail.Send(sel("setHidden:"), !visible || state.Detail == "")
	for kind, icon := range p.emptyIcons {
		icon.view.Send(sel("setHidden:"), !visible || state.Kind != kind)
	}
}

type emptyState struct {
	Kind   emptyStateKind
	Title  string
	Detail string
}

// emptyStateKind selects the Kubai illustration for a scenario.
type emptyStateKind int

const (
	emptyKindNone emptyStateKind = iota
	emptyKindHistory
	emptyKindSearch
	emptyKindText
	emptyKindImage
	emptyKindFiles
	emptyKindFavorites
)

// emptyStateFor resolves the placeholder shown when the panel has no rows.
// A search query wins so the copy is about the search, then an active custom
// group, then the selected category (all, text, image, files or favorites).
func emptyStateFor(rows int, query, filter, group string) emptyState {
	switch {
	case rows > 0:
		return emptyState{}
	case query != "":
		return emptyState{
			Kind:   emptyKindSearch,
			Title:  i18n.T(i18n.PanelNoMatches),
			Detail: i18n.T(i18n.PanelNoMatchesDetail),
		}
	case group != "":
		return emptyState{
			Kind:   emptyKindHistory,
			Title:  i18n.T(i18n.PanelEmptyGroup),
			Detail: i18n.T(i18n.PanelEmptyGroupDetail),
		}
	default:
		return categoryEmptyState(filter)
	}
}

func categoryEmptyState(filter string) emptyState {
	switch filter {
	case "text":
		return emptyState{
			Kind:   emptyKindText,
			Title:  i18n.T(i18n.PanelEmptyText),
			Detail: i18n.T(i18n.PanelEmptyTextDetail),
		}
	case "image":
		return emptyState{
			Kind:   emptyKindImage,
			Title:  i18n.T(i18n.PanelEmptyImage),
			Detail: i18n.T(i18n.PanelEmptyImageDetail),
		}
	case "files":
		return emptyState{
			Kind:   emptyKindFiles,
			Title:  i18n.T(i18n.PanelEmptyFiles),
			Detail: i18n.T(i18n.PanelEmptyFilesDetail),
		}
	case "favorite":
		return emptyState{
			Kind:   emptyKindFavorites,
			Title:  i18n.T(i18n.PanelEmptyFavorites),
			Detail: i18n.T(i18n.PanelEmptyFavoritesDetail),
		}
	default:
		return emptyState{
			Kind:   emptyKindHistory,
			Title:  i18n.T(i18n.PanelEmpty),
			Detail: i18n.T(i18n.PanelEmptyDetail),
		}
	}
}

func emptyIconSymbol(kind emptyStateKind) *kubai.Symbol {
	switch kind {
	case emptyKindSearch:
		return kubai.Search()
	case emptyKindText:
		return kubai.FileText()
	case emptyKindImage:
		return kubai.Image()
	case emptyKindFiles:
		return kubai.Folder()
	case emptyKindFavorites:
		return kubai.Star()
	default:
		return kubai.Clipboard()
	}
}

// emptyKubaiIconSize is the decorative illustration above the empty title.
const emptyKubaiIconSize = 44.0

type emptyIconView struct {
	view objc.ID
	nsgo *nsgo.View
}

// buildEmptyIcons embeds one Kubai illustration per scenario on first use,
// stacked in the same place and switched by visibility. The NSGo views follow
// the panel's effective appearance.
func (p *Panel) buildEmptyIcons() {
	if len(p.emptyIcons) > 0 {
		return
	}
	kinds := []emptyStateKind{
		emptyKindHistory, emptyKindSearch, emptyKindText,
		emptyKindImage, emptyKindFiles, emptyKindFavorites,
	}
	p.emptyIcons = make(map[emptyStateKind]emptyIconView, len(kinds))
	for _, kind := range kinds {
		symbol := emptyIconSymbol(kind)
		icon := nsgo.NewView(func(*core.Scope) core.Node {
			return core.View{
				Style: core.Style{
					Width:          core.Pt(emptyKubaiIconSize),
					Height:         core.Pt(emptyKubaiIconSize),
					AlignItems:     core.AlignCenter,
					JustifyContent: core.JustifyCenter,
				},
				Children: []core.Node{
					kubai.Icon{Symbol: symbol, Size: emptyKubaiIconSize},
				},
			}
		})
		view := objc.ID(icon.Handle())
		view.Send(sel("setHidden:"), true)
		p.view.Send(sel("addSubview:"), view)
		p.emptyIcons[kind] = emptyIconView{view: view, nsgo: icon}
	}
}

func (p *Panel) closeEmptyIcons() {
	for kind, icon := range p.emptyIcons {
		icon.nsgo.Close()
		delete(p.emptyIcons, kind)
	}
}
