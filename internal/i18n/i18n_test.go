package i18n

import (
	"encoding/json"
	"strings"
	"testing"

	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

func TestCatalogsShareEveryKey(t *testing.T) {
	zh := readCatalog(t, LanguageZhCN)
	en := readCatalog(t, LanguageEnUS)
	for key := range zh {
		if _, ok := en[key]; !ok {
			t.Errorf("missing en-US translation for %q", key)
		}
	}
	for key := range en {
		if _, ok := zh[key]; !ok {
			t.Errorf("missing zh-CN translation for %q", key)
		}
	}
}

func TestLogLevelLabelsUseTheirLocale(t *testing.T) {
	zh := readCatalog(t, LanguageZhCN)
	en := readCatalog(t, LanguageEnUS)
	for key, want := range map[string]string{
		"settings.diagnostics.logLevelAll":   "全部",
		"settings.diagnostics.logLevelError": "错误",
		"settings.diagnostics.logLevelWarn":  "警告",
		"settings.diagnostics.logLevelInfo":  "信息",
		"settings.diagnostics.logLevelDebug": "调试",
	} {
		if got := zh[key]; got != want {
			t.Errorf("zh-CN %s = %q, want %q", key, got, want)
		}
	}
	for key, want := range map[string]string{
		"settings.diagnostics.logLevelAll":   "All",
		"settings.diagnostics.logLevelError": "Error",
		"settings.diagnostics.logLevelWarn":  "Warning",
		"settings.diagnostics.logLevelInfo":  "Info",
		"settings.diagnostics.logLevelDebug": "Debug",
	} {
		if got := en[key]; got != want {
			t.Errorf("en-US %s = %q, want %q", key, got, want)
		}
	}
}

func TestCatalogFilesUseNestedObjects(t *testing.T) {
	var check func(lang, prefix string, values map[string]any)
	check = func(lang, prefix string, values map[string]any) {
		for key, value := range values {
			path := prefix + "." + key
			if prefix == "" {
				path = key
			}
			if strings.Contains(key, ".") {
				t.Errorf("%s has a flat key segment %q", lang, path)
			}
			switch node := value.(type) {
			case map[string]any:
				check(lang, path, node)
			case string:
			default:
				t.Errorf("%s key %q is not a string or namespace", lang, path)
			}
		}
	}
	for _, lang := range []Language{LanguageZhCN, LanguageEnUS} {
		data, err := localeFS.ReadFile("locales/" + string(lang) + ".json")
		if err != nil {
			t.Fatal(err)
		}
		var catalog map[string]any
		if err := json.Unmarshal(data, &catalog); err != nil {
			t.Fatal(err)
		}
		check(string(lang), "", catalog)
	}
}

func TestSetLanguageSelectsCatalog(t *testing.T) {
	t.Cleanup(func() { SetLanguage(string(LanguageZhCN)) })

	SetLanguage("en-US")
	if got := T(TrayQuit); got != "Quit EcoPaste" {
		t.Fatalf("en-US quit label = %q", got)
	}
	SetLanguage("zh-CN")
	if got := T(TrayQuit); got != "退出 EcoPaste" {
		t.Fatalf("zh-CN quit label = %q", got)
	}
}

func TestSetLanguageFallsBackToChinese(t *testing.T) {
	t.Cleanup(func() { SetLanguage(string(LanguageZhCN)) })

	SetLanguage("fr-FR")
	if Current() != LanguageZhCN {
		t.Fatalf("current language = %q", Current())
	}
}

func TestResolveLanguageUsesSystemForAuto(t *testing.T) {
	if got := ResolveLanguage("auto", "en-GB"); got != LanguageEnUS {
		t.Fatalf("auto en-GB = %q", got)
	}
	if got := ResolveLanguage("auto", "zh-Hant-TW"); got != LanguageZhCN {
		t.Fatalf("auto zh-Hant-TW = %q", got)
	}
	if got := ResolveLanguage("auto", "fr-FR"); got != DefaultLanguage {
		t.Fatalf("auto fr-FR = %q", got)
	}
	if got := ResolveLanguage("en-US", "zh-CN"); got != LanguageEnUS {
		t.Fatalf("explicit en-US = %q", got)
	}
}

func TestMissingTranslationFallsBackToDefault(t *testing.T) {
	key := Key("test.default-fallback")
	loadBundle().MustAddMessages(language.Make(string(LanguageZhCN)), &goi18n.Message{
		ID:    string(key),
		Other: "默认语言回退",
	})
	t.Cleanup(func() { SetLanguage(string(LanguageZhCN)) })

	SetLanguage(string(LanguageEnUS))
	if got := T(key); got != "默认语言回退" {
		t.Fatalf("fallback text = %q", got)
	}
}

func TestUnknownKeyReturnsItsName(t *testing.T) {
	if got := T(Key("missing.key")); got != "missing.key" {
		t.Fatalf("unknown key = %q", got)
	}
}

func TestTfFormatsArguments(t *testing.T) {
	SetLanguage("en-US")
	t.Cleanup(func() { SetLanguage(string(LanguageZhCN)) })

	if got := Tf(TrayVersion, "1.2.3"); got != "Version 1.2.3" {
		t.Fatalf("version label = %q", got)
	}
}

func readCatalog(t *testing.T, lang Language) map[string]string {
	t.Helper()
	data, err := localeFS.ReadFile("locales/" + string(lang) + ".json")
	if err != nil {
		t.Fatal(err)
	}
	messages, err := goi18n.ParseMessageBytes(data, json.Unmarshal)
	if err != nil {
		t.Fatal(err)
	}
	catalog := make(map[string]string, len(messages))
	for _, message := range messages {
		catalog[message.ID] = message.Other
	}
	return catalog
}
