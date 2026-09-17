package backup

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/CarlFlo/tally/internal/database"
)

func TestLiveRestoreAppliesBackupWithoutRestartAndKeepsRelationshipsAndPreferences(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, err := database.Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := Service{DB: db, DataDir: dir, Path: filepath.Join(dir, "backups"), Keep: 2}
	if _, err = db.Exec(`INSERT INTO profiles(id,display_name,avatar,created_at) VALUES('profile-owner','My profile','violet',1);
UPDATE profiles SET display_name='From backup' WHERE id='profile-owner';
INSERT INTO shows(id,name) VALUES('restore-show','Restore show');
INSERT INTO profile_shows(profile_id,show_id,added_at,favorite) VALUES('profile-owner','restore-show',1,1);
INSERT INTO profile_preferences(profile_id,data) VALUES('profile-owner','{"theme":"dark","calendar_view":"agenda"}');
INSERT INTO browser_preferences VALUES('restore-browser','dark',1);`); err != nil {
		t.Fatal(err)
	}
	filename, err := service.Create(ctx, "manual")
	if err != nil {
		t.Fatal(err)
	}
	id := archiveID(filename)
	if _, err = db.Exec(`UPDATE profiles SET display_name='Current state' WHERE id='profile-owner';
DELETE FROM shows WHERE id='restore-show';
UPDATE profile_preferences SET data='{"theme":"light","calendar_view":"month"}' WHERE profile_id='profile-owner';
UPDATE browser_preferences SET theme='light' WHERE id='restore-browser';`); err != nil {
		t.Fatal(err)
	}
	manifest, err := service.RestoreArchive(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	var name, preference, browserTheme string
	if err = db.QueryRow("SELECT display_name FROM profiles WHERE id='profile-owner'").Scan(&name); err != nil || name != "From backup" {
		t.Fatal("running database did not switch to restored state", name, err)
	}
	var followed int
	if err = db.QueryRow("SELECT COUNT(*) FROM profile_shows WHERE profile_id='profile-owner' AND show_id='restore-show' AND favorite=1").Scan(&followed); err != nil || followed != 1 {
		t.Fatal("restored follow relationship was lost", followed, err)
	}
	if err = db.QueryRow("SELECT data FROM profile_preferences WHERE profile_id='profile-owner'").Scan(&preference); err != nil || preference != `{"theme":"dark","calendar_view":"agenda"}` {
		t.Fatal("profile preferences were not restored", preference, err)
	}
	if err = db.QueryRow("SELECT theme FROM browser_preferences WHERE id='restore-browser'").Scan(&browserTheme); err != nil || browserTheme != "dark" {
		t.Fatal("browser appearance was not restored", browserTheme, err)
	}
	if manifest.Schema != database.Version {
		t.Fatalf("restored schema %d, want %d", manifest.Schema, database.Version)
	}
	if _, err = service.FindArchive(ctx, id); err != nil {
		t.Fatal("restore removed current backup inventory", err)
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
	if _, err = db.Exec("INSERT INTO profiles(id,display_name,avatar,created_at) VALUES('profile-owner','From backup','violet',1)"); err != nil {
		t.Fatal(err)
	}
	filename, err := service.Create(ctx, "manual")
	if err != nil {
		t.Fatal(err)
	}
	id := archiveID(filename)
	if _, err = db.Exec(`UPDATE profiles SET display_name='Keep current' WHERE id='profile-owner';
CREATE TRIGGER prevent_restore BEFORE DELETE ON profiles BEGIN SELECT RAISE(ABORT,'fixture restore failure'); END;`); err != nil {
		t.Fatal(err)
	}
	if _, err = service.RestoreArchive(ctx, id); err == nil {
		t.Fatal("restore unexpectedly succeeded")
	}
	var name string
	if err = db.QueryRow("SELECT display_name FROM profiles WHERE id='profile-owner'").Scan(&name); err != nil || name != "Keep current" {
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
	if _, err = sourceDB.Exec(`INSERT INTO profiles(id,display_name,avatar,created_at) VALUES('user0','My profile','violet',1);
DROP TABLE torrent_automation_feedback; DROP TABLE torrent_bad_hashes; DROP TABLE torrent_show_policy; DROP TABLE torrent_automation_runs;
DROP TRIGGER profile_role_after_insert; DROP TRIGGER profile_role_last_admin_update; DROP TRIGGER profile_role_last_admin_delete;
DROP TABLE profile_roles; DROP TABLE profile_id_aliases;
DROP TABLE inbox_dismissals; DROP TABLE inbox_state; DROP INDEX episode_release_time;
DROP TABLE activity_log; DROP TABLE browser_preferences; DROP TABLE notification_state; DROP TABLE notification_outbox;
DROP TABLE application_settings; DROP TABLE show_actions; ALTER TABLE profile_shows DROP COLUMN favorite;
ALTER TABLE jobs DROP COLUMN enabled; ALTER TABLE jobs DROP COLUMN paused; ALTER TABLE jobs DROP COLUMN failures;
ALTER TABLE jobs DROP COLUMN revision; DROP TABLE download_client_settings; ALTER TABLE profiles DROP COLUMN locale; ALTER TABLE profiles DROP COLUMN auth_method; CREATE TABLE counters (key TEXT PRIMARY KEY, value INTEGER NOT NULL); INSERT INTO counters VALUES ('profile',1); PRAGMA user_version=1;
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
	_ = targetDB.QueryRow("SELECT display_name FROM profiles WHERE display_name='Legacy profile'").Scan(&name)
	if version != database.Version || name != "Legacy profile" {
		t.Fatal("old archive did not restore and upgrade")
	}
	if restoredManifest.Schema != 1 || restoredManifest.AppVersion != "0.1.0" {
		t.Fatal("source backup metadata was not preserved", restoredManifest)
	}
}
