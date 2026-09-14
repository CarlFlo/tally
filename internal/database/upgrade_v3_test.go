package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestVersionTwoUpgradeKeepsSchedulesAndClient(t *testing.T) {
	dir := t.TempDir()
	raw, e := sql.Open("sqlite", filepath.ToSlash(filepath.Join(dir, "app.db")))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = raw.Exec(schema + "\n" + migration002 + "\nPRAGMA user_version=2; INSERT INTO jobs VALUES('metadata','metadata','15 8 * * 1-5',123); UPDATE download_client_settings SET adapter='qbittorrent',fields='{\"url\":\"http://fixture.invalid\",\"api_key\":\"fixture\"}',revision=4 WHERE id=1;"); e != nil {
		t.Fatal(e)
	}
	raw.Close()
	db, e := Open(context.Background(), dir)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	var spec, key string
	var rev int
	if e = db.QueryRow("SELECT schedule FROM jobs WHERE key='metadata'").Scan(&spec); e != nil || spec != "15 8 * * 1-5" {
		t.Fatal("schedule lost", e)
	}
	if e = db.QueryRow("SELECT json_extract(fields,'$.api_key'),revision FROM download_client_settings WHERE id=1").Scan(&key, &rev); e != nil || key != "fixture" || rev != 4 {
		t.Fatal("client settings lost", e)
	}
	paths, _ := filepath.Glob(filepath.Join(dir, "pre-upgrade-v2-*.db"))
	if len(paths) != 1 {
		t.Fatal("schema 2 backup missing")
	}
}
