//go:build darwin

package macos

var panelPropertySetters = map[string]func(*Panel, any){
	"query":       setPanelQuery,
	"placeholder": setPanelPlaceholder,
	"rows":        setPanelRows,
	"total":       setPanelTotal,
	"loading":     setPanelLoading,
	"error":       setPanelError,
	"preview":     setPanelPreview,
	"groups":      setPanelGroups,
	"group":       setPanelGroup,
	"filter":      setPanelFilter,
	"behavior":    setPanelBehavior,
	"drag":        setPanelDrag,
	"display":     setPanelDisplay,
	"actions":     setPanelActions,
}

var panelCommands = map[string]func(*Panel){
	"focusSearch":            (*Panel).focusSearch,
	"clearSearch":            (*Panel).clearSearch,
	"reload":                 reloadPanelTable,
	"scrollTop":              scrollPanelToTop,
	"showAccessibilityAlert": showPanelAccessibilityAlert,
	"toggleWindowPinned":     (*Panel).togglePinned,
	"resignKey":              resignPanelKey,
	"makeKey":                makePanelKey,
}

func setPanelQuery(p *Panel, value any) {
	if s, ok := value.(string); ok && s != p.query {
		p.query = s
		p.search.Send(sel("setStringValue:"), nsString(s))
	}
}

func setPanelPlaceholder(p *Panel, value any) {
	if s, ok := value.(string); ok {
		p.search.Send(sel("setPlaceholderString:"), nsString(s))
	}
}

func setPanelRows(p *Panel, value any) {
	rows, _ := value.([]HistoryRow)
	p.hideHover()
	p.rows = rows
	p.table.Send(sel("reloadData"))
	p.syncTableFrame()
	p.restoreSelection()
	p.updateStatus()
}

func setPanelTotal(p *Panel, value any) {
	if n, ok := int64Value(value); ok {
		p.total = n
		p.updateStatus()
	}
}

func setPanelLoading(p *Panel, value any) {
	if b, ok := value.(bool); ok {
		p.loading = b
		p.updateStatus()
	}
}

func setPanelError(p *Panel, value any) {
	if s, ok := value.(string); ok {
		p.errText = s
		p.updateStatus()
	}
}

func setPanelPreview(p *Panel, value any) {
	if content, ok := value.(PreviewContent); ok {
		p.setPreview(content)
	}
}

func setPanelGroups(p *Panel, value any) {
	if groups, ok := value.([]GroupOption); ok {
		p.groups = groups
	}
}

func setPanelGroup(p *Panel, value any) {
	if id, ok := value.(string); ok {
		p.setGroupSelection(id)
	}
}

func setPanelFilter(p *Panel, value any) {
	if filter, ok := value.(string); ok {
		p.setFilterSelection(filter)
	}
}

func setPanelBehavior(p *Panel, value any) {
	if behavior, ok := value.(PanelBehavior); ok {
		p.behavior = behavior
	}
}

func setPanelDrag(p *Panel, value any) {
	if provider, ok := value.(func(string) (DragPayload, bool)); ok {
		p.drag = provider
	}
}

func setPanelDisplay(p *Panel, value any) {
	if display, ok := value.(PanelDisplay); ok {
		p.display = display
		p.table.Send(sel("setRowHeight:"), tableRowHeight(display))
		p.table.Send(sel("reloadData"))
		p.syncTableFrame()
	}
}

func setPanelActions(p *Panel, value any) {
	if actions, ok := value.([]string); ok {
		p.actions = actions
		p.hideHover()
	}
}

func reloadPanelTable(p *Panel) {
	p.table.Send(sel("reloadData"))
}

func showPanelAccessibilityAlert(_ *Panel) {
	showAccessibilityAlert()
}

func scrollPanelToTop(p *Panel) {
	if len(p.rows) > 0 {
		p.table.Send(sel("scrollRowToVisible:"), int64(0))
	}
}

func resignPanelKey(p *Panel) {
	p.view.Send(sel("window")).Send(sel("resignKeyWindow"))
}

func makePanelKey(p *Panel) {
	p.view.Send(sel("window")).Send(sel("makeKeyWindow"))
}
