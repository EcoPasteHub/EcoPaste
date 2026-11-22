//go:build darwin

package macos

import (
	"strings"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

// The new-group prompt is a lightweight popover anchored to the top-right
// of the panel (right below the nav bar's + button). It replaces the old
// modal dialog: the text field is focused on show, Return submits and Esc
// or a click outside cancels. Its input is the shared textInput component,
// so it uses the same chrome as the settings inputs.
const (
	newGroupPopoverWidth   = 260.0
	newGroupPopoverPadding = popoverContentPadding
	newGroupTitleHeight    = 17.0
	newGroupTitleGap       = 8.0
	newGroupFieldHeight    = settingsControlHeight
)

func newGroupPopoverHeight() float64 {
	return newGroupPopoverPadding*2 + newGroupTitleHeight + newGroupTitleGap + newGroupFieldHeight
}

var (
	groupControllerClass objc.Class
	groupPopoverTarget   objc.ID

	panelsByGroupController = map[objc.ID]*Panel{}
	panelsByGroupPopover    = map[objc.ID]*Panel{}
)

func registerGroupPopoverClasses() error {
	var err error
	groupControllerClass, err = objc.RegisterClass("EcoPasteGroupPopoverController",
		class("NSViewController"), nil, nil, []objc.MethodDef{
			// Focus the field once the popover window is on screen; doing
			// it before the show animation finishes lets AppKit reset the
			// first responder back to the panel.
			{Cmd: sel("viewDidAppear"), Fn: func(self objc.ID, _ objc.SEL) {
				if p := panelsByGroupController[self]; p != nil {
					p.focusNewGroupField()
				}
			}},
		})
	if err != nil {
		return err
	}

	target, err := objc.RegisterClass("EcoPasteGroupPopoverTarget", class("NSObject"), nil, nil,
		[]objc.MethodDef{
			{Cmd: sel("popoverDidClose:"), Fn: panelGroupPopoverDidClose},
		})
	if err != nil {
		return err
	}
	groupPopoverTarget = objc.ID(target).Send(sel("new"))
	return nil
}

func (p *Panel) buildNewGroupPopover() {
	width := newGroupPopoverWidth
	height := newGroupPopoverHeight()

	view := newPopoverContentView(width, height)
	p.groupPopoverView = view

	title := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(i18n.T(i18n.GroupNewTitle)))
	title.Send(sel("setFont:"), objc.ID(classNSFont).Send(sel("boldSystemFontOfSize:"), 13.0))
	title.Send(sel("setLineBreakMode:"), int64(4))
	title.Send(sel("setMaximumNumberOfLines:"), int64(1))
	view.Send(sel("addSubview:"), title)
	p.groupTitle = title

	input := newTextInput(newGroupFieldHeight)
	input.setPlaceholder(i18n.T(i18n.GroupNamePlaceholder))
	input.setAccessibilityLabel(i18n.T(i18n.GroupNameLabel))
	input.setReturnHandler(func() { p.submitNewGroup() })
	input.setCancelHandler(func() { p.dismissNewGroupPopover(true) })
	view.Send(sel("addSubview:"), input.control)
	p.groupInput = input

	layoutNewGroupPopover(p, width, height)

	controller := objc.ID(groupControllerClass).Send(sel("new"))
	controller.Send(sel("setView:"), view)
	panelsByGroupController[controller] = p
	p.groupController = controller

	popover := objc.ID(class("NSPopover")).Send(sel("new"))
	popover.Send(sel("setBehavior:"), int64(1))
	popover.Send(sel("setAnimates:"), true)
	popover.Send(sel("setContentViewController:"), controller)
	setPopoverContentSize(popover, view, width, height)
	popover.Send(sel("setDelegate:"), groupPopoverTarget)
	p.groupPopover = popover
	panelsByGroupPopover[popover] = p
}

func layoutNewGroupPopover(p *Panel, width, height float64) {
	inner := width - 2*newGroupPopoverPadding
	setFrame(p.groupTitle, newGroupPopoverPadding, height-newGroupPopoverPadding-newGroupTitleHeight,
		inner, newGroupTitleHeight)
	if p.groupInput != nil {
		p.groupInput.setFrame(newGroupPopoverPadding, newGroupPopoverPadding, inner, newGroupFieldHeight)
	}
}

func (p *Panel) refreshNewGroupPopoverText() {
	if p.groupTitle == 0 || p.groupInput == nil {
		return
	}
	p.groupTitle.Send(sel("setStringValue:"), nsString(i18n.T(i18n.GroupNewTitle)))
	p.groupInput.setPlaceholder(i18n.T(i18n.GroupNamePlaceholder))
	p.groupInput.setAccessibilityLabel(i18n.T(i18n.GroupNameLabel))
}

// groupPopoverActive reports whether the new-group popover is on screen, so
// the panel does not hide when the popover's window takes key focus.
func (p *Panel) groupPopoverActive() bool {
	return p.groupPopover != 0 && objc.Send[bool](p.groupPopover, sel("isShown"))
}

func (p *Panel) focusNewGroupField() {
	if p.groupInput != nil {
		p.groupInput.focus()
	}
}

// showNewGroupPopover opens the inline prompt centered on the bottom edge
// of the trigger button, so it drops straight down from the button. The
// field focus is forced in viewDidAppear, so anchoring to the nav child
// window still receives typing.
func (p *Panel) showNewGroupPopover(anchor objc.ID) {
	if p.groupPopover == 0 || p.groupInput == nil {
		return
	}
	if p.groupPopoverActive() {
		p.dismissNewGroupPopover(true)
		return
	}
	if anchor == 0 {
		anchor = p.filterButton
	}
	if anchor == 0 {
		return
	}
	p.groupInput.setStringValue("")
	p.refreshNewGroupPopoverText()
	p.groupClosing = false
	p.groupSubmitted = false
	showPopover(p.groupPopover, anchor, bounds(anchor), uint64(1))
}

// newGroupAnchor is the button the prompt should drop down from: the nav
// bar's + button, or the toolbar filter button when the nav is unavailable.
func (p *Panel) newGroupAnchor() objc.ID {
	if p.nav != nil && p.nav.plus != 0 {
		return p.nav.plus
	}
	return p.filterButton
}

func (p *Panel) submitNewGroup() {
	if p.groupInput == nil || p.groupSubmitted {
		return
	}
	name := strings.TrimSpace(p.groupInput.stringValue())
	if name == "" {
		return
	}
	p.groupSubmitted = true
	p.dismissNewGroupPopover(true)
	p.cx.Emit("createGroup", name)
}

// dismissNewGroupPopover closes the prompt. programmatic marks closes we
// start ourselves (submit, cancel, toggling) so the close delegate keeps
// the panel instead of treating it as a click outside.
func (p *Panel) dismissNewGroupPopover(programmatic bool) {
	if !p.groupPopoverActive() {
		return
	}
	p.groupClosing = programmatic
	p.groupPopover.Send(sel("close"))
}

func panelGroupPopoverDidClose(_ objc.ID, _ objc.SEL, note objc.ID) {
	popover := objc.Send[objc.ID](note, sel("object"))
	p := panelsByGroupPopover[popover]
	if p == nil {
		return
	}
	programmatic := p.groupClosing
	p.groupClosing = false
	window := p.view.Send(sel("window"))
	if programmatic {
		// Return key focus to the panel so list navigation keeps working
		// after a submit or cancel.
		if window != 0 && objc.Send[bool](window, sel("isVisible")) {
			window.Send(sel("makeKeyWindow"))
			responder := p.table
			if p.behavior.DefaultFocus {
				responder = p.search
			}
			window.Send(sel("makeFirstResponder:"), responder)
		}
		return
	}
	// Clicking outside a transient popover dismisses it; mirror the preview
	// popover by hiding the panel unless it is pinned.
	if window != 0 && !objc.Send[bool](window, sel("isKeyWindow")) &&
		!p.pinned && objc.Send[bool](window, sel("isVisible")) {
		p.cx.Emit("blurHide", nil)
	}
}
