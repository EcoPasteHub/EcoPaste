// Package i18n resolves user-facing text from the published language setting.
// It is a leaf package so native views and app wiring can share one catalog.
package i18n

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/nsgo-dev/nsgo/core"
	"golang.org/x/text/language"
)

// Language is the published appearance.language value. Auto follows the
// system language and is resolved before translation.
type Language string

const (
	LanguageAuto Language = "auto"
	LanguageZhCN Language = "zh-CN"
	LanguageEnUS Language = "en-US"
)

// DefaultLanguage is the published fallback when a locale cannot be matched.
const DefaultLanguage = LanguageZhCN

// Key identifies a translated string.
type Key string

//go:embed locales/*.json
var localeFS embed.FS

var (
	bundleOnce sync.Once
	bundle     *goi18n.Bundle

	localizerMu sync.Mutex
	localizers  = map[Language]*goi18n.Localizer{}

	current atomic.Value

	languageSignal *core.Signal[Language]
)

func init() {
	current.Store(DefaultLanguage)
}

// Supported reports whether language is a selectable catalog language,
// including auto.
func Supported(language Language) bool {
	switch language {
	case LanguageAuto, LanguageZhCN, LanguageEnUS:
		return true
	}
	return false
}

// MatchLocale maps a BCP 47 locale to the closest catalog language.
func MatchLocale(locale string) Language {
	tag, err := language.Parse(locale)
	if err == nil {
		base, _ := tag.Base()
		switch base.String() {
		case "zh":
			return LanguageZhCN
		case "en":
			return LanguageEnUS
		}
	}
	lower := strings.ToLower(strings.ReplaceAll(locale, "_", "-"))
	switch {
	case strings.HasPrefix(lower, "zh"):
		return LanguageZhCN
	case strings.HasPrefix(lower, "en"):
		return LanguageEnUS
	}
	return DefaultLanguage
}

// ResolveLanguage turns a stored setting into the language to show. Auto and
// empty settings follow system; unknown values keep the published default.
func ResolveLanguage(requested, system string) Language {
	switch Language(requested) {
	case LanguageAuto:
		return MatchLocale(system)
	case LanguageZhCN:
		return LanguageZhCN
	case LanguageEnUS:
		return LanguageEnUS
	case "":
		return DefaultLanguage
	}
	return DefaultLanguage
}

// SetLanguage selects the catalog used by T and Tf. Unknown values fall back
// to Simplified Chinese, matching the published default.
func SetLanguage(language string) {
	current.Store(ResolveLanguage(language, ""))
}

// Current reports the active language.
func Current() Language {
	return current.Load().(Language)
}

// Bind attaches the reactive language signal to the app graph. It must run on
// the graph's goroutine before any component reads Watch.
func Bind(cx *core.Scope) {
	if languageSignal == nil {
		languageSignal = core.NewSignal(cx, Current())
	}
}

// Publish makes language current and notifies components that read Watch. It
// must run on the graph's goroutine.
func Publish(language Language) {
	if language == LanguageAuto || !Supported(language) {
		language = DefaultLanguage
	}
	SetLanguage(string(language))
	if languageSignal != nil {
		languageSignal.Set(language)
	}
}

// Watch returns the active language and subscribes the running computation
// when the app graph is bound, so translated props update on language change.
func Watch(cx *core.Scope) Language {
	if languageSignal != nil {
		return languageSignal.Get()
	}
	return Current()
}

// T returns the text for key in the active language. Missing translations fall
// back to Simplified Chinese, then to the key itself.
func T(key Key) string {
	localizer := localizerFor(Current())
	text, _ := localizer.Localize(&goi18n.LocalizeConfig{
		MessageID: string(key),
		DefaultMessage: &goi18n.Message{
			ID:    string(key),
			Other: string(key),
		},
	})
	if text == "" {
		return string(key)
	}
	return text
}

// Error returns an error whose message is the text for key, for user-facing
// failures that carry no technical detail.
func Error(key Key) error {
	return errors.New(T(key))
}

// Tf is T with fmt.Sprintf arguments.
func Tf(key Key, args ...any) string {
	return fmt.Sprintf(T(key), args...)
}

func localizerFor(language Language) *goi18n.Localizer {
	localizerMu.Lock()
	defer localizerMu.Unlock()
	if localizer := localizers[language]; localizer != nil {
		return localizer
	}
	localizer := goi18n.NewLocalizer(loadBundle(), string(language))
	localizers[language] = localizer
	return localizer
}

func loadBundle() *goi18n.Bundle {
	bundleOnce.Do(func() {
		bundle = goi18n.NewBundle(language.Make(string(LanguageZhCN)))
		bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
		for _, lang := range []Language{LanguageZhCN, LanguageEnUS} {
			data, err := localeFS.ReadFile("locales/" + string(lang) + ".json")
			if err != nil {
				panic(fmt.Errorf("i18n: read %s catalog: %w", lang, err))
			}
			// go-i18n joins nested JSON object keys with dots, so the
			// exported Key values and call sites stay flat.
			if _, err := bundle.ParseMessageFileBytes(data, string(lang)+".json"); err != nil {
				panic(fmt.Errorf("i18n: parse %s catalog: %w", lang, err))
			}
		}
	})
	return bundle
}
