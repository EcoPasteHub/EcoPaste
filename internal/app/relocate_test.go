package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCopyTreeHonorsCanceledContext(t *testing.T) {
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "data.txt"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dst := filepath.Join(t.TempDir(), "target")
	err := copyTree(ctx, src, dst, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("copyTree error = %v, want context.Canceled", err)
	}
	if _, err := os.Stat(dst); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("target exists after canceled copy: %v", err)
	}
}
