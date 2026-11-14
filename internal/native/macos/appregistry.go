//go:build darwin

package macos

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego/objc"
)

// RunningApp is one regular application currently visible in the Dock.
type RunningApp struct {
	BundleID string
	Name     string
	Path     string
}

// RunningApps lists running regular applications, deduplicated by bundle id.
func RunningApps() []RunningApp {
	var apps []RunningApp
	onMain(func() {
		workspace := objc.ID(class("NSWorkspace")).Send(sel("sharedWorkspace"))
		running := objc.Send[objc.ID](workspace, sel("runningApplications"))
		if running == 0 {
			return
		}
		count := objc.Send[uint](running, sel("count"))
		seen := make(map[string]struct{}, count)
		for i := uint(0); i < count; i++ {
			app := objc.Send[objc.ID](running, sel("objectAtIndex:"), i)
			if app == 0 || objc.Send[int64](app, sel("activationPolicy")) != activationPolicyRegular {
				continue
			}
			bundleID := goString(objc.Send[objc.ID](app, sel("bundleIdentifier")))
			if bundleID == "" {
				continue
			}
			if _, ok := seen[bundleID]; ok {
				continue
			}
			seen[bundleID] = struct{}{}
			name := goString(objc.Send[objc.ID](app, sel("localizedName")))
			if name == "" {
				name = bundleID
			}
			path := ""
			if url := objc.Send[objc.ID](app, sel("bundleURL")); url != 0 {
				path = goString(objc.Send[objc.ID](url, sel("path")))
			}
			apps = append(apps, RunningApp{BundleID: bundleID, Name: name, Path: path})
		}
	})
	return apps
}

// ResolveApp reads a .app bundle's identity and display name.
func ResolveApp(path string) (RunningApp, error) {
	if filepath.Ext(path) != ".app" {
		return RunningApp{}, fmt.Errorf("%s", i18n.T(i18n.ErrorSelectBundle))
	}
	info, err := os.Stat(path)
	if err != nil {
		return RunningApp{}, fmt.Errorf(i18n.T(i18n.ErrorAppMissing), err)
	}
	if !info.IsDir() {
		return RunningApp{}, fmt.Errorf("%s", i18n.T(i18n.ErrorAppPackage))
	}

	result := RunningApp{Path: path}
	onMain(func() {
		bundle := objc.Send[objc.ID](objc.ID(class("NSBundle")), sel("bundleWithPath:"), nsString(path))
		if bundle == 0 {
			return
		}
		result.BundleID = bundleString(bundle, sel("infoDictionary"), "CFBundleIdentifier")
		result.Name = bundleDisplayName(bundle)
	})
	if result.BundleID == "" {
		return RunningApp{}, fmt.Errorf("%s", i18n.T(i18n.ErrorAppIdentifier))
	}
	if result.Name == "" {
		result.Name = filepath.Base(path[:len(path)-len(".app")])
	}
	return result, nil
}

func bundleDisplayName(bundle objc.ID) string {
	for _, dictionary := range []objc.SEL{sel("localizedInfoDictionary"), sel("infoDictionary")} {
		for _, key := range []string{"CFBundleDisplayName", "CFBundleName"} {
			if name := bundleString(bundle, dictionary, key); name != "" {
				return name
			}
		}
	}
	return ""
}

func bundleString(bundle objc.ID, dictionary objc.SEL, key string) string {
	info := objc.Send[objc.ID](bundle, dictionary)
	if info == 0 {
		return ""
	}
	return goString(objc.Send[objc.ID](info, sel("objectForKey:"), nsString(key)))
}
