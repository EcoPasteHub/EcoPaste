//go:build darwin

package macos

import (
	"testing"

	"github.com/ebitengine/purego/objc"
)

func TestWriteDragFilesWritesEveryURL(t *testing.T) {
	name := "EcoPasteDragTest"
	pasteboard := objc.ID(class("NSPasteboard")).Send(sel("pasteboardWithName:"), nsString(name))
	if pasteboard == 0 {
		t.Fatal("named pasteboard was not created")
	}
	t.Cleanup(func() {
		pasteboard.Send(sel("releaseGlobally"))
	})

	files := []string{"/tmp/a.txt", "/tmp/b.txt", "/tmp/c.txt"}
	writeDragFiles(pasteboard, files)
	items := pasteboard.Send(sel("pasteboardItems"))
	count := objc.Send[uint](items, sel("count"))
	if items == 0 || count != uint(len(files)) {
		t.Fatalf("drag items = %v (%d), want %d", items, count, len(files))
	}
	for i, path := range files {
		item := items.Send(sel("objectAtIndex:"), uint(i))
		got := goString(item.Send(sel("stringForType:"), nsString(TypeFileURL)))
		want := "file://" + path
		if got != want {
			t.Errorf("item %d = %q, want %q", i, got, want)
		}
	}
}
