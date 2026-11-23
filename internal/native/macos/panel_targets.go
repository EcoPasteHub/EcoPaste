//go:build darwin

package macos

func registerPanelTargets() {
	registrations := []func() error{
		registerPanelViewClass,
		registerPanelControlTargetClass,
		registerNavClasses,
		registerIconButtonClass,
		registerSearchClasses,
		registerPanelBackdrop,
		registerPanelObserverClass,
		registerGroupPopoverClasses,
		registerNotePopoverClasses,
	}
	for _, register := range registrations {
		if err := register(); err != nil {
			panic(err)
		}
	}
	registerHistoryTargets()
	registerPreviewTargets()
}
