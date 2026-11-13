//go:build darwin

package macos

import (
	"fmt"
	"net/url"

	"github.com/ebitengine/purego/objc"
)

// Pasteboard types EcoPaste reads and writes.
const (
	TypeString  = "public.utf8-plain-text"
	TypeHTML    = "public.html"
	TypeRTF     = "public.rtf"
	TypePNG     = "public.png"
	TypeTIFF    = "public.tiff"
	TypeFileURL = "public.file-url"
)

// Reading is everything one pasteboard snapshot can provide.
type Reading struct {
	Text  string
	HTML  string
	RTF   string
	PNG   []byte
	Files []string
}

// AppInfo identifies the application that put content on the pasteboard.
type AppInfo struct {
	BundleID string
	Name     string
}

func generalPasteboard() objc.ID {
	return objc.ID(class("NSPasteboard")).Send(sel("generalPasteboard"))
}

// PasteboardChangeCount returns the system pasteboard's change counter.
func PasteboardChangeCount() int {
	return objc.Send[int](generalPasteboard(), sel("changeCount"))
}

// ReadPasteboard reads every representation EcoPaste can store.
func ReadPasteboard() (Reading, error) {
	p := generalPasteboard()
	var r Reading
	r.Text = boardString(p, TypeString)
	r.HTML = boardString(p, TypeHTML)
	r.RTF = boardString(p, TypeRTF)
	r.PNG = boardData(p, TypePNG)
	if len(r.PNG) == 0 {
		r.PNG = tiffToPNG(boardData(p, TypeTIFF))
	}
	r.Files = boardFiles(p)
	if r.Text == "" && r.HTML == "" && r.RTF == "" && len(r.PNG) == 0 && len(r.Files) == 0 {
		return Reading{}, nil
	}
	return r, nil
}

// FrontmostApp reports the application that was active when the
// pasteboard changed, which is normally the copy's source.
func FrontmostApp() AppInfo {
	workspace := objc.ID(class("NSWorkspace")).Send(sel("sharedWorkspace"))
	app := workspace.Send(sel("frontmostApplication"))
	if app == 0 {
		return AppInfo{}
	}
	return AppInfo{
		BundleID: goString(app.Send(sel("bundleIdentifier"))),
		Name:     goString(app.Send(sel("localizedName"))),
	}
}

// WriteContent is one pasteboard write. HTML and RTF keep the plain text
// fallback in the same item so applications that do not read rich text
// still paste something.
type WriteContent struct {
	Text     string
	HTML     string
	RTF      string
	ImagePNG []byte
	Files    []string
}

// WritePasteboard replaces the pasteboard with content.
func WritePasteboard(content WriteContent) error {
	p := generalPasteboard()
	p.Send(sel("clearContents"))

	if len(content.Files) > 0 {
		urls := objc.ID(class("NSMutableArray")).Send(sel("array"))
		for _, path := range content.Files {
			u := objc.ID(class("NSURL")).Send(sel("fileURLWithPath:"), nsString(path))
			urls.Send(sel("addObject:"), u)
		}
		if p.Send(sel("writeObjects:"), urls) == 0 {
			return fmt.Errorf("write file URLs to pasteboard")
		}
		return nil
	}

	if len(content.ImagePNG) > 0 {
		p.Send(sel("setData:forType:"), nsData(content.ImagePNG), nsString(TypePNG))
		return nil
	}

	if content.Text == "" && content.HTML == "" && content.RTF == "" {
		return nil
	}
	if content.HTML == "" && content.RTF == "" {
		p.Send(sel("setString:forType:"), nsString(content.Text), nsString(TypeString))
		return nil
	}

	item := objc.ID(class("NSPasteboardItem")).Send(sel("new"))
	defer item.Send(sel("release"))
	if content.Text != "" {
		item.Send(sel("setString:forType:"), nsString(content.Text), nsString(TypeString))
	}
	if content.HTML != "" {
		item.Send(sel("setString:forType:"), nsString(content.HTML), nsString(TypeHTML))
	}
	if content.RTF != "" {
		item.Send(sel("setString:forType:"), nsString(content.RTF), nsString(TypeRTF))
	}
	items := objc.ID(class("NSMutableArray")).Send(sel("array"))
	items.Send(sel("addObject:"), item)
	if p.Send(sel("writeObjects:"), items) == 0 {
		return fmt.Errorf("write text representations to pasteboard")
	}
	return nil
}

func boardString(p objc.ID, kind string) string {
	value := p.Send(sel("stringForType:"), nsString(kind))
	if value == 0 {
		return ""
	}
	return goString(value)
}

func boardData(p objc.ID, kind string) []byte {
	data := p.Send(sel("dataForType:"), nsString(kind))
	return dataBytes(data)
}

func dataBytes(data objc.ID) []byte {
	if data == 0 {
		return nil
	}
	length := objc.Send[uint](data, sel("length"))
	if length == 0 {
		return nil
	}
	src := objc.Send[*byte](data, sel("bytes"))
	if src == nil {
		return nil
	}
	out := make([]byte, length)
	copy(out, unsafeSlice(src, length))
	return out
}

func boardFiles(p objc.ID) []string {
	items := p.Send(sel("pasteboardItems"))
	if items == 0 {
		return nil
	}
	count := objc.Send[uint](items, sel("count"))
	var paths []string
	for i := uint(0); i < count; i++ {
		item := items.Send(sel("objectAtIndex:"), i)
		if item == 0 {
			continue
		}
		value := item.Send(sel("stringForType:"), nsString(TypeFileURL))
		if value == 0 {
			continue
		}
		u, err := url.Parse(goString(value))
		if err != nil || u.Scheme != "file" || u.Path == "" {
			continue
		}
		paths = append(paths, u.Path)
	}
	return paths
}

func tiffToPNG(tiff []byte) []byte {
	if len(tiff) == 0 {
		return nil
	}
	image := objc.ID(class("NSImage")).Send(sel("alloc"))
	image = image.Send(sel("initWithData:"), nsData(tiff))
	if image == 0 {
		return nil
	}
	defer image.Send(sel("release"))
	rep := objc.ID(class("NSBitmapImageRep")).Send(sel("imageRepWithData:"), image.Send(sel("TIFFRepresentation")))
	if rep == 0 {
		return nil
	}
	properties := objc.ID(class("NSDictionary")).Send(sel("dictionary"))
	data := rep.Send(sel("representationUsingType:properties:"), 4, properties)
	if data == 0 {
		return nil
	}
	length := objc.Send[uint](data, sel("length"))
	src := objc.Send[*byte](data, sel("bytes"))
	out := make([]byte, length)
	copy(out, unsafeSlice(src, length))
	return out
}

func nsData(b []byte) objc.ID {
	if len(b) == 0 {
		return objc.ID(class("NSData")).Send(sel("data"))
	}
	return objc.ID(class("NSData")).Send(sel("dataWithBytes:length:"), &b[0], uint(len(b)))
}
