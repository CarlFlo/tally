package commands

import (
	"path/filepath"
	"testing"

	"github.com/CarlFlo/tally/internal/config"
)

func TestBackupArchivePathDefaultsToBackupsDirectory(t *testing.T) {
	dir := t.TempDir()
	got, err := backupArchivePath(config.Config{DataDir: dir}, "tally-backup.zip")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "backups", "tally-backup.zip")
	want, err = filepath.Abs(want)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBackupArchivePathPreservesAbsolutePath(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "import.zip")
	got, err := backupArchivePath(config.Config{DataDir: t.TempDir()}, archive)
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Clean(archive) {
		t.Fatalf("got %q, want %q", got, filepath.Clean(archive))
	}
}

func TestBackupArchivePathRejectsEscapingRelativePath(t *testing.T) {
	if _, err := backupArchivePath(config.Config{DataDir: t.TempDir()}, "../outside.zip"); err == nil {
		t.Fatal("relative path escaped the backups directory")
	}
}
