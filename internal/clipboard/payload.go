package clipboard

// Payload is one pasteboard snapshot.
type Payload struct {
	Text  string
	HTML  string
	RTF   string
	PNG   []byte
	Files []string
}

// SourceApp identifies the application the content came from.
type SourceApp struct {
	ID   string
	Name string
}

// Source is the platform pasteboard the watcher polls.
type Source interface {
	ChangeCount() int
	Read() (Payload, SourceApp, error)
}
