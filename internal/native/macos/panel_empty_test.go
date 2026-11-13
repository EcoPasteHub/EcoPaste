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

	if state := emptyStateFor("", 1, "", false); state.Title != "" {
		t.Fatalf("state with rows = %+v", state)
	}
	if state := emptyStateFor("读取失败", 0, "", false); !state.Error || state.Symbol != "exclamationmark.triangle" {
		t.Fatalf("error state = %+v", state)
	}
	if state := emptyStateFor("", 0, "", true); state.Symbol != "clock" || state.Title != i18n.T(i18n.PanelLoading) {
		t.Fatalf("loading state = %+v", state)
	}
	if state := emptyStateFor("", 0, "abc", false); state.Symbol != "magnifyingglass" ||
		state.Title != i18n.T(i18n.PanelNoMatches) || state.Detail != i18n.T(i18n.PanelNoMatchesDetail) {
		t.Fatalf("no-match state = %+v", state)
	}
	if state := emptyStateFor("", 0, "", false); state.Symbol != "doc.on.clipboard" ||
		state.Title != i18n.T(i18n.PanelEmpty) || state.Detail != i18n.T(i18n.PanelEmptyDetail) {
		t.Fatalf("empty state = %+v", state)
	}
}

func TestEmptyStateFollowsLanguage(t *testing.T) {
	previous := string(i18n.Current())
	t.Cleanup(func() { i18n.SetLanguage(previous) })
	i18n.SetLanguage("en-US")

	state := emptyStateFor("", 0, "", false)
	if state.Title != "No clipboard history yet" || state.Detail != "Copy something and it will show up here" {
		t.Fatalf("english empty state = %+v", state)
	}
}
