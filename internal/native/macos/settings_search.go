//go:build darwin

package macos

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/ebitengine/purego/objc"
)

const settingsSearchLimit = 8

type settingsSearchEntry struct {
	title     string
	detail    string
	path      string
	keywords  string
	tab       string
	tabTitle  string
	content   objc.ID
	highlight CGRect
}

func rowHighlightFrame(y float64) CGRect {
	return CGRect{
		Origin: CGPoint{X: 12, Y: y - 8},
		Size:   CGSize{Width: tabContentW - 24, Height: 34},
	}
}

func (v *SettingsView) addSearchEntry(content objc.ID, title, path, detail, keywords string, highlight CGRect) {
	if content == 0 || title == "" || path == "" {
		return
	}
	v.searchEntries = append(v.searchEntries, settingsSearchEntry{
		title:     title,
		detail:    detail,
		path:      path,
		keywords:  keywords,
		tab:       v.currentTab,
		tabTitle:  v.currentTabTitle,
		content:   content,
		highlight: highlight,
	})
}

func (v *SettingsView) buildSearch() {
	v.searchField = objc.ID(classNSSearchField).Send(sel("new"))
	v.searchField.Send(sel("setPlaceholderString:"), nsString("搜索设置"))
	v.searchField.Send(sel("setFont:"), systemFont(13))
	v.searchField.Send(sel("setFrame:"), CGRect{
		Origin: CGPoint{X: 24, Y: settingsHeight - 35},
		Size:   CGSize{Width: 280, Height: 28},
	})
	v.searchField.Send(sel("setSendsSearchStringImmediately:"), true)
	v.searchField.Send(sel("setSendsWholeSearchString:"), false)
	v.searchField.Send(sel("setTarget:"), settingsTarget)
	v.searchField.Send(sel("setAction:"), sel("settingsSearchChanged:"))
	v.searchField.Send(sel("setAccessibilityLabel:"), nsString("搜索所有设置"))
	v.searchField.Send(sel("setAccessibilityHelp:"),
		nsString("输入名称、说明或设置路径，然后从右侧结果中选择要跳转的设置"))
	v.view.Send(sel("addSubview:"), v.searchField)
	settingsViews[v.searchField] = v

	v.searchResults = objc.ID(class("NSPopUpButton")).Send(sel("alloc")).
		Send(sel("initWithFrame:pullsDown:"), CGRect{
			Origin: CGPoint{X: 312, Y: settingsHeight - 38},
			Size:   CGSize{Width: 224, Height: 32},
		}, false)
	v.searchResults.Send(sel("setFont:"), systemFont(12))
	v.searchResults.Send(sel("setTarget:"), settingsTarget)
	v.searchResults.Send(sel("setAction:"), sel("settingsSearchPicked:"))
	v.searchResults.Send(sel("setAccessibilityLabel:"), nsString("设置搜索结果"))
	v.searchResults.Send(sel("setAccessibilityHelp:"), nsString("选择匹配设置并跳转到对应分类"))
	v.searchResults.Send(sel("setHidden:"), true)
	if menu := v.searchResults.Send(sel("menu")); menu != 0 {
		menu.Send(sel("setAutoenablesItems:"), false)
	}
	v.view.Send(sel("addSubview:"), v.searchResults)
	settingsViews[v.searchResults] = v

	v.searchHighlight = objc.ID(class("NSView")).Send(sel("new"))
	v.searchHighlight.Send(sel("setWantsLayer:"), true)
	v.searchHighlight.Send(sel("setHidden:"), true)
	v.searchHighlight.Send(sel("setAccessibilityElement:"), false)
	if layer := v.searchHighlight.Send(sel("layer")); layer != 0 {
		layer.Send(sel("setCornerRadius:"), 7.0)
		color := objc.ID(classNSColor).Send(sel("systemBlueColor")).
			Send(sel("colorWithAlphaComponent:"), 0.16)
		layer.Send(sel("setBackgroundColor:"), color.Send(sel("CGColor")))
	}
}

func (v *SettingsView) searchChanged(sender objc.ID) {
	v.updateSearchResults(goString(sender.Send(sel("stringValue"))))
}

func (v *SettingsView) searchResultPicked(sender objc.ID) {
	index := int(objc.Send[int64](sender, sel("indexOfSelectedItem"))) - 1
	if index < 0 || index >= len(v.searchMatches) {
		return
	}
	entry := v.searchMatches[index]
	v.selectTab(entry.tab)
	v.showSearchHighlight(entry)
	v.searchField.Send(sel("setStringValue:"), nsString(""))
	v.updateSearchResults("")
}

func (v *SettingsView) updateSearchResults(query string) {
	v.searchMatches = matchSettingsSearch(v.searchEntries, query)
	v.searchResults.Send(sel("removeAllItems"))
	if strings.TrimSpace(query) == "" {
		v.searchResults.Send(sel("setHidden:"), true)
		return
	}
	v.searchResults.Send(sel("setHidden:"), false)
	if len(v.searchMatches) == 0 {
		v.searchResults.Send(sel("addItemWithTitle:"), nsString("无匹配设置"))
		v.searchResults.Send(sel("setEnabled:"), false)
		v.searchResults.Send(sel("setToolTip:"), nsString("没有找到匹配的设置"))
		v.searchResults.Send(sel("selectItemAtIndex:"), 0)
		return
	}

	v.searchResults.Send(sel("setEnabled:"), true)
	v.searchResults.Send(sel("addItemWithTitle:"),
		nsString(fmt.Sprintf("找到 %d 项匹配", len(v.searchMatches))))
	header := v.searchResults.Send(sel("lastItem"))
	header.Send(sel("setEnabled:"), false)
	for _, entry := range v.searchMatches {
		title := entry.title + "  ·  " + entry.tabTitle
		v.searchResults.Send(sel("addItemWithTitle:"), nsString(title))
		item := v.searchResults.Send(sel("lastItem"))
		item.Send(sel("setEnabled:"), true)
		if entry.detail != "" {
			item.Send(sel("setToolTip:"), nsString(entry.detail))
		}
	}
	if menu := v.searchResults.Send(sel("menu")); menu != 0 {
		menu.Send(sel("setMinimumWidth:"), 380.0)
	}
	v.searchResults.Send(sel("selectItemAtIndex:"), 0)
	v.searchResults.Send(sel("setToolTip:"), nsString("从菜单中选择匹配的设置"))
}

func (v *SettingsView) showSearchHighlight(entry settingsSearchEntry) {
	if v.searchHighlightTimer != 0 {
		v.searchHighlightTimer.Send(sel("invalidate"))
		delete(settingsTimers, v.searchHighlightTimer)
		v.searchHighlightTimer = 0
	}
	v.searchHighlight.Send(sel("removeFromSuperview"))
	v.searchHighlight.Send(sel("setFrame:"), entry.highlight)
	entry.content.Send(sel("addSubview:positioned:relativeTo:"),
		v.searchHighlight, int64(-1), objc.ID(0))
	v.searchHighlight.Send(sel("setHidden:"), false)
	v.searchHighlightTimer = objc.ID(class("NSTimer")).Send(
		sel("scheduledTimerWithTimeInterval:target:selector:userInfo:repeats:"),
		1.6, settingsTarget, sel("settingsSearchHighlightExpired:"), objc.ID(0), false)
	settingsTimers[v.searchHighlightTimer] = v
}

func matchSettingsSearch(entries []settingsSearchEntry, query string) []settingsSearchEntry {
	terms := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	if len(terms) == 0 {
		return nil
	}
	matches := make([]settingsSearchEntry, 0, settingsSearchLimit)
	seen := make(map[string]struct{}, settingsSearchLimit)
	for _, entry := range entries {
		if _, ok := seen[entry.path]; ok {
			continue
		}
		haystack := settingsSearchHaystack(entry)
		matched := true
		for _, term := range terms {
			if !strings.Contains(haystack, term) {
				matched = false
				break
			}
		}
		if !matched {
			continue
		}
		seen[entry.path] = struct{}{}
		matches = append(matches, entry)
		if len(matches) == settingsSearchLimit {
			break
		}
	}
	return matches
}

func settingsSearchHaystack(entry settingsSearchEntry) string {
	tabTitle := entry.tabTitle
	if tabTitle == "" {
		tabTitle = settingsTabTitles[entry.tab]
	}
	return strings.ToLower(strings.Join([]string{
		entry.title,
		entry.detail,
		entry.path,
		settingsPathWords(entry.path),
		entry.keywords,
		entry.tab,
		tabTitle,
	}, " "))
}

func settingsPathWords(path string) string {
	words := make([]string, 0, 4)
	var current strings.Builder
	flush := func() {
		if current.Len() == 0 {
			return
		}
		words = append(words, current.String())
		current.Reset()
	}
	for _, r := range path {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			flush()
			continue
		}
		if unicode.IsUpper(r) {
			flush()
		}
		current.WriteRune(unicode.ToLower(r))
	}
	flush()
	return strings.Join(words, " ")
}
