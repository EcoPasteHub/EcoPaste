package app

import "testing"

func TestResolveFileReferencesLeavesRealPaths(t *testing.T) {
	files := []string{"/tmp/a.txt", "/tmp/b/c.md"}
	if got := resolveFileReferences(files); !sameFilePaths(got, files) {
		t.Fatalf("resolveFileReferences changed real paths: %v", got)
	}
}

func TestSameFilePaths(t *testing.T) {
	if !sameFilePaths([]string{"/a", "/b"}, []string{"/a", "/b"}) {
		t.Fatal("equal paths reported different")
	}
	if sameFilePaths([]string{"/a"}, []string{"/a", "/b"}) {
		t.Fatal("different lengths reported equal")
	}
	if sameFilePaths([]string{"/a"}, []string{"/b"}) {
		t.Fatal("different paths reported equal")
	}
}
