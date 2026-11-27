package clipboard

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
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
	if item.UseCount != 1 || item.CreatedAt.IsZero() || !item.UpdatedAt.Equal(item.CreatedAt) {
		t.Errorf("new item timestamps/use count = %+v", item)
	}
	createdAt := item.CreatedAt

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
	} else if !again.CreatedAt.Equal(createdAt) {
		t.Errorf("duplicate capture changed created_at: %v -> %v", createdAt, again.CreatedAt)
	}
}

func TestCaptureDetailedDistinguishesNewAndRepeatedContent(t *testing.T) {
	cfg := settings.Default().Clipboard
	svc, db, _ := testService(t, cfg)
	ctx := context.Background()
	createdAt := time.Now().UTC().Add(-time.Hour)
	seed := &database.Item{
		Kind:      database.KindText,
		Content:   "repeat me",
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
		UseCount:  1,
	}
	id, duplicate, err := db.Upsert(ctx, seed)
	if err != nil || duplicate {
		t.Fatalf("seed upsert duplicate=%t err=%v", duplicate, err)
	}

	gotID, changed, deduplicated, err := svc.CaptureDetailed(ctx, Payload{Text: "repeat me"}, SourceApp{})
	if err != nil || !changed || !deduplicated || gotID != id {
		t.Fatalf("repeat capture id=%q changed=%t duplicate=%t err=%v", gotID, changed, deduplicated, err)
	}
	item, err := db.FindByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if !item.CreatedAt.Equal(createdAt) || !item.UpdatedAt.After(createdAt) || item.UseCount != 2 {
		t.Fatalf("repeat capture fields = %+v", item)
	}

	newID, changed, deduplicated, err := svc.CaptureDetailed(ctx, Payload{Text: "new content"}, SourceApp{})
	if err != nil || !changed || deduplicated || newID == "" {
		t.Fatalf("new capture id=%q changed=%t duplicate=%t err=%v", newID, changed, deduplicated, err)
	}
	newItem, err := db.FindByID(ctx, newID)
	if err != nil {
		t.Fatal(err)
	}
	if newItem.UseCount != 1 || newItem.CreatedAt.IsZero() || !newItem.UpdatedAt.Equal(newItem.CreatedAt) {
		t.Fatalf("new capture fields = %+v", newItem)
	}
}

func TestBlockCaptureAppsAreNotCaptured(t *testing.T) {
	cfg := settings.Default().Clipboard
	cfg.SourceApps.Rules = []settings.SourceAppRule{
		{ID: "com.example.blockCapture", Mode: settings.SourceAppBlockCapture},
		{ID: "com.example.blockBoth", Mode: settings.SourceAppBlockCaptureAndWake},
	}
	svc, db, _ := testService(t, cfg)
	ctx := context.Background()
	for _, id := range []string{"com.example.blockCapture", "com.example.blockBoth"} {
		if _, changed, err := svc.Capture(ctx, Payload{Text: "secret"}, SourceApp{ID: id}); err != nil || changed {
			t.Errorf("%s: changed = %v err = %v", id, changed, err)
		}
	}
	// Blocking wake alone still records the content.
	if _, changed, err := svc.Capture(ctx, Payload{Text: "still captured"}, SourceApp{ID: "com.example.blockWake"}); err != nil || !changed {
		t.Fatalf("block wake app: changed = %v err = %v", changed, err)
	}
	page, err := db.Page(ctx, database.Query{})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 {
		t.Errorf("recorded %d rows, want only the block-wake app", page.Total)
	}
}

func TestDefaultPasswordAppsAreNotCaptured(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("password-manager defaults only apply on macOS")
	}
	cfg := settings.Default().Clipboard
	if len(cfg.SourceApps.Rules) != 0 {
		t.Fatalf("default rules = %+v, want none", cfg.SourceApps.Rules)
	}
	svc, db, _ := testService(t, cfg)
	ctx := context.Background()
	for _, id := range []string{"com.apple.keychainaccess", "com.apple.Passwords"} {
		if _, changed, err := svc.Capture(ctx, Payload{Text: "secret from " + id}, SourceApp{ID: id}); err != nil || changed {
			t.Errorf("%s: changed = %v err = %v", id, changed, err)
		}
	}
	if _, changed, err := svc.Capture(ctx, Payload{Text: "normal"}, SourceApp{ID: "com.example.editor"}); err != nil || !changed {
		t.Fatalf("normal app: changed = %v err = %v", changed, err)
	}
	page, err := db.Page(context.Background(), database.Query{})
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

func TestCaptureOrder(t *testing.T) {
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
	payload     WritePayload
	calls       int
	changeCount int
}

func (f *fakeWriter) Write(p WritePayload) (int, bool, error) {
	f.calls++
	f.payload = p
	f.changeCount++
	return f.changeCount, true, nil
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
	if got.Text != "" || len(got.Files) != 2 || got.Files[0] != "/a" || got.Files[1] != "/b" {
		t.Errorf("plain text option changed files = %+v", got)
	}
}

func TestWriteItemsSuppressTheirOwnEventsAndAllowExternalRepeats(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payload Payload
	}{
		{name: "text", payload: Payload{Text: "round trip"}},
		{name: "rich text", payload: Payload{Text: "rich text", HTML: "<b>rich text</b>"}},
		{name: "files", payload: Payload{Files: []string{"/tmp/a", "/tmp/b"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := settings.Default().Clipboard
			svc, db, _ := testService(t, cfg)
			svc.Guard = NewGuard()
			writer := &fakeWriter{}
			svc.Writer = writer
			ctx := context.Background()
			id, _, err := svc.Capture(ctx, tc.payload, SourceApp{})
			if err != nil {
				t.Fatal(err)
			}
			item, err := db.FindByID(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			before := *item
			source := &pollSource{count: writer.changeCount, payload: tc.payload}
			watcher := &Watcher{
				Source: source,
				Guard:  svc.Guard,
				Capture: func(ctx context.Context, payload Payload, source SourceApp) {
					if _, _, err := svc.Capture(ctx, payload, source); err != nil {
						t.Errorf("capture: %v", err)
					}
				},
			}
			if read, err := watcher.Poll(ctx, false); err != nil || read {
				t.Fatalf("initial poll = %t, %v", read, err)
			}
			if err := svc.WriteItem(ctx, item, false); err != nil {
				t.Fatal(err)
			}
			source.count = writer.changeCount
			source.payload = Payload(writer.payload)
			if read, err := watcher.Poll(ctx, false); err != nil || read {
				t.Fatalf("internal write poll = %t, %v", read, err)
			}
			if read, err := watcher.Poll(ctx, true); err != nil || read {
				t.Fatalf("forced internal write poll = %t, %v", read, err)
			}
			afterWrite, err := db.FindByID(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			if afterWrite.UseCount != before.UseCount || !afterWrite.CreatedAt.Equal(before.CreatedAt) || !afterWrite.UpdatedAt.Equal(before.UpdatedAt) {
				t.Fatalf("internal write changed row: before=%+v after=%+v", before, afterWrite)
			}
			source.count++
			if read, err := watcher.Poll(ctx, false); err != nil || !read {
				t.Fatalf("external repeat poll = %t, %v", read, err)
			}
			afterRepeat, err := db.FindByID(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			if afterRepeat.UseCount != before.UseCount+1 || !afterRepeat.CreatedAt.Equal(before.CreatedAt) || !afterRepeat.UpdatedAt.After(before.UpdatedAt) {
				t.Fatalf("external repeat fields: before=%+v after=%+v", before, afterRepeat)
			}
		})
	}
}

func TestWriteItemHonorsCanceledContext(t *testing.T) {
	writer := &fakeWriter{}
	svc := &Service{Writer: writer}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := svc.WriteItem(ctx, &database.Item{Kind: database.KindText, Content: "text"}, false)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("WriteItem error = %v, want context.Canceled", err)
	}
	if writer.calls != 0 {
		t.Fatalf("writer called %d times after cancellation", writer.calls)
	}
}
