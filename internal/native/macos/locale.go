//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

// PreferredLanguage returns the first language the system prefers, as a
// BCP 47 tag, or "" when it reports none. It reads NSGlobalDomain because
// NSLocale.preferredLanguages includes the app's own forced locale.
func PreferredLanguage() string {
	defaults := objc.ID(class("NSUserDefaults")).Send(sel("standardUserDefaults"))
	global := defaults.Send(sel("persistentDomainForName:"), nsString("NSGlobalDomain"))
	if global != 0 {
		languages := global.Send(sel("objectForKey:"), nsString("AppleLanguages"))
		if languages != 0 && objc.Send[uint](languages, sel("count")) > 0 {
			return goString(objc.Send[objc.ID](languages, sel("objectAtIndex:"), uint(0)))
		}
	}
	system := objc.ID(class("NSLocale")).Send(sel("systemLocale"))
	if system != 0 {
		return goString(system.Send(sel("localeIdentifier")))
	}
	return ""
}
