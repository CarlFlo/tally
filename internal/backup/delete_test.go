package backup

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/CarlFlo/mediaManager/internal/database"
)

func TestDeleteRecordRemovesArchiveAndMetadata(t *testing.T) {
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
	var id string
	_ = db.QueryRow("SELECT id FROM backup_records WHERE filename=?", filename).Scan(&id)
	if err = service.DeleteRecord(ctx, id); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(service.Path, filename)); !os.IsNotExist(err) {
		t.Fatal("backup archive still exists")
	}
	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM backup_records WHERE id=?", id).Scan(&count)
	if count != 0 {
		t.Fatal("backup record still exists")
	}
}

func TestDeleteRecordRepairsMissingFileButKeepsMetadataOnUnsafeFailure(t *testing.T) {
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
	var missingID string
	_ = db.QueryRow("SELECT id FROM backup_records WHERE filename=?", missing).Scan(&missingID)
	if err = os.Remove(filepath.Join(service.Path, missing)); err != nil {
		t.Fatal(err)
	}
	if err = service.DeleteRecord(ctx, missingID); err != nil {
		t.Fatal("missing file should be cleaned from inventory", err)
	}

	blocked, err := service.Create(ctx, "manual")
	if err != nil {
		t.Fatal(err)
	}
	var blockedID string
	_ = db.QueryRow("SELECT id FROM backup_records WHERE filename=?", blocked).Scan(&blockedID)
	blockedPath := filepath.Join(service.Path, blocked)
	if err = os.Remove(blockedPath); err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(blockedPath, 0700); err != nil {
		t.Fatal(err)
	}
	if err = service.DeleteRecord(ctx, blockedID); err == nil {
		t.Fatal("non-regular archive should reject deletion")
	}
	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM backup_records WHERE id=?", blockedID).Scan(&count)
	if count != 1 {
		t.Fatal("failed deletion removed backup metadata")
	}
}
