//go:build darwin

package macos

import (
	"fmt"
	"os"

	"github.com/ebitengine/purego/objc"
)

// OpenURL opens an http, file or mailto URL with the default handler.
func OpenURL(raw string) error {
	var result error
	onMain(func() {
		url := objc.ID(class("NSURL")).Send(sel("URLWithString:"), nsString(raw))
		if url == 0 {
			result = fmt.Errorf("无法解析链接")
			return
		}
		if !objc.Send[bool](objc.ID(class("NSWorkspace")).Send(sel("sharedWorkspace")), sel("openURL:"), url) {
			result = fmt.Errorf("无法打开链接")
		}
	})
	return result
}

// OpenPath opens one existing file or folder with its default app.
func OpenPath(path string) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("文件不存在: %w", err)
	}
	var result error
	onMain(func() {
		url := objc.ID(class("NSURL")).Send(sel("fileURLWithPath:"), nsString(path))
		if !objc.Send[bool](objc.ID(class("NSWorkspace")).Send(sel("sharedWorkspace")), sel("openURL:"), url) {
			result = fmt.Errorf("无法打开文件")
		}
	})
	return result
}

// RevealPaths selects existing files in Finder.
func RevealPaths(paths []string) error {
	var existing []string
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			existing = append(existing, path)
		}
	}
	if len(existing) == 0 {
		return fmt.Errorf("文件不存在")
	}
	onMain(func() {
		urls := objc.ID(class("NSMutableArray")).Send(sel("array"))
		for _, path := range existing {
			url := objc.ID(class("NSURL")).Send(sel("fileURLWithPath:"), nsString(path))
			urls.Send(sel("addObject:"), url)
		}
		workspace := objc.ID(class("NSWorkspace")).Send(sel("sharedWorkspace"))
		workspace.Send(sel("activateFileViewerSelectingURLs:"), urls)
	})
	return nil
}

// SavePanel asks for a destination and returns it when the user confirms.
func SavePanel(defaultName, title string) (string, bool) {
	var path string
	onMain(func() {
		suppressPanelBlur.Store(true)
		defer suppressPanelBlur.Store(false)
		app := objc.ID(class("NSApplication")).Send(sel("sharedApplication"))
		window := objc.Send[objc.ID](app, sel("keyWindow"))
		if window == 0 {
			for _, p := range panelsByView {
				window = objc.Send[objc.ID](p.view, sel("window"))
				if window != 0 {
					break
				}
			}
		}
		app.Send(sel("activateIgnoringOtherApps:"), true)
		if window != 0 {
			window.Send(sel("makeKeyAndOrderFront:"), objc.ID(0))
		}
		panel := objc.ID(class("NSSavePanel")).Send(sel("savePanel"))
		panel.Send(sel("retain"))
		defer panel.Send(sel("release"))
		panel.Send(sel("setTitle:"), nsString(title))
		panel.Send(sel("setNameFieldStringValue:"), nsString(defaultName))
		panel.Send(sel("setCanCreateDirectories:"), true)
		panel.Send(sel("setLevel:"), int64(1000))
		panel.Send(sel("makeKeyAndOrderFront:"), objc.ID(0))
		if panel.Send(sel("runModal")) != 1 {
			return
		}
		url := objc.Send[objc.ID](panel, sel("URL"))
		if url != 0 {
			path = goString(objc.Send[objc.ID](url, sel("path")))
		}
	})
	return path, path != ""
}

// OpenBackupPanel asks for an .ecopastebak file.
func OpenBackupPanel(title string) (string, bool) {
	var path string
	onMain(func() {
		suppressPanelBlur.Store(true)
		defer suppressPanelBlur.Store(false)
		app := objc.ID(class("NSApplication")).Send(sel("sharedApplication"))
		window := objc.Send[objc.ID](app, sel("keyWindow"))
		if window == 0 {
			for _, p := range panelsByView {
				window = objc.Send[objc.ID](p.view, sel("window"))
				if window != 0 {
					break
				}
			}
		}
		app.Send(sel("activateIgnoringOtherApps:"), true)
		if window != 0 {
			window.Send(sel("makeKeyAndOrderFront:"), objc.ID(0))
		}
		panel := objc.ID(class("NSOpenPanel")).Send(sel("openPanel"))
		panel.Send(sel("retain"))
		defer panel.Send(sel("release"))
		panel.Send(sel("setTitle:"), nsString(title))
		panel.Send(sel("setCanChooseFiles:"), true)
		panel.Send(sel("setCanChooseDirectories:"), false)
		panel.Send(sel("setAllowsMultipleSelection:"), false)
		extensions := objc.ID(class("NSArray")).Send(sel("arrayWithObject:"), nsString("ecopastebak"))
		panel.Send(sel("setAllowedFileTypes:"), extensions)
		panel.Send(sel("setLevel:"), int64(1000))
		panel.Send(sel("makeKeyAndOrderFront:"), objc.ID(0))
		if panel.Send(sel("runModal")) != 1 {
			return
		}
		url := objc.Send[objc.ID](panel, sel("URL"))
		if url != 0 {
			path = goString(objc.Send[objc.ID](url, sel("path")))
		}
	})
	return path, path != ""
}

// onMain runs AppKit work on the application's main thread.
func onMain(fn func()) {
	if objc.Send[bool](objc.ID(class("NSThread")), sel("isMainThread")) {
		fn()
		return
	}
	done := make(chan struct{})
	block := objc.NewBlock(func(objc.Block) {
		defer close(done)
		fn()
	})
	objc.ID(class("NSOperationQueue")).Send(sel("mainQueue")).
		Send(sel("addOperationWithBlock:"), block)
	block.Release()
	<-done
}
