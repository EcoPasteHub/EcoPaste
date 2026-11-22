//go:build darwin

package macos

import (
	"strings"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

// NoteSave is the payload the note popover emits when the user saves.
type NoteSave struct {
	ID   string
	Text string
}

// The note editor is a popover shown beside the clipboard panel (left or
// right, whichever has room) so it never covers the history list.
const (
	notePopoverWidth   = 340.0
	notePopoverPadding = popoverContentPadding
	noteTitleHeight    = 17.0
	noteTitleGap       = 6.0
	noteHintHeight     = 32.0
	noteHintGap        = 8.0
	noteAreaHeight     = 132.0
	noteAreaGap        = 12.0
	noteButtonHeight   = 30.0
	noteButtonWidth    = 96.0
	noteButtonGap      = 12.0
)

func notePopoverHeight() float64 {
	return notePopoverPadding*2 + noteTitleHeight + noteTitleGap + noteHintHeight +
		noteHintGap + noteAreaHeight + noteAreaGap + noteButtonHeight
}

var (
	noteControllerClass objc.Class
	notePopoverTarget   objc.ID
	noteActivePanel     *Panel

	panelsByNoteController = map[objc.ID]*Panel{}
	panelsByNotePopover    = map[objc.ID]*Panel{}
)

func registerNotePopoverClasses() error {
	var err error
	noteControllerClass, err = objc.RegisterClass("EcoPasteNotePopoverController",
		class("NSViewController"), nil, nil, nil)
	if err != nil {
		return err
	}

	target, err := objc.RegisterClass("EcoPasteNotePopoverTarget", class("NSObject"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("noteSavePressed:"), Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) {
				if noteActivePanel != nil {
					noteActivePanel.saveNote()
				}
			}},
			{Cmd: sel("noteCancelPressed:"), Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) {
				if noteActivePanel != nil {
					noteActivePanel.dismissNotePopover(true)
				}
			}},
			{Cmd: sel("popoverDidClose:"), Fn: panelNotePopoverDidClose},
			// Focus after the popover has finished showing. This is the point at
			// which the popover window is key; focusing in viewDidAppear happens
			// before that on the first open, leaving a first responder with no
			// editing session and no caret.
			{Cmd: sel("popoverDidShow:"), Fn: func(_ objc.ID, _ objc.SEL, note objc.ID) {
				p := panelsByNotePopover[objc.Send[objc.ID](note, sel("object"))]
				if p != nil {
					p.focusNoteArea()
				}
			}},
		})
	if err != nil {
		return err
	}
	notePopoverTarget = objc.ID(target).Send(sel("new"))
	return nil
}

// focusNoteArea focuses the note editor once per opening, only after its
// popover window is key so the caret really appears.
func (p *Panel) focusNoteArea() {
	if p.noteArea == nil || p.noteFocusDone {
		return
	}
	if p.noteArea.focusOnShow() {
		p.noteFocusDone = true
	}
}

func (p *Panel) buildNotePopover() {
	width := notePopoverWidth
	height := notePopoverHeight()

	view := newPopoverContentView(width, height)
	p.notePopoverView = view

	title := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(i18n.T(i18n.NoteEditTitle)))
	title.Send(sel("setFont:"), objc.ID(classNSFont).Send(sel("boldSystemFontOfSize:"), 13.0))
	title.Send(sel("setLineBreakMode:"), int64(4))
	title.Send(sel("setMaximumNumberOfLines:"), int64(1))
	view.Send(sel("addSubview:"), title)
	p.noteTitle = title

	hint := objc.ID(classNSTextField).Send(sel("wrappingLabelWithString:"), nsString(i18n.T(i18n.NoteEditBody)))
	hint.Send(sel("setFont:"), systemFont(11))
	hint.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	view.Send(sel("addSubview:"), hint)
	p.noteHint = hint

	area := newTextArea(noteAreaHeight)
	area.setAccessibilityLabel(i18n.T(i18n.NoteEditTitle))
	area.setCancelHandler(func() { p.dismissNotePopover(true) })
	view.Send(sel("addSubview:"), area.control)
	p.noteArea = area

	p.noteCancel = noteButton(i18n.T(i18n.CommonCancel), "noteCancelPressed:", false)
	p.noteSave = noteButton(i18n.T(i18n.CommonConfirm), "noteSavePressed:", true)
	view.Send(sel("addSubview:"), p.noteCancel)
	view.Send(sel("addSubview:"), p.noteSave)
	p.noteCancel.Send(sel("release"))
	p.noteSave.Send(sel("release"))

	layoutNotePopover(p, width, height)

	controller := objc.ID(noteControllerClass).Send(sel("new"))
	controller.Send(sel("setView:"), view)
	panelsByNoteController[controller] = p
	p.noteController = controller

	popover := objc.ID(class("NSPopover")).Send(sel("new"))
	popover.Send(sel("setBehavior:"), int64(1))
	popover.Send(sel("setAnimates:"), true)
	popover.Send(sel("setContentViewController:"), controller)
	setPopoverContentSize(popover, view, width, height)
	popover.Send(sel("setDelegate:"), notePopoverTarget)
	p.notePopover = popover
	panelsByNotePopover[popover] = p
}

func noteButton(title, action string, primary bool) objc.ID {
	button := objc.ID(class("NSButton")).Send(sel("alloc")).Send(sel("initWithFrame:"), CGRect{})
	button.Send(sel("setTitle:"), nsString(title))
	button.Send(sel("setBezelStyle:"), int64(1))
	button.Send(sel("setControlSize:"), int64(0))
	button.Send(sel("setFont:"), systemFont(13))
	button.Send(sel("setFocusRingType:"), int64(1))
	if primary {
		button.Send(sel("setBezelColor:"), objc.ID(classNSColor).Send(sel("controlAccentColor")))
	}
	button.Send(sel("setTarget:"), notePopoverTarget)
	button.Send(sel("setAction:"), sel(action))
	return button
}

func layoutNotePopover(p *Panel, width, height float64) {
	inner := width - 2*notePopoverPadding
	buttonY := notePopoverPadding
	areaY := buttonY + noteButtonHeight + noteAreaGap
	hintY := areaY + noteAreaHeight + noteHintGap
	titleY := hintY + noteHintHeight + noteTitleGap

	setFrame(p.noteTitle, notePopoverPadding, titleY, inner, noteTitleHeight)
	setFrame(p.noteHint, notePopoverPadding, hintY, inner, noteHintHeight)
	if p.noteArea != nil {
		p.noteArea.setFrame(notePopoverPadding, areaY, inner, noteAreaHeight)
	}
	saveX := width - notePopoverPadding - noteButtonWidth
	cancelX := saveX - noteButtonGap - noteButtonWidth
	setFrame(p.noteCancel, cancelX, buttonY, noteButtonWidth, noteButtonHeight)
	setFrame(p.noteSave, saveX, buttonY, noteButtonWidth, noteButtonHeight)
}

func (p *Panel) refreshNotePopoverText() {
	if p.noteTitle == 0 || p.noteArea == nil {
		return
	}
	p.noteTitle.Send(sel("setStringValue:"), nsString(i18n.T(i18n.NoteEditTitle)))
	p.noteHint.Send(sel("setStringValue:"), nsString(i18n.T(i18n.NoteEditBody)))
	p.noteCancel.Send(sel("setTitle:"), nsString(i18n.T(i18n.CommonCancel)))
	p.noteSave.Send(sel("setTitle:"), nsString(i18n.T(i18n.CommonConfirm)))
	p.noteArea.setAccessibilityLabel(i18n.T(i18n.NoteEditTitle))
}

func (p *Panel) notePopoverActive() bool {
	return p.notePopover != 0 && objc.Send[bool](p.notePopover, sel("isShown"))
}

// showNotePopover edits the note of rowID. When rowID is empty it falls back
// to the current selection.
func (p *Panel) showNotePopover(rowID string) {
	if p.notePopover == 0 || p.noteArea == nil {
		return
	}
	if p.notePopoverActive() {
		p.dismissNotePopover(true)
		return
	}
	if rowID == "" {
		rowID = p.selection
	}
	if rowID == "" {
		return
	}
	note := ""
	for _, row := range p.rows {
		if row.ID == rowID {
			note = row.Note
			break
		}
	}
	noteActivePanel = p
	p.noteTargetID = rowID
	p.noteFocusDone = false
	p.noteArea.resetFocus()
	p.noteArea.setString(note)
	p.refreshNotePopoverText()
	p.noteClosing = false
	rect, edge := p.notePopoverAnchor(rowID)
	showPopover(p.notePopover, p.view, rect, edge)
}

// notePopoverAnchor points at the card's vertical center on the panel's left
// or right edge, whichever has more room, so the popover opens beside the
// panel with its arrow on the card.
func (p *Panel) notePopoverAnchor(rowID string) (CGRect, uint64) {
	frame := bounds(p.view)
	centerY := frame.Size.Height / 2
	if index := p.rowIndex(rowID); index >= 0 {
		rowRect := objc.Send[CGRect](p.table, sel("rectOfRow:"), int64(index))
		inView := objc.Send[CGRect](p.table, sel("convertRect:toView:"), rowRect, p.view)
		centerY = inView.Origin.Y + inView.Size.Height/2
	}
	centerY = min(max(centerY, 0), frame.Size.Height)

	rect := CGRect{Origin: CGPoint{X: frame.Size.Width - 1, Y: centerY - 1}, Size: CGSize{Width: 1, Height: 2}}
	edge := uint64(2) // NSMaxXEdge
	if window := p.view.Send(sel("window")); window != 0 {
		win := objc.Send[CGRect](window, sel("frame"))
		rightRoom, leftRoom := 0.0, 0.0
		if screen, ok := screenForRect(win, screensBounds()); ok {
			rightRoom = screen.Frame.Origin.X + screen.Frame.Size.Width - (win.Origin.X + win.Size.Width)
			leftRoom = win.Origin.X - screen.Frame.Origin.X
		}
		if rightRoom < notePopoverWidth+panelScreenMargin && rightRoom < leftRoom {
			rect.Origin.X = 0
			edge = 0 // NSMinXEdge
		}
	}
	return rect, edge
}

func (p *Panel) rowIndex(id string) int {
	for index, row := range p.rows {
		if row.ID == id {
			return index
		}
	}
	return -1
}

func (p *Panel) saveNote() {
	if p.noteArea == nil {
		return
	}
	id := p.noteTargetID
	text := strings.TrimSpace(p.noteArea.stringValue())
	p.dismissNotePopover(true)
	if id == "" {
		return
	}
	p.cx.Emit("saveNote", NoteSave{ID: id, Text: text})
}

func (p *Panel) dismissNotePopover(programmatic bool) {
	if !p.notePopoverActive() {
		return
	}
	p.noteClosing = programmatic
	p.notePopover.Send(sel("close"))
}

func panelNotePopoverDidClose(_ objc.ID, _ objc.SEL, note objc.ID) {
	popover := objc.Send[objc.ID](note, sel("object"))
	p := panelsByNotePopover[popover]
	if p == nil {
		return
	}
	programmatic := p.noteClosing
	p.noteClosing = false
	if noteActivePanel == p {
		noteActivePanel = nil
	}
	window := p.view.Send(sel("window"))
	if programmatic {
		if window != 0 && objc.Send[bool](window, sel("isVisible")) {
			window.Send(sel("makeKeyWindow"))
			window.Send(sel("makeFirstResponder:"), p.search)
		}
		return
	}
	if window != 0 && !objc.Send[bool](window, sel("isKeyWindow")) &&
		!p.pinned && objc.Send[bool](window, sel("isVisible")) {
		p.cx.Emit("blurHide", nil)
	}
}
