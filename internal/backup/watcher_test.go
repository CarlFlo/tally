package backup

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/CarlFlo/tally/internal/database"
)

func TestArchiveWatcherFiltersAndDebouncesChanges(t *testing.T) {
	dir := t.TempDir()
	changes := make(chan struct{}, 8)
	watcher, err := WatchArchives(dir, func() { changes <- struct{}{} })
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()

	if err = os.WriteFile(filepath.Join(dir, "ignore.tmp"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	assertNoArchiveChange(t, changes)

	path := filepath.Join(dir, "external.zip")
	if err = os.WriteFile(path, []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	if file, openErr := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0); openErr != nil {
		t.Fatal(openErr)
	} else {
		_, _ = file.WriteString("two")
		_ = file.Close()
	}
	waitForArchiveChange(t, changes)
	assertNoArchiveChange(t, changes)

	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	waitForArchiveChange(t, changes)
}

func TestArchiveWatcherDetectsRenameIntoDirectory(t *testing.T) {
	dir := t.TempDir()
	changes := make(chan struct{}, 4)
	watcher, err := WatchArchives(dir, func() { changes <- struct{}{} })
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()

	source := filepath.Join(dir, "incoming.part")
	if err = os.WriteFile(source, []byte("archive"), 0o600); err != nil {
		t.Fatal(err)
	}
	assertNoArchiveChange(t, changes)
	if err = os.Rename(source, filepath.Join(dir, "incoming.zip")); err != nil {
		t.Fatal(err)
	}
	waitForArchiveChange(t, changes)
}

func TestArchiveWatcherDetectsTallyCreatedBackup(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	db, err := database.Open(ctx, dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	path := filepath.Join(dataDir, "backups")
	changes := make(chan struct{}, 4)
	watcher, err := WatchArchives(path, func() { changes <- struct{}{} })
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()

	service := Service{DB: db, DataDir: dataDir, Path: path, Keep: 2}
	if _, err = service.Create(ctx, "manual"); err != nil {
		t.Fatal(err)
	}
	waitForArchiveChange(t, changes)
}

func waitForArchiveChange(t *testing.T, changes <-chan struct{}) {
	t.Helper()
	select {
	case <-changes:
	case <-time.After(3 * time.Second):
		t.Fatal("backup archive change was not observed")
	}
}

func assertNoArchiveChange(t *testing.T, changes <-chan struct{}) {
	t.Helper()
	select {
	case <-changes:
		t.Fatal("unrelated or duplicate backup archive change was observed")
	case <-time.After(350 * time.Millisecond):
	}
}
