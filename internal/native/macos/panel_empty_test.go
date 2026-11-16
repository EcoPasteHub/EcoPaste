//go:build darwin

package macos

import (
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/i18n"
)

func TestEmptyStateForContent(t *testing.T) {
	previous := string(i18n.Current())
	t.Cleanup(func() { i18n.SetLanguage(previous) })
	i18n.SetLanguage("zh-CN")

	if state := emptyStateFor(1, ""); state.Title != "" {
		t.Fatalf("state with rows = %+v", state)
	}
	if state := emptyStateFor(0, "abc"); state.Kind != emptyKindSearch ||
		state.Title != i18n.T(i18n.PanelNoMatches) || state.Detail != i18n.T(i18n.PanelNoMatchesDetail) {
		t.Fatalf("no-match state = %+v", state)
	}
	if state := emptyStateFor(0, ""); state.Kind != emptyKindHistory ||
		state.Title != i18n.T(i18n.PanelEmpty) || state.Detail != i18n.T(i18n.PanelEmptyDetail) {
		t.Fatalf("empty state = %+v", state)
	}
}

func TestEmptyStateIcons(t *testing.T) {
	if name := emptyIconSymbol(emptyKindHistory).Name(); name != "clipboard" {
		t.Fatalf("history icon = %q", name)
	}
	if name := emptyIconSymbol(emptyKindSearch).Name(); name != "search" {
		t.Fatalf("search icon = %q", name)
	}
}

func TestEmptyStateFollowsLanguage(t *testing.T) {
	previous := string(i18n.Current())
	t.Cleanup(func() { i18n.SetLanguage(previous) })
	i18n.SetLanguage("en-US")

	state := emptyStateFor(0, "")
	if state.Title != "No clipboard history yet" || state.Detail != "Copy something and it will show up here" {
		t.Fatalf("english empty state = %+v", state)
	}
}
