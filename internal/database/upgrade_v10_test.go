package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestVersionNineUpgradeAddsTorrentMediaProfiles(t *testing.T) {
	dir := t.TempDir()
	raw, err := sql.Open("sqlite", filepath.ToSlash(filepath.Join(dir, "app.db")))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = raw.Exec(schema + "\n" + migration002 + "\n" + migration003 + "\n" + migration004 + "\n" + migration005 + "\n" + migration006 + "\n" + migration007 + "\n" + migration008 + "\n" + migration009 + "\nPRAGMA user_version=9;"); err != nil {
		t.Fatal(err)
	}
	if _, err = raw.Exec("INSERT INTO shows(id,name) VALUES('show-1','Example Show')"); err != nil {
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

	var showType string
	if err = db.QueryRow("SELECT show_type FROM shows WHERE id='show-1'").Scan(&showType); err != nil {
		t.Fatal(err)
	}
	if showType != "" {
		t.Fatalf("legacy show received unexpected type %q", showType)
	}
	if _, err = db.Exec("INSERT INTO torrent_show_media_profile(show_id,profile,updated_at) VALUES('show-1','animated',1)"); err != nil {
		t.Fatalf("media profile table unavailable after upgrade: %v", err)
	}
	var version int
	if err = db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != Version {
		t.Fatalf("unexpected schema version %d err=%v", version, err)
	}
}
