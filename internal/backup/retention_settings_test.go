package backup

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/CarlFlo/tally/internal/database"
	"github.com/CarlFlo/tally/internal/settings"
)

func TestRetentionReloadsSavedSettingWithoutRestart(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, err := database.Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := settings.Store{DB: db}
	if _, err := db.Exec(`INSERT INTO application_settings(key,data,revision) VALUES('backups','{"keep":3}',1)`); err != nil {
		t.Fatal(err)
	}
	revision, err := store.Save(ctx, "backups", settings.Backups{Keep: 3}, 1)
	if err != nil {
		t.Fatal(err)
	}
	service := Service{DB: db, DataDir: dir, Path: filepath.Join(dir, "backups"), Keep: 1}
	if _, err = service.Create(ctx, "manual"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if _, err = service.Create(ctx, "auto"); err != nil {
			t.Fatal(err)
		}
	}
	if files, err := ListFiles(service.Path); err != nil || len(files) != 4 {
		t.Fatal("ignored saved retention", files, err)
	}
	if _, err = store.Save(ctx, "backups", settings.Backups{Keep: 1}, revision); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Create(ctx, "auto"); err != nil {
		t.Fatal(err)
	}
	if files, err := ListFiles(service.Path); err != nil || len(files) != 2 {
		t.Fatal("did not apply retention change", files, err)
	}
}
