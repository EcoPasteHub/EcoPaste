//go:build darwin

package macos

import (
	"strings"

	"github.com/ebitengine/purego/objc"
)

type settingsOption struct {
	Title string
	Value string
}

func (v *SettingsView) newTab(identifier, title string) objc.ID {
	v.currentTab = identifier
	v.currentTabTitle = title
	item := objc.ID(class("NSTabViewItem")).Send(sel("alloc")).Send(sel("initWithIdentifier:"), nsString(identifier))
	item.Send(sel("setLabel:"), nsString(title))
	content := objc.ID(class("NSView")).Send(sel("new"))
	content.Send(sel("setFrame:"), CGRect{Size: CGSize{Width: tabContentW, Height: tabContentH}})
	item.Send(sel("setView:"), content)
	v.tabs.Send(sel("addTabViewItem:"), item)
	item.Send(sel("release"))
	content.Send(sel("release"))
	return content
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
	card.Send(sel("setBorderWidth:"), 1.0)
	card.Send(sel("setBorderColor:"), objc.ID(classNSColor).Send(sel("separatorColor")))
	card.Send(sel("setFillColor:"), objc.ID(classNSColor).Send(sel("controlBackgroundColor")))
	card.Send(sel("setCornerRadius:"), 8.0)
	content.Send(sel("addSubview:"), card)
	card.Send(sel("release"))
}

func newSettingsCheckbox(target objc.ID, action string) objc.ID {
	checkbox := objc.ID(class("NSButton")).Send(sel("checkboxWithTitle:target:action:"),
		nsString(""), target, sel(action))
	checkbox.Send(sel("retain"))
	checkbox.Send(sel("setFocusRingType:"), int64(1))
	return checkbox
}

func (v *SettingsView) addSwitch(content objc.ID, y float64, title, path string, get func(SettingsValues) bool) {
	v.addLabelWidth(content, y, title, tabContentW-104)
	v.addSearchEntry(content, title, path, "", "", rowHighlightFrame(y))
	control := objc.ID(class("NSSwitch")).Send(sel("new"))
	control.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: tabContentW - 64, Y: y - 2}, Size: CGSize{Width: 40, Height: 24}})
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
	control := objc.ID(class("NSPopUpButton")).Send(sel("alloc")).Send(sel("initWithFrame:pullsDown:"),
		CGRect{Origin: CGPoint{X: tabContentW - 260, Y: y - 4}, Size: CGSize{Width: 240, Height: 28}}, false)
	for _, option := range options {
		values = append(values, option.Value)
		titles = append(titles, option.Title)
		control.Send(sel("addItemWithTitle:"), nsString(option.Title))
	}
	v.addSearchEntry(content, title, path, "", strings.Join(titles, " "), rowHighlightFrame(y))
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
	v.addSearchEntry(content, title, path, suffix, "", rowHighlightFrame(y))
	valueLabel := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(""))
	valueLabel.Send(sel("setFont:"), systemFont(12))
	valueLabel.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("secondaryLabelColor")))
	valueLabel.Send(sel("setAlignment:"), int64(2))
	valueLabel.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: tabContentW - 210, Y: y}, Size: CGSize{Width: 120, Height: 18}})
	content.Send(sel("addSubview:"), valueLabel)

	control := objc.ID(class("NSStepper")).Send(sel("new"))
	control.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: tabContentW - 80, Y: y - 5}, Size: CGSize{Width: 19, Height: 27}})
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

func (v *SettingsView) addLabel(content objc.ID, y float64, title string) objc.ID {
	return v.addLabelWidth(content, y, title, tabContentW-300)
}

func (v *SettingsView) addLabelWidth(content objc.ID, y float64, title string, width float64) objc.ID {
	label := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(title))
	label.Send(sel("setFont:"), systemFont(13))
	label.Send(sel("setLineBreakMode:"), int64(4))
	label.Send(sel("setFrame:"), CGRect{Origin: CGPoint{X: 24, Y: y}, Size: CGSize{Width: width, Height: 18}})
	content.Send(sel("addSubview:"), label)
	return label
}

func (v *SettingsView) addHint(content objc.ID, y, height float64, text string) objc.ID {
	hint := objc.ID(classNSTextField).Send(sel("wrappingLabelWithString:"), nsString(text))
	hint.Send(sel("setFont:"), systemFont(11))
	hint.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("tertiaryLabelColor")))
	hint.Send(sel("setPreferredMaxLayoutWidth:"), tabContentW-48)
	hint.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: 24, Y: y},
		Size:   CGSize{Width: tabContentW - 48, Height: height},
	})
	content.Send(sel("addSubview:"), hint)
	return hint
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
