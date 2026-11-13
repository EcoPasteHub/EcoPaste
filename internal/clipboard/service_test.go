package clipboard

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/EcoPasteHub/EcoPaste/internal/database"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
	"github.com/EcoPasteHub/EcoPaste/internal/storage"
)

func testService(t *testing.T, cfg settings.Clipboard) (*Service, *database.DB, *storage.ImageStore) {
	t.Helper()
	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "db", "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	images := storage.NewImageStore(filepath.Join(dir, "resources"))
	return &Service{
		DB:       db,
		Images:   images,
		Settings: func() settings.Clipboard { return cfg },
	}, db, images
}

func TestCaptureStoresTextAndDeduplicates(t *testing.T) {
	cfg := settings.Default().Clipboard
	svc, db, _ := testService(t, cfg)
	ctx := context.Background()
	source := SourceApp{ID: "com.example.app", Name: "Example"}

	id, changed, err := svc.Capture(ctx, Payload{Text: "hello world"}, source)
	if err != nil {
		t.Fatal(err)
	}
	if !changed || id == "" {
		t.Fatalf("changed = %v id = %q", changed, id)
	}
	item, err := db.FindByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if item.Kind != database.KindText || item.Content != "hello world" {
		t.Errorf("item = %+v", item)
	}
	if item.SourceAppID == nil || *item.SourceAppID != source.ID {
		t.Errorf("source = %v", item.SourceAppID)
	}

	id, _, err = svc.Capture(ctx, Payload{Text: "hello world"}, source)
	if err != nil {
		t.Fatal(err)
	}
	page, err := db.Page(ctx, database.Query{})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("duplicate created another row: %+v", page)
	}
	if again, _ := db.FindByID(ctx, id); again.UseCount != 2 {
		t.Errorf("use count = %d, want 2", again.UseCount)
	}
}

func TestGuardSuppressesOwnWrite(t *testing.T) {
	cfg := settings.Default().Clipboard
	svc, _, _ := testService(t, cfg)
	svc.Guard = NewGuard(2 * time.Second)
	svc.Guard.Mark(database.Hash(database.KindText, "echo"))
	if _, changed, err := svc.Capture(context.Background(), Payload{Text: "echo"}, SourceApp{}); err != nil || changed {
		t.Fatalf("changed = %v err = %v", changed, err)
	}
}

func TestIgnoredAppIsNotCaptured(t *testing.T) {
	cfg := settings.Default().Clipboard
	cfg.Filters.ExcludedAppIDs = []string{"com.apple.Passwords"}
	svc, db, _ := testService(t, cfg)
	if _, changed, err := svc.Capture(context.Background(), Payload{Text: "secret"}, SourceApp{ID: "com.apple.Passwords"}); err != nil || changed {
		t.Fatalf("changed = %v err = %v", changed, err)
	}
	page, err := db.Page(context.Background(), database.Query{})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 0 {
		t.Errorf("ignored app created %d rows", page.Total)
	}
}

func TestDefaultExcludedAppsAreNotCaptured(t *testing.T) {
	cfg := settings.Default().Clipboard
	svc, db, _ := testService(t, cfg)
	ctx := context.Background()
	for _, id := range cfg.Filters.ExcludedAppIDs {
		if _, changed, err := svc.Capture(ctx, Payload{Text: "secret from " + id}, SourceApp{ID: id}); err != nil || changed {
			t.Errorf("%s: changed = %v err = %v", id, changed, err)
		}
	}
	if _, changed, err := svc.Capture(ctx, Payload{Text: "normal"}, SourceApp{ID: "com.example.editor"}); err != nil || !changed {
		t.Fatalf("normal app: changed = %v err = %v", changed, err)
	}
	page, err := db.Page(ctx, database.Query{})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 {
		t.Errorf("recorded %d rows, want only the normal app", page.Total)
	}
}

func TestSecretCapturePolicy(t *testing.T) {
	secret := "sk-" + strings.Repeat("z", 32)
	ctx := context.Background()

	cfg := settings.Default().Clipboard
	cfg.Sensitive.CollectSecrets = false
	svc, db, _ := testService(t, cfg)
	if _, changed, err := svc.Capture(ctx, Payload{Text: secret}, SourceApp{}); err != nil || changed {
		t.Fatalf("excluded secret: changed = %v err = %v", changed, err)
	}
	page, err := db.Page(ctx, database.Query{})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 0 {
		t.Errorf("excluded secret created %d rows", page.Total)
	}

	cfg.Sensitive.CollectSecrets = true
	svc.Settings = func() settings.Clipboard { return cfg }
	id, changed, err := svc.Capture(ctx, Payload{Text: secret}, SourceApp{})
	if err != nil || !changed {
		t.Fatalf("collected secret: changed = %v err = %v", changed, err)
	}
	item, err := db.FindByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if !item.IsSensitive {
		t.Error("secret was not marked sensitive")
	}
}

func TestCaptureOrderAndTypeSwitches(t *testing.T) {
	cfg := settings.Default().Clipboard
	cfg.Capture.Order = []settings.CaptureKind{settings.CaptureRTF, settings.CaptureHTML, settings.CaptureText}
	svc, db, _ := testService(t, cfg)
	ctx := context.Background()
	payload := Payload{Text: "plain", HTML: "<b>html</b>", RTF: "{\\rtf1 rtf}"}
	id, _, err := svc.Capture(ctx, payload, SourceApp{})
	if err != nil {
		t.Fatal(err)
	}
	item, _ := db.FindByID(ctx, id)
	if item.SubKind == nil || *item.SubKind != database.SubRTF {
		t.Fatalf("sub kind = %v, want rtf", item.SubKind)
	}
	if item.SearchText == nil || *item.SearchText != "plain" {
		t.Errorf("search text = %v", item.SearchText)
	}

	cfg.Capture.RTF = false
	svc.Settings = func() settings.Clipboard { return cfg }
	id, _, err = svc.Capture(ctx, payload, SourceApp{})
	if err != nil {
		t.Fatal(err)
	}
	item, _ = db.FindByID(ctx, id)
	if item.SubKind == nil || *item.SubKind != database.SubHTML {
		t.Fatalf("sub kind = %v, want html", item.SubKind)
	}
}

func TestCaptureImage(t *testing.T) {
	cfg := settings.Default().Clipboard
	svc, db, images := testService(t, cfg)
	ctx := context.Background()
	img := image.NewRGBA(image.Rect(0, 0, 2, 1))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	id, changed, err := svc.Capture(ctx, Payload{PNG: buf.Bytes()}, SourceApp{})
	if err != nil || !changed {
		t.Fatalf("changed = %v err = %v", changed, err)
	}
	item, err := db.FindByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if item.Kind != database.KindImage || item.Width == nil || *item.Width != 2 {
		t.Errorf("item = %+v", item)
	}
	if _, err := os.Stat(images.OriginPath(item.Content)); err != nil {
		t.Errorf("image not stored: %v", err)
	}
}

func TestCaptureFiles(t *testing.T) {
	cfg := settings.Default().Clipboard
	svc, db, _ := testService(t, cfg)
	ctx := context.Background()
	dir := t.TempDir()
	file := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	id, changed, err := svc.Capture(ctx, Payload{Files: []string{file, dir}}, SourceApp{})
	if err != nil || !changed {
		t.Fatalf("changed = %v err = %v", changed, err)
	}
	item, _ := db.FindByID(ctx, id)
	if item.Kind != database.KindFiles {
		t.Fatalf("kind = %q", item.Kind)
	}
	if item.FileTypes == nil || *item.FileTypes != "f,d" {
		t.Errorf("file types = %v", item.FileTypes)
	}
	if got := DecodeFiles(item.Content); len(got) != 2 {
		t.Errorf("paths = %v", got)
	}
}

func TestTextSizeLimit(t *testing.T) {
	cfg := settings.Default().Clipboard
	cfg.Capture.MaxTextMB = 1
	svc, _, _ := testService(t, cfg)
	big := strings.Repeat("x", 1024*1024+1)
	if _, changed, err := svc.Capture(context.Background(), Payload{Text: big}, SourceApp{}); err != nil || changed {
		t.Fatalf("changed = %v err = %v", changed, err)
	}
}

func TestImageSizeLimit(t *testing.T) {
	cfg := settings.Default().Clipboard
	cfg.Capture.MaxImageMB = 1
	svc, _, _ := testService(t, cfg)
	big := make([]byte, 1024*1024+1)
	if _, changed, err := svc.Capture(context.Background(), Payload{PNG: big}, SourceApp{}); err != nil || changed {
		t.Fatalf("changed = %v err = %v", changed, err)
	}
}

type fakeSource struct {
	count   atomic.Int64
	payload Payload
}

func (f *fakeSource) ChangeCount() int { return int(f.count.Load()) }

func (f *fakeSource) Read() (Payload, SourceApp, error) {
	return f.payload, SourceApp{ID: "fake", Name: "Fake"}, nil
}

func TestWatcherReportsChangesOnly(t *testing.T) {
	source := &fakeSource{payload: Payload{Text: "new"}}
	captured := make(chan string, 4)
	w := &Watcher{
		Source:   source,
		Interval: 10 * time.Millisecond,
		Capture: func(_ context.Context, payload Payload, _ SourceApp) {
			captured <- payload.Text
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go w.Run(ctx)

	time.Sleep(30 * time.Millisecond)
	select {
	case text := <-captured:
		t.Fatalf("watcher captured the initial pasteboard: %q", text)
	default:
	}
	source.count.Add(1)
	select {
	case text := <-captured:
		if text != "new" {
			t.Fatalf("text = %q", text)
		}
	case <-time.After(time.Second):
		t.Fatal("watcher did not report a change")
	}
}

type fakeWriter struct {
	payload WritePayload
}

func (f *fakeWriter) Write(p WritePayload) error {
	f.payload = p
	return nil
}

func TestBuildWritePreservesRepresentations(t *testing.T) {
	cfg := settings.Default().Clipboard
	svc, _, _ := testService(t, cfg)
	plain := "plain"
	html := database.SubHTML
	text := &database.Item{Kind: database.KindText, Content: "<b>plain</b>", SearchText: &plain, SubKind: &html}

	got, err := svc.BuildWrite(text, false)
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "plain" || got.HTML != "<b>plain</b>" || got.RTF != "" {
		t.Errorf("write = %+v", got)
	}
	got, err = svc.BuildWrite(text, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "plain" || got.HTML != "" {
		t.Errorf("plain write = %+v", got)
	}

	files := "/a\n/b"
	got, err = svc.BuildWrite(&database.Item{Kind: database.KindFiles, Content: files}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Files) != 2 || got.Files[1] != "/b" {
		t.Errorf("files = %v", got.Files)
	}
	got, err = svc.BuildWrite(&database.Item{Kind: database.KindFiles, Content: files}, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != files || len(got.Files) != 0 {
		t.Errorf("files as text = %+v", got)
	}
}

func TestWriteItemSuppressesItsOwnCapture(t *testing.T) {
	cfg := settings.Default().Clipboard
	svc, db, _ := testService(t, cfg)
	svc.Guard = NewGuard(2 * time.Second)
	writer := &fakeWriter{}
	svc.Writer = writer
	ctx := context.Background()
	id, _, err := svc.Capture(ctx, Payload{Text: "round trip"}, SourceApp{})
	if err != nil {
		t.Fatal(err)
	}
	item, err := db.FindByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.WriteItem(ctx, item, false); err != nil {
		t.Fatal(err)
	}
	if writer.payload.Text != "round trip" {
		t.Errorf("write = %+v", writer.payload)
	}
	if _, changed, err := svc.Capture(ctx, Payload{Text: writer.payload.Text}, SourceApp{}); err != nil || changed {
		t.Fatalf("own write was captured: changed = %v err = %v", changed, err)
	}
}

func TestWriteRichTextSuppressesHTMLAndPlainFallback(t *testing.T) {
	cfg := settings.Default().Clipboard
	svc, db, _ := testService(t, cfg)
	svc.Guard = NewGuard(2 * time.Second)
	svc.Writer = &fakeWriter{}
	ctx := context.Background()
	id, _, err := svc.Capture(ctx, Payload{
		Text: "rich text",
		HTML: "<b>rich text</b>",
	}, SourceApp{})
	if err != nil {
		t.Fatal(err)
	}
	item, err := db.FindByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.WriteItem(ctx, item, false); err != nil {
		t.Fatal(err)
	}
	if _, changed, err := svc.Capture(ctx, Payload{Text: "rich text"}, SourceApp{}); err != nil || changed {
		t.Fatalf("plain fallback was captured: changed = %v err = %v", changed, err)
	}
	if _, changed, err := svc.Capture(ctx, Payload{
		Text: "rich text",
		HTML: "<b>rich text</b>",
	}, SourceApp{}); err != nil || changed {
		t.Fatalf("HTML representation was captured: changed = %v err = %v", changed, err)
	}
}

func TestWriteFilesAsTextSuppressesFallbackCapture(t *testing.T) {
	cfg := settings.Default().Clipboard
	svc, db, _ := testService(t, cfg)
	svc.Guard = NewGuard(2 * time.Second)
	svc.Writer = &fakeWriter{}
	ctx := context.Background()
	id, _, err := svc.Capture(ctx, Payload{Files: []string{"/tmp/a", "/tmp/b"}}, SourceApp{})
	if err != nil {
		t.Fatal(err)
	}
	item, err := db.FindByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.WriteItem(ctx, item, true); err != nil {
		t.Fatal(err)
	}
	if _, changed, err := svc.Capture(ctx, Payload{Text: "/tmp/a\n/tmp/b"}, SourceApp{}); err != nil || changed {
		t.Fatalf("path text was captured: changed = %v err = %v", changed, err)
	}
}
