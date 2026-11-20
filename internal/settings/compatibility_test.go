package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRetiredSettingsAreIgnored(t *testing.T) {
	cases := map[string]string{
		"published values": `{
			"general":{"autoStart":true,"trayIcon":false,"runAsAdmin":true},
			"appearance":{"theme":"dark"},
			"shortcuts":{"openClipboard":"Command+Shift+V","winV":true},
			"clipboard":{
				"window":{"position":"center","lightweightMode":true,"idleDestroySeconds":60},
				"preview":{"spaceEnabled":false,"hoverEnabled":true,"hoverDelayMs":500}
			},
			"update":{"autoCheck":true,"frequency":"weekly","includeBeta":true,
				"includeNightly":true,"lastCheckedAt":"2026-11-20T00:00:00Z","skippedVersion":"0.2.0"}
		}`,
		"invalid retired values": `{
			"general":{"autoStart":true,"trayIcon":false,"runAsAdmin":"obsolete"},
			"appearance":{"theme":"dark"},
			"shortcuts":{"openClipboard":"Command+Shift+V","winV":{}},
			"clipboard":{
				"window":{"position":"center","lightweightMode":[],"idleDestroySeconds":"obsolete"},
				"preview":{"spaceEnabled":false,"hoverEnabled":{},"hoverDelayMs":200}
			},
			"update":{"autoCheck":[],"frequency":"hourly","includeBeta":{},
				"includeNightly":"obsolete","lastCheckedAt":42,"skippedVersion":false}
		}`,
	}
	for name, content := range cases {
		for _, mode := range []string{"load", "backup patch"} {
			t.Run(name+"/"+mode, func(t *testing.T) {
				path := filepath.Join(t.TempDir(), FileName)
				if mode == "load" {
					if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
						t.Fatal(err)
					}
				}
				store, err := Open(path)
				if err != nil {
					t.Fatal(err)
				}
				if mode == "backup patch" {
					data, err := os.ReadFile(path)
					if err != nil {
						t.Fatal(err)
					}
					assertRetiredSettingsAbsent(t, data)
					var patch map[string]any
					if err := json.Unmarshal([]byte(content), &patch); err != nil {
						t.Fatal(err)
					}
					if _, err := store.Update(patch); err != nil {
						t.Fatal(err)
					}
				} else if data, err := os.ReadFile(path); err != nil || string(data) != content {
					t.Fatalf("loading rewrote existing settings: %v", err)
				}

				want := Default()
				want.General.AutoStart = true
				want.General.TrayIcon = false
				want.Appearance.Theme = ThemeDark
				want.Shortcuts.OpenClipboard = "Command+Shift+V"
				want.Clipboard.Window.Position = PositionCenter
				want.Clipboard.Preview.SpaceEnabled = false
				if got := store.Snapshot(); !reflect.DeepEqual(got, want) {
					t.Fatalf("active settings changed: got %+v, want %+v", got, want)
				}
				if _, err := store.Update(map[string]any{"logging": map[string]any{"level": "debug"}}); err != nil {
					t.Fatal(err)
				}
				want.Logging.Level = LogLevelDebug
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				assertRetiredSettingsAbsent(t, data)
				reopened, err := Open(path)
				if err != nil {
					t.Fatal(err)
				}
				if got := reopened.Snapshot(); !reflect.DeepEqual(got, want) {
					t.Fatalf("active settings lost after saving: got %+v, want %+v", got, want)
				}
			})
		}
	}
}

func assertRetiredSettingsAbsent(t *testing.T, data []byte) {
	t.Helper()
	for _, key := range []string{
		"runAsAdmin", "winV", "lightweightMode", "idleDestroySeconds",
		"hoverEnabled", "hoverDelayMs", "autoCheck", "frequency", "includeBeta",
		"includeNightly", "lastCheckedAt", "skippedVersion", "update",
	} {
		if strings.Contains(string(data), `"`+key+`"`) {
			t.Errorf("encoded settings contain retired key %q", key)
		}
	}
}
