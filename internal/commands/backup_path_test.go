package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CarlFlo/tally/internal/config"
)

func writeBackupFixture(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, "backups", name)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestBackupArchivePathDefaultsToBackupsDirectory(t *testing.T) {
	dir := t.TempDir()
	want := writeBackupFixture(t, dir, "tally-backup.zip")
	got, err := backupArchivePath(config.Config{DataDir: dir}, "tally-backup.zip")
	if err != nil {
		t.Fatal(err)
	}
	want, err = filepath.Abs(want)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBackupArchivePathAcceptsFilenameWithoutZipSuffix(t *testing.T) {
	dir := t.TempDir()
	want := writeBackupFixture(t, dir, "tally-backup.zip")
	got, err := backupArchivePath(config.Config{DataDir: dir}, "tally-backup")
	if err != nil {
		t.Fatal(err)
	}
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
	if err := os.WriteFile(archive, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
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

func TestBackupArchivePathExplainsMissingBackup(t *testing.T) {
	_, err := backupArchivePath(config.Config{DataDir: t.TempDir()}, "missing.zip")
	if err == nil {
		t.Fatal("missing backup was accepted")
	}
	if !strings.Contains(err.Error(), "without a filename to list available backups") {
		t.Fatalf("missing backup error is not actionable: %v", err)
	}
}
