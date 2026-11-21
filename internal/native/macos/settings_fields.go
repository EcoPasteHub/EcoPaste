//go:build darwin

package macos

import (
	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

type settingsOption struct {
	Key   i18n.Key
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
func (v *SettingsView) addSettingsListTable(content objc.ID, card CGRect, rowHeight, columnWidth float64, columnID string, accessibility i18n.Key, target objc.ID) objc.ID {
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
	table.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(accessibility)))
	v.bindText(func() {
		table.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(accessibility)))
	})

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
func (v *SettingsView) addSettingsMoveButton(content objc.ID, frame CGRect, title, accessibility i18n.Key, target objc.ID, action string) objc.ID {
	button := objc.ID(class("NSButton")).Send(sel("buttonWithTitle:target:action:"),
		nsString(i18n.T(title)), target, sel(action))
	button.Send(sel("setFrame:"), frame)
	button.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(accessibility)))
	button.Send(sel("setEnabled:"), false)
	content.Send(sel("addSubview:"), button)
	settingsViews[button] = v
	v.bindText(func() {
		button.Send(sel("setTitle:"), nsString(i18n.T(title)))
		button.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(accessibility)))
	})
	return button
}

func (v *SettingsView) addSwitch(content objc.ID, y float64, title i18n.Key, path string, get func(SettingsValues) bool) {
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

func (v *SettingsView) addPopup(content objc.ID, y float64, title i18n.Key, path string, options []settingsOption, get func(SettingsValues) string) *settingsField {
	v.addLabel(content, y, title)
	values := make([]string, 0, len(options))
	titles := make([]string, 0, len(options))
	control := newSettingsPopup(CGRect{
		Origin: CGPoint{
			X: settingsContentX + settingsContentWidth - settingsRowInset - settingsPopupMinWidth,
			Y: y + (settingsRowHeight-settingsPopupHeight)/2,
		},
		Size: CGSize{Width: settingsPopupMinWidth, Height: settingsPopupHeight},
	})
	for _, option := range options {
		text := settingsOptionText(option)
		values = append(values, option.Value)
		titles = append(titles, text)
		control.Send(sel("addItemWithTitle:"), nsString(text))
	}
	control.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(title)))
	control.Send(sel("setTarget:"), settingsTarget)
	control.Send(sel("setAction:"), sel("popupChanged:"))
	field := &settingsField{
		path: path, kind: "popup", control: control, options: values, titles: titles,
		title: title, choices: options,
		get: func(s SettingsValues) any { return get(s) },
	}
	v.addField(content, control, field)
	v.bindText(func() {
		v.rebuildPopup(field)
	})
	v.layoutPopup(field)
	return field
}

// layoutPopup keeps the control right aligned while sizing the collapsed
// capsule from its selected title.
func (v *SettingsView) layoutPopup(field *settingsField) {
	if field == nil || field.control == 0 {
		return
	}
	width := settingsPopupWidth(settingsPopupTitle(field.control))
	frame := objc.Send[CGRect](field.control, sel("frame"))
	frame.Origin.X = settingsContentX + settingsContentWidth - settingsRowInset - width
	frame.Size.Width = width
	field.control.Send(sel("setFrame:"), frame)
}

// rebuildPopup retranslates a popup's static choices while keeping the
// selected value. Groups are re-added by applyGroups afterwards.
func (v *SettingsView) rebuildPopup(field *settingsField) {
	if field == nil || field.control == 0 {
		return
	}
	selected := ""
	if index := int(objc.Send[int64](field.control, sel("indexOfSelectedItem"))); index >= 0 && index < len(field.options) {
		selected = field.options[index]
	}
	field.control.Send(sel("removeAllItems"))
	field.options = field.options[:0]
	field.titles = field.titles[:0]
	for _, choice := range field.choices {
		text := settingsOptionText(choice)
		field.options = append(field.options, choice.Value)
		field.titles = append(field.titles, text)
		field.control.Send(sel("addItemWithTitle:"), nsString(text))
	}
	field.control.Send(sel("setAccessibilityLabel:"), nsString(i18n.T(field.title)))
	selectPopup(field, selected)
	v.layoutPopup(field)
}

func (v *SettingsView) addLabel(content objc.ID, y float64, title i18n.Key) {
	v.addLabelWidth(content, y, title, settingsContentWidth-2*settingsRowInset-260)
}

func (v *SettingsView) addLabelWidth(content objc.ID, y float64, title i18n.Key, width float64) {
	v.addLabelText(content, y, width, func() string { return i18n.T(title) })
}

func (v *SettingsView) addLabelText(content objc.ID, y, width float64, text func() string) objc.ID {
	label := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(text()))
	label.Send(sel("setFont:"), systemFont(13))
	label.Send(sel("setLineBreakMode:"), int64(4))
	label.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: settingsContentX + settingsRowInset, Y: y + (settingsRowHeight-18)/2},
		Size:   CGSize{Width: width, Height: 18},
	})
	content.Send(sel("addSubview:"), label)
	v.bindText(func() {
		label.Send(sel("setStringValue:"), nsString(text()))
	})
	return label
}

func (v *SettingsView) addField(content, control objc.ID, field *settingsField) {
	field.tab = v.currentTab
	tag := int64(len(v.fields) + 1)
	control.Send(sel("setTag:"), tag)
	v.fields = append(v.fields, field)
	content.Send(sel("addSubview:"), control)
	settingsViews[control] = v
	control.Send(sel("release"))
}

func (v *SettingsView) fieldFor(sender objc.ID) *settingsField {
	tag := int(objc.Send[int64](sender, sel("tag")))
	if tag < 1 || tag > len(v.fields) {
		return nil
	}
	return v.fields[tag-1]
}
