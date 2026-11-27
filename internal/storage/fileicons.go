package storage

import "path/filepath"

// FileIconStore keeps file-type icons under the published resources
// directory. The database key is separate from the content-addressed file
// name so different paths can share one icon.
type FileIconStore struct {
	*pngIconStore
}

// FileIconMaxEdge is the square edge used for cached file icons.
const FileIconMaxEdge = 128

// NewFileIconStore returns the file icon store under a resources directory.
func NewFileIconStore(resources string) *FileIconStore {
	return &FileIconStore{pngIconStore: newPNGIconStore(
		filepath.Join(resources, "file-icons"), "file icon")}
}
