package app

import (
	"fmt"

	"github.com/EcoPasteHub/EcoPaste/internal/storage"
	"github.com/nsgo-dev/nsgo/core"
)

const developmentBundleID = storage.DevelopmentBundleID

// developmentIdentityError prevents a development build from claiming the
// release identity, which would share LaunchServices state and settings with
// an installed EcoPaste build.
func developmentIdentityError(info core.AppInfo) error {
	if info.Development && info.ID != "" && info.ID != developmentBundleID {
		return fmt.Errorf("development build uses bundle id %q; build with -id %s", info.ID, developmentBundleID)
	}
	return nil
}
