package app

import (
	"github.com/EcoPasteHub/EcoPaste/internal/clipboard"
	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
)

// pasteboardSource adapts the AppKit pasteboard to the clipboard
// package's source interface.
type pasteboardSource struct {
	icons *appIconCache
}

func (pasteboardSource) ChangeCount() int { return macos.PasteboardChangeCount() }

func (s pasteboardSource) Read() (clipboard.Payload, clipboard.SourceApp, error) {
	reading, err := macos.ReadPasteboard()
	if err != nil {
		return clipboard.Payload{}, clipboard.SourceApp{}, err
	}
	app := macos.FrontmostApp()
	var iconFile *string
	if s.icons != nil {
		iconFile = s.icons.File(app.BundleID)
	}
	return clipboard.Payload{
		Text:  reading.Text,
		HTML:  reading.HTML,
		RTF:   reading.RTF,
		PNG:   reading.PNG,
		Files: reading.Files,
	}, clipboard.SourceApp{ID: app.BundleID, Name: app.Name, IconFile: iconFile}, nil
}

// pasteboardWriter adapts the clipboard package's write payload to
// AppKit's pasteboard.
type pasteboardWriter struct{}

func (pasteboardWriter) Write(p clipboard.WritePayload) error {
	return macos.WritePasteboard(macos.WriteContent{
		Text:     p.Text,
		HTML:     p.HTML,
		RTF:      p.RTF,
		ImagePNG: p.PNG,
		Files:    p.Files,
	})
}
