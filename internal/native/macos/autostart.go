//go:build darwin

package macos

import (
	"fmt"
	"sync"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

const (
	appServiceNotRegistered = 0
	appServiceEnabled       = 1
	appServiceNeedsApproval = 2
	appServiceNotFound      = 3
)

var (
	appServiceOnce  sync.Once
	appServiceError error
)

// SyncAutostart aligns the login item with the persisted setting without
// re-registering an already enabled service.
func SyncAutostart(enabled bool) error {
	status, err := autostartStatus()
	if err != nil {
		return err
	}
	if !autostartNeedsChange(status, enabled) {
		return nil
	}
	return setAutostart(enabled)
}

// SetAutostart registers or removes EcoPaste as a login item.
func SetAutostart(enabled bool) error {
	status, err := autostartStatus()
	if err != nil {
		return err
	}
	if !autostartNeedsChange(status, enabled) {
		return nil
	}
	return setAutostart(enabled)
}

func autostartNeedsChange(status int64, enabled bool) bool {
	if enabled {
		return status != appServiceEnabled && status != appServiceNeedsApproval
	}
	return status != appServiceNotRegistered && status != appServiceNotFound
}

func autostartStatus() (int64, error) {
	service, err := appService()
	if err != nil {
		return 0, err
	}
	var status int64
	onMain(func() {
		status = objc.Send[int64](service, sel("status"))
	})
	return status, nil
}

func setAutostart(enabled bool) error {
	service, err := appService()
	if err != nil {
		return err
	}
	var ok bool
	var action objc.SEL
	if enabled {
		action = sel("registerAndReturnError:")
	} else {
		action = sel("unregisterAndReturnError:")
	}
	onMain(func() {
		ok = objc.Send[bool](service, action, objc.ID(0))
	})
	if ok {
		return nil
	}
	status, err := autostartStatus()
	if err != nil {
		return fmt.Errorf(i18n.T(i18n.AutostartQuery), err)
	}
	return autostartError(status, enabled)
}

func autostartError(status int64, enabled bool) error {
	if !enabled {
		return fmt.Errorf(i18n.T(i18n.AutostartDisable), status)
	}
	if status == appServiceNotFound {
		return fmt.Errorf("%s", i18n.T(i18n.AutostartInstall))
	}
	return fmt.Errorf(i18n.T(i18n.AutostartEnable), status)
}

func appService() (objc.ID, error) {
	appServiceOnce.Do(func() {
		_, appServiceError = purego.Dlopen(
			"/System/Library/Frameworks/ServiceManagement.framework/ServiceManagement",
			purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	})
	if appServiceError != nil {
		return 0, fmt.Errorf(i18n.T(i18n.AutostartService), appServiceError)
	}
	appServiceClass := objc.GetClass("SMAppService")
	if appServiceClass == 0 {
		return 0, fmt.Errorf("%s", i18n.T(i18n.AutostartVersion))
	}
	var service objc.ID
	onMain(func() {
		service = objc.ID(appServiceClass).Send(sel("mainAppService"))
	})
	if service == 0 {
		return 0, fmt.Errorf("%s", i18n.T(i18n.AutostartUnavailable))
	}
	return service, nil
}
