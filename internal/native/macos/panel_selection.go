//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

// selectionIntent tells the next list update how to place the selection.
type selectionIntent uint8

const (
	// selectionKeep preserves the current row, or the row that took its
	// position when it is gone (deleting a row selects the next one).
	selectionKeep selectionIntent = iota
	// selectionFirst always selects the top row (switching groups, categories
	// or searches, and returning to the top on open).
	selectionFirst
)

// selectionIndex chooses the row to select once the list is replaced. rows is
// the new list, currentID the id selected before the change and previousIndex
// its old position (-1 when nothing was selected). The list always keeps one
// row selected when it is not empty.
func selectionIndex(rows []HistoryRow, currentID string, previousIndex int, intent selectionIntent) int {
	if len(rows) == 0 {
		return -1
	}
	if intent == selectionFirst {
		return 0
	}
	for index, row := range rows {
		if row.ID == currentID {
			return index
		}
	}
	if previousIndex < 0 {
		return 0
	}
	if previousIndex >= len(rows) {
		return len(rows) - 1
	}
	return previousIndex
}

// markSelectionFirst asks the next list update to select the top row.
func (p *Panel) markSelectionFirst() {
	p.selectionIntent = selectionFirst
}

// syncSelection reconciles the table selection after the rows change so the
// list never ends up without a selected row. previousID and previousIndex are
// the selection captured before the rows were replaced.
func (p *Panel) syncSelection(previousID string, previousIndex int, intent selectionIntent) {
	index := selectionIndex(p.rows, previousID, previousIndex, intent)
	if index < 0 {
		p.table.Send(sel("deselectAll:"), objc.ID(0))
		if p.selection != "" {
			p.selection = ""
			p.cx.Emit("select", "")
		}
		return
	}
	// A forced-first placement happens when the list is replaced by a new
	// category, group or search. Reveal the top row so the viewport starts at
	// the beginning instead of keeping the old scroll offset over the new
	// rows, which otherwise looks like a stuck or shifted list.
	p.selectRowIndex(index, intent == selectionFirst)
	if p.selection != previousID {
		p.cx.Emit("select", p.selection)
	}
}

// ensureSelection re-applies a valid selection after the table reloads its
// rows or row heights without the row set changing.
func (p *Panel) ensureSelection() {
	p.syncSelection(p.selection, p.selectedIndex(), selectionKeep)
}

// selectRowIndex highlights one row and records it as the selection. scroll
// reveals it when asked.
func (p *Panel) selectRowIndex(index int, scroll bool) {
	if index < 0 || index >= len(p.rows) {
		return
	}
	p.selection = p.rows[index].ID
	indexes := objc.ID(class("NSIndexSet")).Send(sel("indexSetWithIndex:"), int64(index))
	p.table.Send(sel("selectRowIndexes:byExtendingSelection:"), indexes, false)
	if scroll {
		p.table.Send(sel("scrollRowToVisible:"), int64(index))
	}
}

func (p *Panel) selectRow(index int) {
	// An explicit selection (click, keyboard navigation) wins over a pending
	// automatic placement such as the return-to-top reset from opening.
	p.selectionIntent = selectionKeep
	p.selectRowIndex(index, true)
}

// selectRowClamped keeps a computed target inside the list instead of
// silently doing nothing at the edges, so Page Up/Down and single-step
// movement stop on the first or last row.
func (p *Panel) selectRowClamped(index int) {
	if len(p.rows) == 0 {
		return
	}
	if index < 0 {
		index = 0
	}
	if index >= len(p.rows) {
		index = len(p.rows) - 1
	}
	p.selectRow(index)
}

// nextRowIndex returns the row reached by moving delta rows from current,
// clamped to the list bounds. With nothing selected it activates the first row.
func nextRowIndex(current, delta, count int) int {
	if count <= 0 {
		return -1
	}
	if current < 0 {
		return 0
	}
	next := current + delta
	if next < 0 {
		return 0
	}
	if next >= count {
		return count - 1
	}
	return next
}

// moveSelection moves one row, clamping to the list bounds.
func (p *Panel) moveSelection(delta int) {
	if len(p.rows) == 0 {
		return
	}
	p.selectRow(nextRowIndex(p.selectedIndex(), delta, len(p.rows)))
}

// applyNavigation performs a navigation action shared by the table key
// handler and the preview event monitor.
func (p *Panel) applyNavigation(action string) {
	switch action {
	case "firstRow":
		p.selectRowClamped(0)
	case "lastRow":
		p.selectRowClamped(len(p.rows) - 1)
	case "pageUp", "pageDown":
		current := p.selectedIndex()
		if current < 0 {
			current = 0
		}
		delta := p.pageSize()
		if action == "pageUp" {
			delta = -delta
		}
		p.selectRowClamped(current + delta)
	case "previousRow":
		p.moveSelection(-1)
	case "nextRow":
		p.moveSelection(1)
	}
}

func (p *Panel) selectedIndex() int {
	for index, row := range p.rows {
		if row.ID == p.selection {
			return index
		}
	}
	return -1
}

func (p *Panel) pageSize() int {
	visible := objc.Send[CGRect](p.table, sel("visibleRect")).Size.Height
	if visible <= 0 || len(p.rowHeights) == 0 {
		return 1
	}
	spacing := objc.Send[CGSize](p.table, sel("intercellSpacing")).Height
	total := 0.0
	size := 0
	for _, height := range p.rowHeights {
		if total+height > visible {
			break
		}
		total += height + spacing
		size++
	}
	if size < 1 {
		return 1
	}
	return size
}
