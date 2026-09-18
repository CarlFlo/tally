package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestVersionElevenUpgradePromotesDownloadedStateToEpisode(t *testing.T) {
	dir := t.TempDir()
	raw, err := sql.Open("sqlite", filepath.ToSlash(filepath.Join(dir, "app.db")))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = raw.Exec(schema + "\n" + migration002 + "\n" + migration003 + "\n" + migration004 + "\n" + migration005 + "\n" + migration006 + "\n" + migration007 + "\n" + migration008 + "\n" + migration009 + "\n" + migration010 + "\n" + migration011 + "\nPRAGMA user_version=11;"); err != nil {
		t.Fatal(err)
	}
	if _, err = raw.Exec(`INSERT INTO profiles(id,display_name,avatar,created_at) VALUES
		('profile-a','A','mint',1),('profile-b','B','violet',2);
	INSERT INTO shows(id,name) VALUES('show-a','Example Show');
	INSERT INTO episodes(id,show_id,season,number,name) VALUES
		('episode-a','show-a',1,1,'One'),('episode-b','show-a',1,2,'Two');
	INSERT INTO profile_episode_state(profile_id,episode_id,watched,downloaded,updated_at) VALUES
		('profile-a','episode-a',0,0,1),
		('profile-b','episode-a',0,1,1),
		('profile-a','episode-b',1,0,1);`); err != nil {
		t.Fatal(err)
	}
	if err = raw.Close(); err != nil {
		t.Fatal(err)
	}

	db, err := Open(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var downloadedA, downloadedB int
	if err = db.QueryRow("SELECT downloaded FROM episodes WHERE id='episode-a'").Scan(&downloadedA); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow("SELECT downloaded FROM episodes WHERE id='episode-b'").Scan(&downloadedB); err != nil {
		t.Fatal(err)
	}
	if downloadedA != 1 || downloadedB != 0 {
		t.Fatalf("legacy downloaded state was not promoted globally: a=%d b=%d", downloadedA, downloadedB)
	}
	var version int
	if err = db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != Version {
		t.Fatalf("unexpected schema version %d err=%v", version, err)
	}
}
