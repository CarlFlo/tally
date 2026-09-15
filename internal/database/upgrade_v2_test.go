package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestVersionOneUpgradePreservesDataAndSnapshot(t *testing.T) {
	dir := t.TempDir()
	raw, e := sql.Open("sqlite", filepath.ToSlash(filepath.Join(dir, "app.db")))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = raw.Exec(schema + "\nPRAGMA user_version=1; INSERT INTO profiles VALUES('user0','Keep this profile','violet',1); INSERT INTO profile_preferences VALUES('user0','{\"theme\":\"light\"}')"); e != nil {
		t.Fatal(e)
	}
	raw.Close()
	upgraded, e := Open(context.Background(), dir)
	if e != nil {
		t.Fatal(e)
	}
	defer upgraded.Close()
	var id, name, prefs string
	if e = upgraded.QueryRow("SELECT p.id,p.display_name,pp.data FROM profiles p JOIN profile_preferences pp ON pp.profile_id=p.id WHERE p.display_name='Keep this profile'").Scan(&id, &name, &prefs); e != nil || name != "Keep this profile" || prefs != "{\"theme\":\"light\"}" || id == "user0" {
		t.Fatal("upgrade lost or failed to migrate personal state", id, e)
	}
	var alias string
	if e = upgraded.QueryRow("SELECT profile_id FROM profile_id_aliases WHERE alias='user0'").Scan(&alias); e != nil || alias != id {
		t.Fatal("legacy profile alias was not preserved", alias, e)
	}
	var version, count int
	_ = upgraded.QueryRow("PRAGMA user_version").Scan(&version)
	_ = upgraded.QueryRow("SELECT COUNT(*) FROM download_client_settings WHERE id=1").Scan(&count)
	if version != Version || count != 1 {
		t.Fatal("client settings migration incomplete")
	}
	if err := upgraded.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='counters'").Scan(&count); err != nil || count != 0 {
		t.Fatal("obsolete profile counter survived schema 6", err)
	}
	paths, _ := filepath.Glob(filepath.Join(dir, "pre-upgrade-v1-*.db"))
	if len(paths) != 1 {
		t.Fatal("pre-upgrade snapshot missing")
	}
	snapshot, e := sql.Open("sqlite", filepath.ToSlash(paths[0]))
	if e != nil {
		t.Fatal(e)
	}
	defer snapshot.Close()
	_ = snapshot.QueryRow("PRAGMA user_version").Scan(&version)
	if version != 1 {
		t.Fatal("pre-upgrade snapshot was modified")
	}
}

func TestVersionTwoMigrationFailureIsAtomic(t *testing.T) {
	dir := t.TempDir()
	raw, e := sql.Open("sqlite", filepath.ToSlash(filepath.Join(dir, "app.db")))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = raw.Exec(schema + "\nPRAGMA user_version=1; DROP TABLE profile_preferences;"); e != nil {
		t.Fatal(e)
	}
	raw.Close()
	if _, e = Open(context.Background(), dir); e == nil {
		t.Fatal("incomplete source schema upgraded successfully")
	}
	raw, e = sql.Open("sqlite", filepath.ToSlash(filepath.Join(dir, "app.db")))
	if e != nil {
		t.Fatal(e)
	}
	defer raw.Close()
	var version, count int
	_ = raw.QueryRow("PRAGMA user_version").Scan(&version)
	_ = raw.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE name='download_client_settings'").Scan(&count)
	if version != 1 || count != 0 {
		t.Fatal("failed migration did not roll back its new table and version")
	}
}
