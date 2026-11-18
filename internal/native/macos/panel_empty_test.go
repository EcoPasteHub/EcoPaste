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

	if state := emptyStateFor(1, "", "", ""); state.Title != "" {
		t.Fatalf("state with rows = %+v", state)
	}
	if state := emptyStateFor(0, "abc", "", ""); state.Kind != emptyKindSearch ||
		state.Title != i18n.T(i18n.PanelNoMatches) || state.Detail != i18n.T(i18n.PanelNoMatchesDetail) {
		t.Fatalf("no-match state = %+v", state)
	}
	if state := emptyStateFor(0, "", "", ""); state.Kind != emptyKindHistory ||
		state.Title != i18n.T(i18n.PanelEmpty) || state.Detail != i18n.T(i18n.PanelEmptyDetail) {
		t.Fatalf("empty state = %+v", state)
	}
}

func TestEmptyStateForCategories(t *testing.T) {
	previous := string(i18n.Current())
	t.Cleanup(func() { i18n.SetLanguage(previous) })
	i18n.SetLanguage("zh-CN")

	cases := []struct {
		filter string
		kind   emptyStateKind
		title  i18n.Key
		detail i18n.Key
	}{
		{"text", emptyKindText, i18n.PanelEmptyText, i18n.PanelEmptyTextDetail},
		{"image", emptyKindImage, i18n.PanelEmptyImage, i18n.PanelEmptyImageDetail},
		{"files", emptyKindFiles, i18n.PanelEmptyFiles, i18n.PanelEmptyFilesDetail},
		{"favorite", emptyKindFavorites, i18n.PanelEmptyFavorites, i18n.PanelEmptyFavoritesDetail},
	}
	for _, c := range cases {
		state := emptyStateFor(0, "", c.filter, "")
		if state.Kind != c.kind || state.Title != i18n.T(c.title) || state.Detail != i18n.T(c.detail) {
			t.Fatalf("%s empty state = %+v", c.filter, state)
		}
		if state.Title == i18n.T(i18n.PanelEmpty) {
			t.Fatalf("%s should not reuse the all-history copy", c.filter)
		}
	}
}

func TestEmptyStateForGroup(t *testing.T) {
	previous := string(i18n.Current())
	t.Cleanup(func() { i18n.SetLanguage(previous) })
	i18n.SetLanguage("zh-CN")

	state := emptyStateFor(0, "", "", "group-1")
	if state.Kind != emptyKindHistory || state.Title != i18n.T(i18n.PanelEmptyGroup) ||
		state.Detail != i18n.T(i18n.PanelEmptyGroupDetail) {
		t.Fatalf("group empty state = %+v", state)
	}

	// A search query still wins over the group copy.
	state = emptyStateFor(0, "abc", "", "group-1")
	if state.Kind != emptyKindSearch {
		t.Fatalf("search should win over group, got %+v", state)
	}
}

func TestEmptyStateIcons(t *testing.T) {
	cases := map[emptyStateKind]string{
		emptyKindHistory:   "clipboard",
		emptyKindSearch:    "search",
		emptyKindText:      "file-text",
		emptyKindImage:     "image",
		emptyKindFiles:     "folder",
		emptyKindFavorites: "star",
	}
	for kind, want := range cases {
		if name := emptyIconSymbol(kind).Name(); name != want {
			t.Fatalf("icon for kind %d = %q, want %q", kind, name, want)
		}
	}
}

func TestEmptyStateFollowsLanguage(t *testing.T) {
	previous := string(i18n.Current())
	t.Cleanup(func() { i18n.SetLanguage(previous) })
	i18n.SetLanguage("en-US")

	if state := emptyStateFor(0, "", "", ""); state.Title != "No clipboard history yet" ||
		state.Detail != "Copy something and it will show up here" {
		t.Fatalf("english empty state = %+v", state)
	}
	if state := emptyStateFor(0, "", "text", ""); state.Title != "No text yet" ||
		state.Detail != "Copy some text and it will show up here" {
		t.Fatalf("english text empty state = %+v", state)
	}
	if state := emptyStateFor(0, "", "", "group-1"); state.Title != "This group is empty" {
		t.Fatalf("english group empty state = %+v", state)
	}
}
