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

func TestShouldHideAfterDrag(t *testing.T) {
	cases := []struct {
		name       string
		visible    bool
		accepted   bool
		key        bool
		active     bool
		pinned     bool
		onboarding bool
		suppressed bool
		want       bool
	}{
		{name: "accepted external drop", visible: true, accepted: true, key: true, active: true, want: true},
		{name: "inactive app", visible: true, key: true, want: true},
		{name: "cancelled drag", visible: true, key: true, active: true},
		{name: "window lost key", visible: true, active: true, want: true},
		{name: "hidden panel"},
		{name: "pinned panel", visible: true, accepted: true, key: true, active: true, pinned: true},
		{name: "onboarding panel", visible: true, accepted: true, key: true, active: true, onboarding: true},
		{name: "suppressed panel", visible: true, accepted: true, key: true, active: true, suppressed: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldHideAfterDrag(tc.visible, tc.accepted, tc.key, tc.active, tc.pinned, tc.onboarding, tc.suppressed)
			if got != tc.want {
				t.Errorf("shouldHideAfterDrag() = %v, want %v", got, tc.want)
			}
		})
	}
}
