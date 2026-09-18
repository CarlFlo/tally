package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestVersionTenUpgradeAddsPendingMagnetVerificationState(t *testing.T) {
	dir := t.TempDir()
	raw, err := sql.Open("sqlite", filepath.ToSlash(filepath.Join(dir, "app.db")))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = raw.Exec(schema + "\n" + migration002 + "\n" + migration003 + "\n" + migration004 + "\n" + migration005 + "\n" + migration006 + "\n" + migration007 + "\n" + migration008 + "\n" + migration009 + "\n" + migration010 + "\nPRAGMA user_version=10;"); err != nil {
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

	if _, err = db.Exec("INSERT INTO torrent_magnet_verifications(run_id,infohash,status) VALUES('missing','aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','pending')"); err == nil {
		t.Fatal("magnet verification foreign key unexpectedly accepted a missing run")
	}
	var version int
	if err = db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 11 {
		t.Fatalf("unexpected schema version %d err=%v", version, err)
	}
	rows, err := db.Query("SELECT run_id,infohash,status,attempts,last_checked_at,completed_at,assessment,size_profile,error FROM torrent_magnet_verifications LIMIT 0")
	if err != nil {
		t.Fatal(err)
	}
	rows.Close()
}
