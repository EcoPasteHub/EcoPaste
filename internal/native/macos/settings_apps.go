//go:build darwin

package macos

import (
	"sync"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

const (
	settingsAppRowHeight               = settingsRowHeight
	settingsAppIconInset               = settingsRowInset
	settingsAppIconEdge                = 24.0
	settingsAppTextGap                 = 10.0
	settingsAppTextInset               = settingsAppIconInset + settingsAppIconEdge + settingsAppTextGap
	settingsAppPopupInset              = settingsRowInset
	settingsAppNameTag                 = 202
	settingsAppModeAllowed             = "allowed"
	settingsAppModeBlockCapture        = "blockCapture"
	settingsAppModeBlockWake           = "blockWake"
	settingsAppModeBlockCaptureAndWake = "blockCaptureAndWake"
)

// settingsAppRowView is one ordinary settings row: an icon, the app name and
// the mode popup. Rows are rebuilt only when the app list itself changes.
type settingsAppRowView struct {
	view  objc.ID
	popup objc.ID
}

// settingsAppControl links a row's mode popup to its view and row.
type settingsAppControl struct {
	view *SettingsView
	row  int
}

var (
	registerSettingsAppsOnce sync.Once
	settingsAppsTarget       objc.ID
	settingsAppsHostClass    objc.Class
	settingsAppControls      = map[objc.ID]settingsAppControl{}
)

func settingsAppModeOptions() []settingsOption {
	return []settingsOption{
		{Key: i18n.SettingsAppModeAllowed, Value: settingsAppModeAllowed},
		{Key: i18n.SettingsAppModeBlockCapture, Value: settingsAppModeBlockCapture},
		{Key: i18n.SettingsAppModeBlockWake, Value: settingsAppModeBlockWake},
		{Key: i18n.SettingsAppModeBlockCaptureAndWake, Value: settingsAppModeBlockCaptureAndWake},
	}
}

func (v *SettingsView) buildSourceAppsSection(document objc.ID, top float64) {
	registerSettingsAppsOnce.Do(registerSettingsAppsTargets)
	v.addSectionLabel(document, top, i18n.SettingsAppsSection)
	v.addInfoAfterText(document, settingsContentX+4, top+8, i18n.SettingsAppsSection,
		settingsSectionFont(), i18n.SettingsAppsIntro)

	v.appsDocument = document
	v.appsTop = top + 24
	v.appsHost = objc.ID(settingsAppsHostClass).Send(sel("new"))
	v.appsHost.Send(sel("setAutoresizesSubviews:"), false)
	setFrame(v.appsHost, settingsContentX, v.appsTop, settingsContentWidth, 0)
	document.Send(sel("addSubview:"), v.appsHost)
	v.appsHost.Send(sel("release"))
}

// settingsAppsEqual reports whether two app lists are identical, including
// each row's mode.
func settingsAppsEqual(a, b []SettingsApp) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// settingsAppsSameIdentity reports whether two app lists describe the same rows
// in the same order, ignoring their mode.
func settingsAppsSameIdentity(a, b []SettingsApp) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].ID != b[i].ID || a[i].Name != b[i].Name || a[i].IconPath != b[i].IconPath {
			return false
		}
	}
	return true
}

// settingsAppChangedRows returns the rows whose mode differs.
func settingsAppChangedRows(current, next []SettingsApp) []int {
	rows := make([]int, 0, len(current))
	for i := 0; i < len(current) && i < len(next); i++ {
		if current[i].Mode != next[i].Mode {
			rows = append(rows, i)
		}
	}
	return rows
}

// applyApps reconciles the ordinary settings rows with the loaded apps. A mode
// change only updates that row's popup; an app-list change rebuilds the rows
// and reflows the page height.
func (v *SettingsView) applyApps(values SettingsValues) {
	apps := append([]SettingsApp(nil), values.Apps...)
	if v.appsHost == 0 {
		v.apps = apps
		return
	}
	if settingsAppsEqual(v.apps, apps) {
		return
	}
	previous := v.apps
	if settingsAppsSameIdentity(previous, apps) {
		v.apps = apps
		for _, row := range settingsAppChangedRows(previous, apps) {
			v.selectSourceAppMode(row, apps[row].Mode)
		}
		return
	}
	v.rebuildSourceApps(apps)
}

// rebuildSourceApps replaces the rows inside the standard settings card.
func (v *SettingsView) rebuildSourceApps(apps []SettingsApp) {
	if v.appsHost == 0 {
		v.apps = apps
		return
	}
	for control, owner := range settingsAppControls {
		if owner.view == v {
			discardSettingsPopup(control)
			delete(settingsAppControls, control)
		}
	}
	subviews := objc.Send[objc.ID](v.appsHost, sel("subviews"))
	for i := int(objc.Send[uint](subviews, sel("count"))) - 1; i >= 0; i-- {
		view := objc.Send[objc.ID](subviews, sel("objectAtIndex:"), uint(i))
		view.Send(sel("removeFromSuperview"))
	}
	v.appsRowViews = v.appsRowViews[:0]
	v.apps = apps

	height := float64(len(apps)) * settingsAppRowHeight
	if height > 0 {
		addSettingsListCard(v.appsHost, CGRect{
			Size: CGSize{Width: settingsContentWidth, Height: height},
		})
	}
	popupWidth := settingsAppPopupWidth()
	for i, app := range apps {
		rowTop := float64(i) * settingsAppRowHeight
		row := objc.ID(class("NSView")).Send(sel("new"))
		row.Send(sel("setAutoresizesSubviews:"), false)
		setFrame(row, 0, rowTop, settingsContentWidth, settingsAppRowHeight)

		icon := objc.ID(class("NSImageView")).Send(sel("new"))
		setFrame(icon, settingsAppIconInset, (settingsAppRowHeight-settingsAppIconEdge)/2,
			settingsAppIconEdge, settingsAppIconEdge)
		icon.Send(sel("setImageScaling:"), int64(3))
		icon.Send(sel("setAccessibilityElement:"), false)
		if app.IconPath != "" {
			icon.Send(sel("setImage:"), loadImage(app.IconPath))
		}
		row.Send(sel("addSubview:"), icon)
		icon.Send(sel("release"))

		name := objc.ID(classNSTextField).Send(sel("labelWithString:"), nsString(app.Name))
		name.Send(sel("setTag:"), int64(settingsAppNameTag))
		name.Send(sel("setFont:"), systemFont(13))
		name.Send(sel("setTextColor:"), objc.ID(classNSColor).Send(sel("labelColor")))
		name.Send(sel("setLineBreakMode:"), int64(4))
		name.Send(sel("setMaximumNumberOfLines:"), int64(1))
		popupX := settingsContentWidth - settingsAppPopupInset - popupWidth
		nameWidth := popupX - settingsAppTextInset - settingsAppTextGap
		if nameWidth < 1 {
			nameWidth = 1
		}
		setFrame(name, settingsAppTextInset, (settingsAppRowHeight-18)/2, nameWidth, 18)
		row.Send(sel("addSubview:"), name)

		popup := newSettingsPopup(CGRect{
			Origin: CGPoint{X: popupX, Y: (settingsAppRowHeight - settingsPopupHeight) / 2},
			Size:   CGSize{Width: popupWidth, Height: settingsPopupHeight},
		})
		popup.Send(sel("setTarget:"), settingsAppsTarget)
		popup.Send(sel("setAction:"), sel("appModeChanged:"))
		popup.Send(sel("setAccessibilityLabel:"), nsString(i18n.Tf(i18n.SettingsAppModeA11y, app.Name)))
		for _, option := range settingsAppModeOptions() {
			popup.Send(sel("addItemWithTitle:"), nsString(settingsOptionText(option)))
		}
		popup.Send(sel("selectItemAtIndex:"), int64(settingsAppModeIndex(app.Mode)))
		row.Send(sel("addSubview:"), popup)
		popup.Send(sel("release"))
		settingsAppControls[popup] = settingsAppControl{view: v, row: i}

		v.appsHost.Send(sel("addSubview:"), row)
		row.Send(sel("release"))
		v.appsRowViews = append(v.appsRowViews, settingsAppRowView{view: row, popup: popup})

		if i > 0 {
			v.addSourceAppSeparator(rowTop)
		}
	}
	setFrame(v.appsHost, settingsContentX, v.appsTop, settingsContentWidth, height)
	v.updateAppsPageHeight()
}

func (v *SettingsView) addSourceAppSeparator(top float64) {
	separator := objc.ID(classNSBox).Send(sel("new"))
	separator.Send(sel("setBoxType:"), int64(2))
	separator.Send(sel("setTitlePosition:"), int64(0))
	separator.Send(sel("setBorderColor:"), settingsSeparatorColor())
	setFrame(separator, settingsRowInset, top-1, settingsContentWidth-2*settingsRowInset, 1)
	v.appsHost.Send(sel("addSubview:"), separator)
	separator.Send(sel("release"))
}

// updateAppsPageHeight recalculates the clipboard page after rows are added or
// removed, because the app list is loaded after the native tree is built.
func (v *SettingsView) updateAppsPageHeight() {
	if v.appsDocument == 0 {
		return
	}
	height := settingsMeasuredPageHeight(v.appsDocument)
	setFrame(v.appsDocument, 0, 0, settingsWidth, height)
	v.pageHeights["clipboard"] = height
	if v.currentTab == "clipboard" {
		v.resizeWindow(false)
	}
}

func (v *SettingsView) selectSourceAppMode(row int, mode string) {
	if row < 0 || row >= len(v.appsRowViews) {
		return
	}
	v.appsRowViews[row].popup.Send(sel("selectItemAtIndex:"), int64(settingsAppModeIndex(mode)))
}

func settingsAppModeIndex(mode string) int {
	for i, option := range settingsAppModeOptions() {
		if option.Value == mode {
			return i
		}
	}
	return 0
}

// settingsAppPopupWidth reserves the width of the longest mode so the popup
// stays right aligned when the selected title changes.
func settingsAppPopupWidth() float64 {
	width := settingsPopupMinWidth
	for _, option := range settingsAppModeOptions() {
		width = max(width, settingsPopupWidth(settingsOptionText(option)))
	}
	return width
}

func (v *SettingsView) emitSourceAppMode(row int) {
	if row < 0 || row >= len(v.apps) {
		return
	}
	app := v.apps[row]
	v.cx.Emit("change", SettingsChange{
		Path:  "sourceApps.mode",
		Value: SettingsAppSelection{ID: app.ID, Name: app.Name, Mode: app.Mode},
	})
}

func registerSettingsAppsTargets() {
	var err error
	settingsAppsHostClass, err = objc.RegisterClass("EcoPasteSettingsAppsHost", class("NSView"), nil, nil,
		[]objc.MethodDef{{Cmd: sel("isFlipped"), Fn: func(objc.ID, objc.SEL) bool { return true }}})
	if err != nil {
		panic(err)
	}
	target, err := objc.RegisterClass("EcoPasteSettingsAppsTarget", class("NSObject"), nil, nil, []objc.MethodDef{
		{Cmd: sel("appModeChanged:"), Fn: func(_ objc.ID, _ objc.SEL, sender objc.ID) {
			control, ok := settingsAppControls[sender]
			if !ok || control.view == nil {
				return
			}
			v := control.view
			row := control.row
			if row < 0 || row >= len(v.apps) {
				return
			}
			index := int(objc.Send[int64](sender, sel("indexOfSelectedItem")))
			options := settingsAppModeOptions()
			if index < 0 || index >= len(options) {
				return
			}
			v.apps[row].Mode = options[index].Value
			v.emitSourceAppMode(row)
		}},
	})
	if err != nil {
		panic(err)
	}
	settingsAppsTarget = objc.ID(target).Send(sel("new"))
}
