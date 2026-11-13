//go:build darwin

package macos

import (
	"strings"

	"github.com/ebitengine/purego/objc"
)

// EditNote shows a native note editor and returns the trimmed note.
func EditNote(note string) (string, bool) {
	var result string
	var saved bool
	onMain(func() {
		suppressPanelBlur.Store(true)
		defer suppressPanelBlur.Store(false)

		app := objc.ID(class("NSApplication")).Send(sel("sharedApplication"))
		app.Send(sel("activateIgnoringOtherApps:"), true)

		alert := objc.ID(class("NSAlert")).Send(sel("new"))
		defer alert.Send(sel("release"))
		alert.Send(sel("setMessageText:"), nsString("编辑备注"))
		alert.Send(sel("setInformativeText:"), nsString("备注会显示在列表中，并参与剪贴板搜索。"))
		alert.Send(sel("addButtonWithTitle:"), nsString("保存"))
		alert.Send(sel("addButtonWithTitle:"), nsString("取消"))

		text := objc.ID(class("NSTextView")).Send(sel("alloc")).
			Send(sel("initWithFrame:"), CGRect{Size: CGSize{Width: 360, Height: 96}})
		text.Send(sel("setString:"), nsString(note))
		text.Send(sel("setFont:"), systemFont(13))
		text.Send(sel("setRichText:"), false)
		text.Send(sel("setAllowsUndo:"), true)
		text.Send(sel("setHorizontallyResizable:"), false)
		text.Send(sel("setVerticallyResizable:"), true)
		text.Send(sel("setTextContainerInset:"), CGSize{Width: 6, Height: 6})
		if container := text.Send(sel("textContainer")); container != 0 {
			container.Send(sel("setWidthTracksTextView:"), true)
			container.Send(sel("setContainerSize:"), CGSize{Width: 360, Height: 10000})
		}

		scroll := objc.ID(class("NSScrollView")).Send(sel("new"))
		scroll.Send(sel("setFrame:"), CGRect{Size: CGSize{Width: 360, Height: 96}})
		scroll.Send(sel("setHasVerticalScroller:"), true)
		scroll.Send(sel("setAutohidesScrollers:"), true)
		scroll.Send(sel("setBorderType:"), int64(2))
		scroll.Send(sel("setDocumentView:"), text)
		alert.Send(sel("setAccessoryView:"), scroll)

		window := objc.Send[objc.ID](alert, sel("window"))
		window.Send(sel("setLevel:"), int64(1000))
		window.Send(sel("makeKeyAndOrderFront:"), objc.ID(0))
		window.Send(sel("makeFirstResponder:"), text)
		if alert.Send(sel("runModal")) == 1000 {
			result = strings.TrimSpace(goString(text.Send(sel("string"))))
			saved = true
		}
		text.Send(sel("release"))
		scroll.Send(sel("release"))
	})
	return result, saved
}
