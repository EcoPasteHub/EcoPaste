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
	"language":    setPanelLanguage,
}

var panelCommands = map[string]func(*Panel){
	"focusSearch":            (*Panel).focusSearch,
	"clearSearch":            (*Panel).clearSearch,
	"reload":                 reloadPanelTable,
	"scrollTop":              scrollPanelToTop,
	"selectFirstOnOpen":      selectFirstOnOpen,
	"applyPendingSelection":  applyPendingSelection,
	"showAccessibilityAlert": showPanelAccessibilityAlert,
	"toggleWindowPinned":     (*Panel).togglePinned,
	"resignKey":              resignPanelKey,
	"makeKey":                makePanelKey,
}

func setPanelQuery(p *Panel, value any) {
	s, ok := value.(string)
	if !ok || s == p.query {
		return
	}
	p.query = s
	p.markSelectionFirst()
	p.search.Send(sel("setStringValue:"), nsString(s))
	// Re-render the visible rows so their keyword highlights follow the
	// query even when the row props arrive in the same batch. Stop any hover
	// fade first: reloading the cells can release the view it paints.
	if p.table != 0 {
		clearRowHoverStates()
		p.table.Send(sel("reloadData"))
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
	// hideHover starts a fade-out on the cell under the pointer. The list is
	// about to reload and may release that cell, so drop the pending fade
	// before reloadData rather than animating a view that is going away.
	clearRowHoverStates()
	previousID := p.selection
	previousIndex := p.selectedIndex()
	intent := p.selectionIntent
	p.selectionIntent = selectionKeep
	p.rows = rows
	p.recomputeRowHeights()
	p.table.Send(sel("reloadData"))
	p.syncTableFrame()
	p.syncSelection(previousID, previousIndex, intent)
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
	id, ok := value.(string)
	if !ok {
		return
	}
	if id != p.groupID {
		// Switching groups always starts at the top, even when the previous
		// row also belongs to the new group.
		p.markSelectionFirst()
	}
	p.setGroupSelection(id)
}

func setPanelFilter(p *Panel, value any) {
	filter, ok := value.(string)
	if !ok {
		return
	}
	if filter != p.filterValue {
		// Switching the visible category also starts at the top.
		p.markSelectionFirst()
	}
	p.setFilterSelection(filter)
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
		p.recomputeRowHeights()
		clearRowHoverStates()
		p.table.Send(sel("reloadData"))
		p.syncTableFrame()
		p.ensureSelection()
	}
}

func setPanelLanguage(p *Panel, value any) {
	if language, ok := value.(string); ok {
		p.applyLanguage(language)
	}
}

func setPanelActions(p *Panel, value any) {
	if actions, ok := value.([]string); ok {
		p.actions = actions
		p.hideHover()
	}
}

func reloadPanelTable(p *Panel) {
	clearRowHoverStates()
	p.table.Send(sel("reloadData"))
	p.ensureSelection()
}

func showPanelAccessibilityAlert(_ *Panel) {
	showAccessibilityAlert()
}

func scrollPanelToTop(p *Panel) {
	if len(p.rows) > 0 {
		p.table.Send(sel("scrollRowToVisible:"), int64(0))
	}
}

// selectFirstOnOpen marks the next list update to select the first row, so
// opening the panel back at the top lands on the first item instead of an
// unrelated row.
func selectFirstOnOpen(p *Panel) {
	p.markSelectionFirst()
}

// applyPendingSelection consumes a selection intent that no rows update
// applied, which happens when a group, category or search switch returns
// exactly the same rows as before. The list still has to start at the first
// row.
func applyPendingSelection(p *Panel) {
	if p.selectionIntent == selectionKeep {
		return
	}
	previousID := p.selection
	previousIndex := p.selectedIndex()
	intent := p.selectionIntent
	p.selectionIntent = selectionKeep
	p.syncSelection(previousID, previousIndex, intent)
}

func resignPanelKey(p *Panel) {
	p.view.Send(sel("window")).Send(sel("resignKeyWindow"))
}

func makePanelKey(p *Panel) {
	p.view.Send(sel("window")).Send(sel("makeKeyWindow"))
}
