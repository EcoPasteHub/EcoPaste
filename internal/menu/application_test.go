package menu

import (
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/nsgo-dev/nsgo/desktop"
)

func TestApplicationBuildsLocalizedMenuBar(t *testing.T) {
	cases := []struct {
		language string
		edit     string
		window   string
	}{
		{"en-US", "Edit", "Window"},
		{"zh-CN", "编辑", "窗口"},
	}
	for _, tc := range cases {
		t.Run(tc.language, func(t *testing.T) {
			i18n.SetLanguage(tc.language)
			items := Application(tc.language, nil, nil)
			if len(items) != 3 {
				t.Fatalf("menu count %d, want 3", len(items))
			}
			if items[0].Label != "" || len(items[0].Items) == 0 {
				t.Fatalf("app menu = %#v", items[0])
			}
			if items[1].Label != tc.edit {
				t.Errorf("edit label = %q, want %q", items[1].Label, tc.edit)
			}
			if items[2].Label != tc.window {
				t.Errorf("window label = %q, want %q", items[2].Label, tc.window)
			}
		})
	}
}

func TestApplicationUsesRoleLabels(t *testing.T) {
	items := Application("en-US", func() {}, nil)
	appRoles := rolesOf(items[0].Items)
	for _, want := range []desktop.MenuRole{
		desktop.MenuAbout, desktop.MenuHide, desktop.MenuHideOthers,
		desktop.MenuShowAll, desktop.MenuQuit,
	} {
		if !appRoles[want] {
			t.Errorf("app menu misses role %v", want)
		}
	}
	for _, item := range items[0].Items {
		if item.Role != desktop.MenuRoleNone && item.Role.Label() != "" && item.Label != "" {
			t.Errorf("role %v carries its own label %q", item.Role, item.Label)
		}
	}

	editRoles := rolesOf(items[1].Items)
	for _, want := range []desktop.MenuRole{
		desktop.MenuUndo, desktop.MenuRedo, desktop.MenuCut,
		desktop.MenuCopy, desktop.MenuPaste, desktop.MenuSelectAll,
	} {
		if !editRoles[want] {
			t.Errorf("edit menu misses role %v", want)
		}
	}

	windowRoles := rolesOf(items[2].Items)
	for _, want := range []desktop.MenuRole{
		desktop.MenuMinimize, desktop.MenuZoom,
		desktop.MenuToggleFullScreen, desktop.MenuClose,
	} {
		if !windowRoles[want] {
			t.Errorf("window menu misses role %v", want)
		}
	}
}

func TestApplicationSettingsItem(t *testing.T) {
	i18n.SetLanguage("zh-CN")
	if got := settingsItem(Application("zh-CN", nil, nil)); got != nil {
		t.Fatalf("settings item without callback = %#v", got)
	}

	opened := 0
	item := settingsItem(Application("zh-CN", func() { opened++ }, nil))
	if item == nil {
		t.Fatal("settings item missing")
	}
	if item.Label != i18n.T(i18n.TraySettings) {
		t.Errorf("settings label = %q", item.Label)
	}
	if item.Shortcut != "meta+," {
		t.Errorf("settings shortcut = %q", item.Shortcut)
	}
	item.OnSelect()
	if opened != 1 {
		t.Errorf("settings opened %d times", opened)
	}
}

func TestApplicationUpdateItem(t *testing.T) {
	i18n.SetLanguage("zh-CN")
	if got := updateItem(Application("zh-CN", nil, nil)); got != nil {
		t.Fatalf("update item without callback = %#v", got)
	}

	checked := 0
	item := updateItem(Application("zh-CN", nil, func() { checked++ }))
	if item == nil {
		t.Fatal("update item missing")
	}
	if item.Label != i18n.T(i18n.TrayCheckUpdates) {
		t.Errorf("update label = %q", item.Label)
	}
	if item.Shortcut != "" {
		t.Errorf("update shortcut = %q", item.Shortcut)
	}
	item.OnSelect()
	if checked != 1 {
		t.Errorf("update checked %d times", checked)
	}
}

func rolesOf(items []desktop.MenuItem) map[desktop.MenuRole]bool {
	roles := map[desktop.MenuRole]bool{}
	for _, item := range items {
		roles[item.Role] = true
	}
	return roles
}

func settingsItem(items []desktop.MenuItem) *desktop.MenuItem {
	return labelledItem(items, i18n.T(i18n.TraySettings))
}

func updateItem(items []desktop.MenuItem) *desktop.MenuItem {
	return labelledItem(items, i18n.T(i18n.TrayCheckUpdates))
}

func labelledItem(items []desktop.MenuItem, label string) *desktop.MenuItem {
	for i, item := range items[0].Items {
		if item.Role == desktop.MenuRoleNone && item.Label == label && !item.Separator {
			return &items[0].Items[i]
		}
	}
	return nil
}
