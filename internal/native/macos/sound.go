//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

// copySoundName is the short system sound played when a new clipboard
// item is captured. It ships with macOS, so no audio asset is bundled.
const copySoundName = "Pop"

// PlayCopySound plays the capture feedback asynchronously.
func PlayCopySound() {
	sound := copySound()
	if sound == 0 {
		return
	}
	sound.Send(sel("stop"))
	sound.Send(sel("play"))
}

func copySound() objc.ID {
	return objc.ID(class("NSSound")).Send(sel("soundNamed:"), nsString(copySoundName))
}
