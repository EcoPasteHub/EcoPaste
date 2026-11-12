package clipboard

import (
	"testing"

	"github.com/EcoPasteHub/EcoPaste/internal/domain"
	"github.com/EcoPasteHub/EcoPaste/internal/storage"
)

type fakeWriter struct {
	plain string
	html  string
	rtf   string
	force bool
	files []string
}

func (f *fakeWriter) WriteText(plain, html, rtf string, forcePlain bool) {
	f.plain, f.html, f.rtf, f.force = plain, html, rtf, forcePlain
}
func (f *fakeWriter) WriteImage([]byte) error { return nil }
func (f *fakeWriter) WriteFiles(paths []string) error {
	f.files = paths
	return nil
}

func TestWriteHTMLWritesPlainFallbackAndArmsBothHashes(t *testing.T) {
	guard := NewWritebackGuard()
	writer := &fakeWriter{}
	item := domain.Item{
		Kind:        domain.KindText,
		SubKind:     "html",
		Content:     "<b>hello</b>",
		SearchText:  "hello",
		ContentHash: storage.ContentHash(domain.KindText, "<b>hello</b>"),
	}
	if err := WriteItem(writer, guard, nil, item, false); err != nil {
		t.Fatal(err)
	}
	if writer.html != item.Content || writer.plain != "hello" || writer.force {
		t.Fatalf("writer = %+v", writer)
	}
	if !guard.Consume(storage.ContentHash(domain.KindText, item.Content)) {
		t.Fatal("rich hash not armed")
	}
	if !guard.Consume(storage.ContentHash(domain.KindText, "hello")) {
		t.Fatal("plain hash not armed")
	}
}

func TestWriteFilesAsTextUsesNewlineContent(t *testing.T) {
	guard := NewWritebackGuard()
	writer := &fakeWriter{}
	item := domain.Item{
		Kind:        domain.KindFiles,
		Content:     "/tmp/a\n/tmp/b",
		ContentHash: storage.ContentHash(domain.KindFiles, "/tmp/a\n/tmp/b"),
	}
	if err := WriteItem(writer, guard, nil, item, true); err != nil {
		t.Fatal(err)
	}
	if writer.plain != item.Content || !writer.force {
		t.Fatalf("writer = %+v", writer)
	}
	if !guard.Consume(storage.ContentHash(domain.KindText, item.Content)) {
		t.Fatal("files-as-text hash not armed")
	}
}
