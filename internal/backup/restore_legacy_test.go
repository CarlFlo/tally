package backup

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/CarlFlo/tally/internal/database"
)

func TestVersionOneArchiveStillRestoresAndUpgrades(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, e := database.Open(ctx, dir)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if _, e = db.Exec("INSERT INTO profiles(id,display_name,avatar,created_at) VALUES('user0','My profile','violet',1); DROP TRIGGER profile_role_after_insert; DROP TRIGGER profile_role_last_admin_update; DROP TRIGGER profile_role_last_admin_delete; DROP TABLE profile_roles; DROP TABLE profile_id_aliases; DROP TABLE inbox_dismissals; DROP TABLE inbox_state; DROP INDEX episode_release_time; DROP TABLE activity_log; DROP TABLE browser_preferences; DROP TABLE notification_state; DROP TABLE notification_outbox; DROP TABLE application_settings; DROP TABLE show_actions; ALTER TABLE profile_shows DROP COLUMN favorite; ALTER TABLE jobs DROP COLUMN enabled; ALTER TABLE jobs DROP COLUMN paused; ALTER TABLE jobs DROP COLUMN failures; ALTER TABLE jobs DROP COLUMN revision; DROP TABLE download_client_settings; ALTER TABLE profiles DROP COLUMN locale; ALTER TABLE profiles DROP COLUMN auth_method; CREATE TABLE counters (key TEXT PRIMARY KEY, value INTEGER NOT NULL); INSERT INTO counters VALUES ('profile',1); PRAGMA user_version=1; UPDATE profiles SET display_name='From version one' WHERE id='user0'"); e != nil {
		t.Fatal(e)
	}
	s := Service{DB: db, DataDir: dir, Path: filepath.Join(dir, "backups"), Keep: 1}
	name, e := s.Create(ctx, "manual")
	if e != nil {
		t.Fatal(e)
	}
	target := t.TempDir()
	if e = Restore(ctx, filepath.Join(s.Path, name), target); e != nil {
		t.Fatal(e)
	}
	restored, e := database.Open(ctx, target)
	if e != nil {
		t.Fatal(e)
	}
	defer restored.Close()
	var version int
	var profile string
	_ = restored.QueryRow("PRAGMA user_version").Scan(&version)
	_ = restored.QueryRow("SELECT display_name FROM profiles WHERE display_name='From version one'").Scan(&profile)
	if version != database.Version || profile != "From version one" {
		t.Fatal("old archive did not restore and upgrade")
	}
}
