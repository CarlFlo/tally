package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestVersionThreeUpgradePreservesLibraryAndNotificationSettings(t *testing.T) {
	dir := t.TempDir()
	raw, err := sql.Open("sqlite", filepath.ToSlash(filepath.Join(dir, "app.db")))
	if err != nil {
		t.Fatal(err)
	}
	_, err = raw.Exec(schema + "\n" + migration002 + "\n" + migration003 + `PRAGMA user_version=3; INSERT INTO profiles VALUES('user0','Fixture','violet',1); INSERT INTO application_settings VALUES('notifications','{"enabled":true,"url":"http://fixture.invalid/secret"}',7); INSERT INTO shows(id,name) VALUES('show','Preserved show'); INSERT INTO profile_shows VALUES('user0','show',1,1);`)
	if err != nil {
		t.Fatal(err)
	}
	raw.Close()
	db, err := Open(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var favorite, revision, version int
	_ = db.QueryRow("SELECT favorite FROM profile_shows").Scan(&favorite)
	_ = db.QueryRow("SELECT revision FROM application_settings").Scan(&revision)
	_ = db.QueryRow("PRAGMA user_version").Scan(&version)
	if favorite != 1 || revision != 7 || version != Version {
		t.Fatal("upgrade lost state", favorite, revision, version)
	}
	paths, _ := filepath.Glob(filepath.Join(dir, "pre-upgrade-v3-*.db"))
	if len(paths) != 1 {
		t.Fatal("missing pre-upgrade backup")
	}
	snapshot, err := sql.Open("sqlite", filepath.ToSlash(paths[0]))
	if err != nil {
		t.Fatal(err)
	}
	defer snapshot.Close()
	if err = Validate(context.Background(), snapshot); err != nil {
		t.Fatal(err)
	}
	_ = snapshot.QueryRow("PRAGMA user_version").Scan(&version)
	if version != 3 {
		t.Fatal("snapshot taken after upgrade")
	}
}
