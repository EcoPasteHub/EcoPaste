// Package i18n resolves user-facing text from the published language setting.
// It is a leaf package so native views and app wiring can share one catalog.
package i18n

import (
	"fmt"
	"strings"
	"sync/atomic"
)

// Language is the published appearance.language value.
type Language string

const (
	LanguageZhCN Language = "zh-CN"
	LanguageEnUS Language = "en-US"
)

// Key identifies a translated string.
type Key string

var current atomic.Value

var catalogs = map[Language]map[Key]string{
	LanguageZhCN: catalogZhCN,
	LanguageEnUS: catalogEnUS,
}

func init() {
	current.Store(LanguageZhCN)
}

// SetLanguage selects the catalog used by T and Tf. Unknown values fall back
// to Simplified Chinese, matching the published default.
func SetLanguage(language string) {
	if strings.EqualFold(language, string(LanguageEnUS)) {
		current.Store(LanguageEnUS)
		return
	}
	current.Store(LanguageZhCN)
}

// Current reports the active language.
func Current() Language {
	return current.Load().(Language)
}

// T returns the text for key in the active language. Missing translations fall
// back to Simplified Chinese, then to the key itself.
func T(key Key) string {
	if text, ok := catalogs[Current()][key]; ok {
		return text
	}
	if text, ok := catalogs[LanguageZhCN][key]; ok {
		return text
	}
	return string(key)
}

// Tf is T with fmt.Sprintf arguments.
func Tf(key Key, args ...any) string {
	return fmt.Sprintf(T(key), args...)
}
