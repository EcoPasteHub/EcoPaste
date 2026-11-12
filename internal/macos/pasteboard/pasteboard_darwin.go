//go:build darwin

package pasteboard

import (
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"

	"github.com/EcoPasteHub/EcoPaste/internal/clipboard"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
)

const (
	typeText      = "public.utf8-plain-text"
	typeHTML      = "public.html"
	typeRTF       = "public.rtf"
	typePNG       = "public.png"
	typeTIFF      = "public.tiff"
	typeFileNames = "NSFilenamesPboardType"
)

var (
	loadOnce sync.Once
	loadErr  error
	sels     sync.Map
)

// Pasteboard wraps an NSPasteboard and keeps all Objective-C calls
// inside an autorelease pool.
type Pasteboard struct {
	pb objc.ID
}

// App identifies the frontmost application at capture time.
type App struct {
	BundleID string
	Name     string
}

// General returns the system general pasteboard.
func General() (*Pasteboard, error) {
	ensureAppKit()
	if loadErr != nil {
		return nil, loadErr
	}
	class := objc.GetClass("NSPasteboard")
	if class == 0 {
		return nil, errors.New("NSPasteboard is unavailable")
	}
	return &Pasteboard{pb: objc.ID(class).Send(sel("generalPasteboard"))}, nil
}

// Named returns a named pasteboard, useful for isolated round-trip tests.
func Named(name string) (*Pasteboard, error) {
	ensureAppKit()
	if loadErr != nil {
		return nil, loadErr
	}
	class := objc.GetClass("NSPasteboard")
	if class == 0 {
		return nil, errors.New("NSPasteboard is unavailable")
	}
	pb := objc.ID(class).Send(sel("pasteboardWithName:"), nsString(name))
	if pb == 0 {
		return nil, fmt.Errorf("create pasteboard %q", name)
	}
	return &Pasteboard{pb: pb}, nil
}

// ChangeCount identifies pasteboard revisions.
func (p *Pasteboard) ChangeCount() int64 {
	var count int64
	withPool(func() {
		count = objc.Send[int64](p.pb, sel("changeCount"))
	})
	return count
}

// Read returns the first enabled representation according to capture order.
func (p *Pasteboard) Read(capture settings.Capture) (clipboard.Payload, bool, error) {
	var payload clipboard.Payload
	var ok bool
	var err error
	withPool(func() {
		for _, kind := range orderedKinds(capture) {
			if !captureEnabled(capture, kind) {
				continue
			}
			switch kind {
			case "files":
				if files := p.files(); len(files) > 0 {
					payload = clipboard.Payload{Kind: "files", Files: files}
					ok = true
					return
				}
			case "image":
				if image, found := p.image(); found {
					payload = clipboard.Payload{Kind: "image", Image: image}
					ok = true
					return
				}
			case "html":
				if text, found := p.text(); found && text.HTML != "" {
					payload = clipboard.Payload{Kind: "text", Text: text}
					ok = true
					return
				}
			case "rtf":
				if text, found := p.text(); found && text.RTF != "" {
					payload = clipboard.Payload{Kind: "text", Text: text}
					ok = true
					return
				}
			case "text":
				if text, found := p.text(); found {
					payload = clipboard.Payload{Kind: "text", Text: text}
					ok = true
					return
				}
			}
		}
	})
	return payload, ok, err
}

// WriteText writes plain text and, when requested, its rich representation.
func (p *Pasteboard) WriteText(plain, html, rtf string, forcePlain bool) {
	withPool(func() {
		p.pb.Send(sel("clearContents"))
		if !forcePlain && html != "" {
			p.pb.Send(sel("setString:forType:"), nsString(html), nsString(typeHTML))
		}
		if !forcePlain && rtf != "" {
			p.pb.Send(sel("setString:forType:"), nsString(rtf), nsString(typeRTF))
		}
		p.pb.Send(sel("setString:forType:"), nsString(plain), nsString(typeText))
	})
}

// WriteImage writes PNG and TIFF representations.
func (p *Pasteboard) WriteImage(png []byte) error {
	if len(png) == 0 {
		return errors.New("empty PNG")
	}
	withPool(func() {
		p.pb.Send(sel("clearContents"))
		p.pb.Send(sel("setData:forType:"), nsData(png), nsString(typePNG))
		img := objc.ID(objc.GetClass("NSImage")).Send(sel("alloc")).Send(sel("initWithData:"), nsData(png))
		if img != 0 {
			if tiff := img.Send(sel("TIFFRepresentation")); tiff != 0 {
				p.pb.Send(sel("setData:forType:"), tiff, nsString(typeTIFF))
			}
			img.Send(sel("release"))
		}
	})
	return nil
}

// WriteFiles writes file URLs for Finder and other file-aware apps.
func (p *Pasteboard) WriteFiles(paths []string) error {
	if len(paths) == 0 {
		return errors.New("no files to write")
	}
	withPool(func() {
		p.pb.Send(sel("clearContents"))
		objects := objc.ID(objc.GetClass("NSMutableArray")).Send(sel("array"))
		names := objc.ID(objc.GetClass("NSMutableArray")).Send(sel("array"))
		for _, path := range paths {
			u := objc.ID(objc.GetClass("NSURL")).Send(sel("fileURLWithPath:"), nsString(path))
			objects.Send(sel("addObject:"), u)
			names.Send(sel("addObject:"), nsString(path))
		}
		p.pb.Send(sel("writeObjects:"), objects)
		p.pb.Send(sel("setPropertyList:forType:"), names, nsString(typeFileNames))
	})
	return nil
}

// Frontmost returns the application that was active when called.
func Frontmost() App {
	ensureAppKit()
	if loadErr != nil {
		return App{}
	}
	var app App
	withPool(func() {
		ws := objc.ID(objc.GetClass("NSWorkspace")).Send(sel("sharedWorkspace"))
		current := ws.Send(sel("frontmostApplication"))
		if current == 0 {
			return
		}
		app.BundleID = goString(current.Send(sel("bundleIdentifier")))
		app.Name = goString(current.Send(sel("localizedName")))
	})
	return app
}

func (p *Pasteboard) text() (clipboard.Text, bool) {
	var text clipboard.Text
	found := false
	if p.hasType(typeText) {
		text.Plain = p.stringForType(typeText)
		found = true
	}
	if p.hasType(typeHTML) {
		text.HTML = p.stringForType(typeHTML)
		found = true
	}
	if p.hasType(typeRTF) {
		text.RTF = p.stringForType(typeRTF)
		found = true
	}
	return text, found
}

func (p *Pasteboard) image() (clipboard.Image, bool) {
	if p.hasType(typePNG) {
		data := p.dataForType(typePNG)
		if image, err := clipboard.DecodeImage(data); err == nil {
			return image, true
		}
	}
	if p.hasType(typeTIFF) {
		data := p.dataForType(typeTIFF)
		if png, err := tiffToPNG(data); err == nil {
			if image, err := clipboard.DecodeImage(png); err == nil {
				return image, true
			}
		}
	}
	return clipboard.Image{}, false
}

func (p *Pasteboard) files() []string {
	classes := objc.ID(objc.GetClass("NSArray")).Send(sel("arrayWithObject:"), objc.ID(objc.GetClass("NSURL")))
	key := nsString("NSPasteboardURLReadingFileURLsOnlyKey")
	yes := objc.ID(objc.GetClass("NSNumber")).Send(sel("numberWithBool:"), true)
	opts := objc.ID(objc.GetClass("NSDictionary")).Send(sel("dictionaryWithObject:forKey:"), yes, key)
	list := p.pb.Send(sel("readObjectsForClasses:options:"), classes, opts)
	if list == 0 {
		return nil
	}
	count := objc.Send[uint](list, sel("count"))
	files := make([]string, 0, count)
	for index := uint(0); index < count; index++ {
		u := list.Send(sel("objectAtIndex:"), index)
		if path := goString(u.Send(sel("path"))); path != "" {
			files = append(files, path)
		}
	}
	return files
}

func (p *Pasteboard) hasType(name string) bool {
	return goString(p.pb.Send(sel("availableTypeFromArray:"),
		objc.ID(objc.GetClass("NSArray")).Send(sel("arrayWithObject:"), nsString(name)))) != ""
}

func (p *Pasteboard) stringForType(name string) string {
	return goString(p.pb.Send(sel("stringForType:"), nsString(name)))
}

func (p *Pasteboard) dataForType(name string) []byte {
	return nsBytes(p.pb.Send(sel("dataForType:"), nsString(name)))
}

func tiffToPNG(data []byte) ([]byte, error) {
	img := objc.ID(objc.GetClass("NSImage")).Send(sel("alloc")).Send(sel("initWithData:"), nsData(data))
	if img == 0 {
		return nil, errors.New("decode TIFF")
	}
	defer img.Send(sel("release"))
	tiff := img.Send(sel("TIFFRepresentation"))
	rep := objc.ID(objc.GetClass("NSBitmapImageRep")).Send(sel("imageRepWithData:"), tiff)
	if rep == 0 {
		return nil, errors.New("create bitmap representation")
	}
	props := objc.ID(objc.GetClass("NSDictionary")).Send(sel("dictionary"))
	png := rep.Send(sel("representationUsingType:properties:"), 4, props)
	if png == 0 {
		return nil, errors.New("encode PNG")
	}
	return nsBytes(png), nil
}

func nsString(value string) objc.ID {
	bytes := append([]byte(value), 0)
	return objc.ID(objc.GetClass("NSString")).Send(sel("stringWithUTF8String:"), unsafe.Pointer(&bytes[0]))
}

func goString(value objc.ID) string {
	if value == 0 {
		return ""
	}
	ptr := objc.Send[*byte](value, sel("UTF8String"))
	if ptr == nil {
		return ""
	}
	length := 0
	for *(*byte)(unsafe.Add(unsafe.Pointer(ptr), length)) != 0 {
		length++
	}
	return string(unsafe.Slice(ptr, length))
}

func nsData(data []byte) objc.ID {
	if len(data) == 0 {
		return objc.ID(objc.GetClass("NSData")).Send(sel("data"))
	}
	return objc.ID(objc.GetClass("NSData")).Send(sel("dataWithBytes:length:"),
		unsafe.Pointer(&data[0]), uint(len(data)))
}

func nsBytes(data objc.ID) []byte {
	if data == 0 {
		return nil
	}
	length := objc.Send[uint](data, sel("length"))
	if length == 0 {
		return nil
	}
	out := make([]byte, length)
	copy(out, unsafe.Slice(objc.Send[*byte](data, sel("bytes")), length))
	return out
}

func withPool(fn func()) {
	pool := objc.ID(objc.GetClass("NSAutoreleasePool")).Send(sel("alloc")).Send(sel("init"))
	defer pool.Send(sel("drain"))
	fn()
}

func sel(name string) objc.SEL {
	if cached, ok := sels.Load(name); ok {
		return cached.(objc.SEL)
	}
	value := objc.RegisterName(name)
	sels.Store(name, value)
	return value
}

func ensureAppKit() {
	loadOnce.Do(func() {
		_, loadErr = purego.Dlopen("/System/Library/Frameworks/AppKit.framework/AppKit",
			purego.RTLD_GLOBAL|purego.RTLD_NOW)
	})
}

func orderedKinds(capture settings.Capture) []string {
	seen := map[string]bool{}
	out := make([]string, 0, 5)
	for _, kind := range append(capture.Order, "files", "image", "html", "rtf", "text") {
		if !seen[kind] {
			seen[kind] = true
			out = append(out, kind)
		}
	}
	return out
}

func captureEnabled(capture settings.Capture, kind string) bool {
	switch kind {
	case "files":
		return capture.Files
	case "image":
		return capture.Image
	case "html":
		return capture.HTML
	case "rtf":
		return capture.RTF
	default:
		return capture.Text
	}
}
