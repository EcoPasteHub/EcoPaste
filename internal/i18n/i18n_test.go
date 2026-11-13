package i18n

import "testing"

func TestCatalogsShareEveryKey(t *testing.T) {
	for key := range catalogZhCN {
		if _, ok := catalogEnUS[key]; !ok {
			t.Errorf("missing en-US translation for %q", key)
		}
	}
	for key := range catalogEnUS {
		if _, ok := catalogZhCN[key]; !ok {
			t.Errorf("missing zh-CN translation for %q", key)
		}
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
