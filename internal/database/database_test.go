package database

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMigrationFailurePreservesSnapshotAndRollsBack(t *testing.T) {
	dir := t.TempDir()
	db, e := sql.Open("sqlite", filepath.ToSlash(filepath.Join(dir, "app.db")))
	if e != nil {
		t.Fatal(e)
	}
	_, e = db.Exec("CREATE TABLE profiles(id TEXT PRIMARY KEY); INSERT INTO profiles VALUES('legacy')")
	if e != nil {
		t.Fatal(e)
	}
	db.Close()
	if _, e = Open(context.Background(), dir); e == nil {
		t.Fatal("conflicting migration succeeded")
	}
	snapshots, _ := filepath.Glob(filepath.Join(dir, "pre-upgrade-v0-*.db"))
	if len(snapshots) != 1 {
		t.Fatal("pre-upgrade snapshot missing")
	}
	raw, _ := sql.Open("sqlite", filepath.ToSlash(filepath.Join(dir, "app.db")))
	defer raw.Close()
	var id string
	if e = raw.QueryRow("SELECT id FROM profiles").Scan(&id); e != nil || id != "legacy" {
		t.Fatal("migration damaged original schema")
	}
	if _, e = os.Stat(filepath.Join(dir, "migration-failed.json")); e != nil {
		t.Fatal("failure marker missing")
	}
}

func TestDowngradeRefusedAndStaleJobsInterrupted(t *testing.T) {
	dir := t.TempDir()
	s, e := Open(context.Background(), dir)
	if e != nil {
		t.Fatal(e)
	}
	_, _ = s.Exec("INSERT INTO job_runs(id,job_key,trigger,started_at,status) VALUES('run','metadata','scheduled',?,'running')", time.Now().Unix())
	s.Close()
	s, e = Open(context.Background(), dir)
	if e != nil {
		t.Fatal(e)
	}
	var status string
	_ = s.QueryRow("SELECT status FROM job_runs WHERE id='run'").Scan(&status)
	if status != "interrupted" {
		t.Fatal("stale job stayed running")
	}
	_, _ = s.Exec("PRAGMA user_version=999")
	s.Close()
	if _, e = Open(context.Background(), dir); e == nil {
		t.Fatal("incompatible downgrade allowed")
	}
}
