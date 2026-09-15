package backup

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/CarlFlo/mediaManager/internal/database"
)

func TestLiveRestoreAppliesBackupWithoutRestartAndKeepsInventory(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, err := database.Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := Service{DB: db, DataDir: dir, Path: filepath.Join(dir, "backups"), Keep: 2}
	if _, err = db.Exec("UPDATE profiles SET display_name='From backup' WHERE id='user0'"); err != nil {
		t.Fatal(err)
	}
	filename, err := service.Create(ctx, "manual")
	if err != nil {
		t.Fatal(err)
	}
	var id string
	if err = db.QueryRow("SELECT id FROM backup_records WHERE filename=?", filename).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("UPDATE profiles SET display_name='Current state' WHERE id='user0'"); err != nil {
		t.Fatal(err)
	}
	manifest, err := service.RestoreRecord(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	var name string
	if err = db.QueryRow("SELECT display_name FROM profiles WHERE id='user0'").Scan(&name); err != nil || name != "From backup" {
		t.Fatal("running database did not switch to restored state", name, err)
	}
	if manifest.Schema != database.Version {
		t.Fatalf("restored schema %d, want %d", manifest.Schema, database.Version)
	}
	var count int
	if err = db.QueryRow("SELECT COUNT(*) FROM backup_records WHERE id=?", id).Scan(&count); err != nil || count != 1 {
		t.Fatal("restore removed current backup inventory", count, err)
	}
}

func TestLiveRestoreRollsBackDatabaseChangesOnApplyFailure(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, err := database.Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := Service{DB: db, DataDir: dir, Path: filepath.Join(dir, "backups"), Keep: 2}
	if _, err = db.Exec("UPDATE profiles SET display_name='From backup' WHERE id='user0'"); err != nil {
		t.Fatal(err)
	}
	filename, err := service.Create(ctx, "manual")
	if err != nil {
		t.Fatal(err)
	}
	var id string
	_ = db.QueryRow("SELECT id FROM backup_records WHERE filename=?", filename).Scan(&id)
	if _, err = db.Exec(`UPDATE profiles SET display_name='Keep current' WHERE id='user0';
CREATE TRIGGER prevent_restore BEFORE DELETE ON profiles BEGIN SELECT RAISE(ABORT,'fixture restore failure'); END;`); err != nil {
		t.Fatal(err)
	}
	if _, err = service.RestoreRecord(ctx, id); err == nil {
		t.Fatal("restore unexpectedly succeeded")
	}
	var name string
	if err = db.QueryRow("SELECT display_name FROM profiles WHERE id='user0'").Scan(&name); err != nil || name != "Keep current" {
		t.Fatal("failed restore changed current data", name, err)
	}
}

func TestLiveRestoreMigratesOlderSchemaAndReportsSourceVersion(t *testing.T) {
	ctx := context.Background()
	sourceDir := t.TempDir()
	sourceDB, err := database.Open(ctx, sourceDir)
	if err != nil {
		t.Fatal(err)
	}
	defer sourceDB.Close()
	if _, err = sourceDB.Exec(`DROP TABLE inbox_dismissals; DROP TABLE inbox_state; DROP INDEX episode_release_time;
DROP TABLE activity_log; DROP TABLE browser_preferences; DROP TABLE notification_state; DROP TABLE notification_outbox;
DROP TABLE application_settings; DROP TABLE show_actions; ALTER TABLE profile_shows DROP COLUMN favorite;
ALTER TABLE jobs DROP COLUMN enabled; ALTER TABLE jobs DROP COLUMN paused; ALTER TABLE jobs DROP COLUMN failures;
ALTER TABLE jobs DROP COLUMN revision; DROP TABLE download_client_settings; PRAGMA user_version=1;
UPDATE profiles SET display_name='Legacy profile' WHERE id='user0'`); err != nil {
		t.Fatal(err)
	}
	source := Service{DB: sourceDB, DataDir: sourceDir, Path: filepath.Join(sourceDir, "backups"), Keep: 1}
	stage := t.TempDir()
	snapshot := filepath.Join(stage, "app.db")
	if err = sourceDB.Snapshot(ctx, snapshot); err != nil {
		t.Fatal(err)
	}
	manifest, files, err := source.snapshotFiles(ctx, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	manifest.AppVersion = "0.1.0"
	archive := filepath.Join(stage, "legacy.zip")
	if err = writeArchive(ctx, archive, files, manifest); err != nil {
		t.Fatal(err)
	}

	targetDir := t.TempDir()
	targetDB, err := database.Open(ctx, targetDir)
	if err != nil {
		t.Fatal(err)
	}
	defer targetDB.Close()
	target := Service{DB: targetDB, DataDir: targetDir, Path: filepath.Join(targetDir, "backups")}
	restoredManifest, err := target.RestoreLive(ctx, archive)
	if err != nil {
		t.Fatal(err)
	}
	var version int
	var name string
	_ = targetDB.QueryRow("PRAGMA user_version").Scan(&version)
	_ = targetDB.QueryRow("SELECT display_name FROM profiles WHERE id='user0'").Scan(&name)
	if version != database.Version || name != "Legacy profile" {
		t.Fatal("legacy backup was not migrated into the running database", version, name)
	}
	if restoredManifest.Schema != 1 || restoredManifest.AppVersion != "0.1.0" {
		t.Fatal("source backup metadata was not preserved", restoredManifest)
	}
}
