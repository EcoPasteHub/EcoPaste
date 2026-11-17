package app

import "github.com/EcoPasteHub/EcoPaste/internal/native/macos"

// resolveFileReferences turns stored /.file/id= references into real
// paths. Captures made before the pasteboard reader resolved them keep
// the opaque reference, which would otherwise look like a missing file.
func resolveFileReferences(files []string) []string {
	resolved := make([]string, len(files))
	changed := false
	for index, path := range files {
		resolved[index] = macos.ResolveFileReference(path)
		if resolved[index] != path {
			changed = true
		}
	}
	if !changed {
		return files
	}
	return resolved
}

func sameFilePaths(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
