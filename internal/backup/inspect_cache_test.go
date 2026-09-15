package backup

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestInspectDoesNotCacheCancellation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "imported.zip")
	if err := os.WriteFile(path, []byte("not a zip"), 0600); err != nil {
		t.Fatal(err)
	}
	service := &Service{Path: dir}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.Inspect(ctx, "imported.zip"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled inspection error = %v, want context.Canceled", err)
	}

	if _, err := service.Inspect(context.Background(), "imported.zip"); err == nil || errors.Is(err, context.Canceled) {
		t.Fatalf("subsequent inspection reused cancellation: %v", err)
	}
}
