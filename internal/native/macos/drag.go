//go:build darwin

package macos

import "github.com/ebitengine/purego/objc"

// DragPayload is what one history row puts on a dragging pasteboard.
// Files are written as separate pasteboard items so Finder receives a
// multi-file drop; text keeps its rich representations.
type DragPayload struct {
	Text      string
	HTML      string
	RTF       string
	ImageData []byte
	Files     []string
}

// pasteboardWriter turns one row into the object NSTableView asks for
// when the user starts dragging. The payload is read synchronously
// because AppKit builds the dragging pasteboard at that moment.
func (p *Panel) pasteboardWriter(row HistoryRow) objc.ID {
	p.dragFiles = nil
	if p.drag == nil || row.ID == "" {
		return 0
	}
	payload, ok := p.drag(row.ID)
	if !ok {
		return 0
	}
	switch {
	case len(payload.Files) > 0:
		p.dragFiles = payload.Files
		return objc.ID(class("NSURL")).Send(sel("fileURLWithPath:"), nsString(payload.Files[0]))
	case len(payload.ImageData) > 0:
		item := objc.ID(class("NSPasteboardItem")).Send(sel("new"))
		item.Send(sel("setData:forType:"), nsData(payload.ImageData), nsString(TypePNG))
		return item.Send(sel("autorelease"))
	case payload.Text != "" || payload.HTML != "" || payload.RTF != "":
		item := objc.ID(class("NSPasteboardItem")).Send(sel("new"))
		if payload.Text != "" {
			item.Send(sel("setString:forType:"), nsString(payload.Text), nsString(TypeString))
		}
		if payload.HTML != "" {
			item.Send(sel("setString:forType:"), nsString(payload.HTML), nsString(TypeHTML))
		}
		if payload.RTF != "" {
			item.Send(sel("setString:forType:"), nsString(payload.RTF), nsString(TypeRTF))
		}
		return item.Send(sel("autorelease"))
	}
	return 0
}

// appendDragFiles adds the remaining files of a multi-file row once the
// dragging session exists. The table view asks for only one writer per
// row, so extra file URLs have to be appended to the session pasteboard.
func (p *Panel) appendDragFiles(session objc.ID) {
	files := p.dragFiles
	p.dragFiles = nil
	if len(files) <= 1 {
		return
	}
	urls := objc.ID(class("NSMutableArray")).Send(sel("array"))
	for _, path := range files[1:] {
		urls.Send(sel("addObject:"), objc.ID(class("NSURL")).Send(sel("fileURLWithPath:"), nsString(path)))
	}
	session.Send(sel("draggingPasteboard")).Send(sel("writeObjects:"), urls)
}
