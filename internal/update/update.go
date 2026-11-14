package update

import (
	"log"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
	"github.com/nsgo-dev/nsgo/core"
)

// CheckNow asks the platform updater for a newer release. Sparkle owns
// its download, signature verification, install and restart on macOS;
// other platforms only receive a concise result alert.
func CheckNow(cx *core.Scope) {
	cx.CheckForUpdates(false, func(result core.UpdateCheck) {
		if result.Shown {
			return
		}
		if result.Err != nil {
			log.Printf("check updates: %v", result.Err)
		}
		title, message := Describe(result, macos.AppVersion())
		if result.Err == nil && result.Version != "" && result.URL != "" {
			if macos.Confirm(title, message, i18n.T(i18n.UpdateOpenDownload)) {
				if err := macos.OpenURL(result.URL); err != nil {
					macos.ShowMessage(i18n.T(i18n.UpdateCheckFailedTitle), err.Error())
				}
			}
			return
		}
		macos.ShowMessage(title, message)
	})
}

// Describe turns an updater answer into the alert shown where the
// platform does not own the update window.
func Describe(result core.UpdateCheck, fallbackVersion string) (title, message string) {
	current := result.Current
	if current == "" {
		current = fallbackVersion
	}
	switch {
	case result.Err != nil:
		return i18n.T(i18n.UpdateCheckFailedTitle), i18n.T(i18n.UpdateCheckFailedBody)
	case result.Version == "":
		return i18n.T(i18n.UpdateUpToDateTitle), i18n.Tf(i18n.UpdateUpToDateBody, current)
	default:
		return i18n.T(i18n.UpdateAvailableTitle),
			i18n.Tf(i18n.UpdateAvailableBody, result.Version, current)
	}
}
