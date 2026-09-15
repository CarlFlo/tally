package backup

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/CarlFlo/tally/internal/database"
)

func TestDeleteRecordRemovesArchive(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, err := database.Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := Service{DB: db, DataDir: dir, Path: filepath.Join(dir, "backups")}
	filename, err := service.Create(ctx, "manual")
	if err != nil {
		t.Fatal(err)
	}
	if err = service.DeleteRecord(ctx, archiveID(filename)); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(service.Path, filename)); !os.IsNotExist(err) {
		t.Fatal("backup archive still exists")
	}
}

func TestDeleteRecordRejectsMissingAndNonRegularArchives(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, err := database.Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := Service{DB: db, DataDir: dir, Path: filepath.Join(dir, "backups")}

	missing, err := service.Create(ctx, "manual")
	if err != nil {
		t.Fatal(err)
	}
	missingID := archiveID(missing)
	if err = os.Remove(filepath.Join(service.Path, missing)); err != nil {
		t.Fatal(err)
	}
	if err = service.DeleteRecord(ctx, missingID); !errors.Is(err, ErrNotFound) {
		t.Fatal("missing archive should not remain addressable", err)
	}

	blocked, err := service.Create(ctx, "manual")
	if err != nil {
		t.Fatal(err)
	}
	blockedID := archiveID(blocked)
	blockedPath := filepath.Join(service.Path, blocked)
	if err = os.Remove(blockedPath); err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(blockedPath, 0700); err != nil {
		t.Fatal(err)
	}
	if err = service.DeleteRecord(ctx, blockedID); !errors.Is(err, ErrNotFound) {
		t.Fatal("non-regular archive should not be addressable", err)
	}
	if info, statErr := os.Stat(blockedPath); statErr != nil || !info.IsDir() {
		t.Fatal("non-regular archive path was modified", statErr)
	}
}
