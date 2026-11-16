//go:build darwin

package macos

import (
	"github.com/ebitengine/purego/objc"
)

type settingsOption struct {
	Title string
	Value string
}

func sizeSettingsTable(table objc.ID, width, rowHeight float64, rows int) {
	if table == 0 {
		return
	}
	height := float64(rows) * rowHeight
	if height < 1 {
		height = 1
	}
	table.Send(sel("setFrame:"), CGRect{Size: CGSize{Width: width, Height: height}})
	table.Send(sel("setAutoresizingMask:"), int64(2))
}

func addSettingsListCard(content objc.ID, frame CGRect) {
	card := objc.ID(classNSBox).Send(sel("new"))
	card.Send(sel("setFrame:"), frame)
	card.Send(sel("setBoxType:"), int64(4))
	card.Send(sel("setTitlePosition:"), int64(0))
	card.Send(sel("setBorderWidth:"), 0.0)
	card.Send(sel("setFillColor:"), settingsCardFillColor())
	card.Send(sel("setCornerRadius:"), settingsCardRadius)
	content.Send(sel("addSubview:"), card)
	card.Send(sel("release"))
}

// addSettingsListTable places a borderless table inside a list card and
// returns it, so the order editors share one table configuration.
func addSettingsListTable(content objc.ID, card CGRect, rowHeight, columnWidth float64, columnID, accessibility string, target objc.ID) objc.ID {
	addSettingsListCard(content, card)
	scroll := objc.ID(class("NSScrollView")).Send(sel("new"))
	scroll.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: card.Origin.X + 1, Y: card.Origin.Y + 1},
		Size:   CGSize{Width: card.Size.Width - 2, Height: card.Size.Height - 2},
	})
	scroll.Send(sel("setHasVerticalScroller:"), true)
	scroll.Send(sel("setAutohidesScrollers:"), true)
	scroll.Send(sel("setDrawsBackground:"), false)
	scroll.Send(sel("setBorderType:"), int64(0))

	table := objc.ID(class("NSTableView")).Send(sel("new"))
	table.Send(sel("setRowHeight:"), rowHeight)
	table.Send(sel("setIntercellSpacing:"), CGSize{})
	table.Send(sel("setHeaderView:"), objc.ID(0))
	table.Send(sel("setGridStyleMask:"), int64(0))
	table.Send(sel("setAllowsMultipleSelection:"), false)
	table.Send(sel("setAllowsEmptySelection:"), false)
	table.Send(sel("setSelectionHighlightStyle:"), int64(1))
	table.Send(sel("setFocusRingType:"), int64(1))
	table.Send(sel("setStyle:"), int64(3))
	table.Send(sel("setBackgroundColor:"), objc.ID(classNSColor).Send(sel("clearColor")))
	table.Send(sel("setDataSource:"), target)
	table.Send(sel("setDelegate:"), target)
	table.Send(sel("setAccessibilityLabel:"), nsString(accessibility))

	column := objc.ID(class("NSTableColumn")).Send(sel("new"))
	column.Send(sel("setIdentifier:"), nsString(columnID))
	column.Send(sel("setResizingMask:"), int64(1))
	column.Send(sel("setWidth:"), columnWidth)
	table.Send(sel("addTableColumn:"), column)
	column.Send(sel("release"))
	table.Send(sel("setColumnAutoresizingStyle:"), int64(1))

	scroll.Send(sel("setDocumentView:"), table)
	table.Send(sel("release"))
	content.Send(sel("addSubview:"), scroll)
	scroll.Send(sel("release"))
	return table
}

func newSettingsCheckbox(target objc.ID, action string) objc.ID {
	checkbox := objc.ID(class("NSButton")).Send(sel("checkboxWithTitle:target:action:"),
		nsString(""), target, sel(action))
	checkbox.Send(sel("retain"))
	checkbox.Send(sel("setFocusRingType:"), int64(1))
	return checkbox
}

// addSettingsMoveButton adds one reorder button, disabled until a row is
// selected, and registers the view it belongs to.
func (v *SettingsView) addSettingsMoveButton(content objc.ID, frame CGRect, title, accessibility string, target objc.ID, action string) objc.ID {
	button := objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(title), target, sel(action))
	button.Send(sel("setFrame:"), frame)
	button.Send(sel("setAccessibilityLabel:"), nsString(accessibility))
	button.Send(sel("setEnabled:"), false)
	content.Send(sel("addSubview:"), button)
	settingsViews[button] = v
	return button
}

func (v *SettingsView) addSwitch(content objc.ID, y float64, title, path string, get func(SettingsValues) bool) {
	v.addLabelWidth(content, y, title, settingsContentWidth-2*settingsRowInset-100)
	control := objc.ID(class("NSSwitch")).Send(sel("new"))
	control.Send(sel("setControlSize:"), int64(2))
	control.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{
			X: settingsContentX + settingsContentWidth - settingsRowInset - settingsSwitchWidth,
			Y: y + (settingsRowHeight-settingsSwitchHeight)/2,
		},
		Size: CGSize{Width: settingsSwitchWidth, Height: settingsSwitchHeight},
	})
	control.Send(sel("setTarget:"), settingsTarget)
	control.Send(sel("setAction:"), sel("toggleChanged:"))
	v.addField(content, control, &settingsField{
		path: path, kind: "bool", control: control,
		get: func(s SettingsValues) any { return get(s) },
	})
}

func (v *SettingsView) addPopup(content objc.ID, y float64, title, path string, options []settingsOption, get func(SettingsValues) string) *settingsField {
	v.addLabel(content, y, title)
	values := make([]string, 0, len(options))
	titles := make([]string, 0, len(options))
	const popupWidth = 220.0
	control := objc.ID(class("NSPopUpButton")).Send(sel("alloc")).Send(sel("initWithFrame:pullsDown:"),
		CGRect{
			Origin: CGPoint{
				X: settingsContentX + settingsContentWidth - settingsRowInset - popupWidth,
				Y: y + (settingsRowHeight-26)/2,
			},
			Size: CGSize{Width: popupWidth, Height: 26},
		}, false)
	for _, option := range options {
		values = append(values, option.Value)
		titles = append(titles, option.Title)
		control.Send(sel("addItemWithTitle:"), nsString(option.Title))
	}
	control.Send(sel("setTarget:"), settingsTarget)
	control.Send(sel("setAction:"), sel("popupChanged:"))
	field := &settingsField{
		path: path, kind: "popup", control: control, options: values, titles: titles,
		get: func(s SettingsValues) any { return get(s) },
	}
	v.addField(content, control, field)
	return field
}

func (v *SettingsView) addStepper(content objc.ID, y float64, title, path string, step, maxValue float64, suffix string, get func(SettingsValues) uint32) {
	v.addStepperRange(content, y, title, path, 0, maxValue, step, suffix, get)
}

func (v *SettingsView) addStepperRange(content objc.ID, y float64, title, path string, minValue, maxValue, step float64, suffix string, get func(SettingsValues) uint32) {
	v.addLabel(content, y, title)
	valueLabel := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
	valueLabel.Send(sel("setFont:"), systemFont(12))
	valueLabel.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	valueLabel.Send(sel("setAlignment:"), int64(2))
	valueLabel.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{
			X: settingsContentX + settingsContentWidth - settingsRowInset - settingsSwitchWidth - 110,
			Y: y + (settingsRowHeight-18)/2,
		},
		Size: CGSize{Width: 100, Height: 18},
	})
	content.Send(sel("addSubview:"), valueLabel)

	control := objc.ID(class("NSStepper")).Send(sel("new"))
	control.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{
			X: settingsContentX + settingsContentWidth - settingsRowInset - 19,
			Y: y + (settingsRowHeight-24)/2,
		},
		Size: CGSize{Width: 19, Height: 24},
	})
	control.Send(sel("setMinValue:"), minValue)
	control.Send(sel("setMaxValue:"), maxValue)
	control.Send(sel("setIncrement:"), step)
	control.Send(sel("setValueWraps:"), false)
	control.Send(sel("setTarget:"), settingsTarget)
	control.Send(sel("setAction:"), sel("stepperChanged:"))
	v.addField(content, control, &settingsField{
		path: path, kind: "stepper", control: control, valueLabel: valueLabel,
		suffix: suffix,
		get:    func(s SettingsValues) any { return get(s) },
	})
}

func (v *SettingsView) addLabel(content objc.ID, y float64, title string) {
	v.addLabelWidth(content, y, title, settingsContentWidth-2*settingsRowInset-260)
}

func (v *SettingsView) addLabelWidth(content objc.ID, y float64, title string, width float64) {
	label := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(title))
	label.Send(sel("setFont:"), systemFont(13))
	label.Send(sel("setLineBreakMode:"), int64(4))
	label.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsContentX + settingsRowInset, Y: y + (settingsRowHeight-18)/2},
		Size:   CGSize{Width: width, Height: 18},
	})
	content.Send(sel("addSubview:"), label)
}

func (v *SettingsView) addHint(content objc.ID, y, height float64, text string) {
	hint := objc.ID(classNSTextField).Send(sel("wrappingLabelWithString:"), nsString(text))
	hint.Send(sel("setFont:"), systemFont(11))
	hint.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("tertiaryLabelColor")))
	hint.Send(sel("setPreferredMaxLayoutWidth:"), settingsContentWidth-2*settingsRowInset)
	hint.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsContentX + settingsRowInset, Y: y},
		Size:   CGSize{Width: settingsContentWidth - 2*settingsRowInset, Height: height},
	})
	content.Send(sel("addSubview:"), hint)
}

// addParagraph lays out a wrapped settings description with explicit typography.
func (v *SettingsView) addParagraph(content objc.ID, frame CGRect, text string, size float64, color objc.ID, maxLines int64) {
	paragraph := objc.ID(classNSTextField).Send(sel("wrappingLabelWithString:"), nsString(text))
	paragraph.Send(sel("setFont:"), systemFont(size))
	paragraph.Send(sel("setTextColor:"), color)
	paragraph.Send(sel("setPreferredMaxLayoutWidth:"), frame.Size.Width)
	if maxLines > 0 {
		paragraph.Send(sel("setMaximumNumberOfLines:"), maxLines)
	}
	paragraph.Send(sel("setFrame:"), frame)
	content.Send(sel("addSubview:"), paragraph)
}

func (v *SettingsView) addField(content, control objc.ID, field *settingsField) {
	tag := int64(len(v.fields) + 1)
	control.Send(sel("setTag:"), tag)
	v.fields = append(v.fields, field)
	content.Send(sel("addSubview:"), control)
	settingsViews[control] = v
}

func (v *SettingsView) fieldFor(sender objc.ID) *settingsField {
	tag := int(objc.Send[int64](sender, sel("tag")))
	if tag < 1 || tag > len(v.fields) {
		return nil
	}
	return v.fields[tag-1]
}
