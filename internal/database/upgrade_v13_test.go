package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestVersionTwelveUpgradePersistsAutomationRetryState(t *testing.T) {
	dir := t.TempDir()
	raw, err := sql.Open("sqlite", filepath.ToSlash(filepath.Join(dir, "app.db")))
	if err != nil {
		t.Fatal(err)
	}
	legacy := schema + "\n" + migration002 + "\n" + migration003 + "\n" + migration004 + "\n" + migration005 + "\n" + migration006 + "\n" + migration007 + "\n" + migration008 + "\n" + migration009 + "\n" + migration010 + "\n" + migration011 + "\n" + migration012 + "\nPRAGMA user_version=12;"
	if _, err = raw.Exec(legacy); err != nil {
		t.Fatal(err)
	}
	const started = int64(1_789_666_800)
	if _, err = raw.Exec(`INSERT INTO shows(id,name) VALUES('show-a','Example Show');
	INSERT INTO episodes(id,show_id,season,number,name,airstamp) VALUES('episode-a','show-a',1,2,'Second','2026-09-17T18:00:00Z');
	INSERT INTO torrent_automation_runs(id,show_id,episode_id,show_name,season,episode,query,status,engine_version,started_at,ended_at)
	VALUES('run-a','show-a','episode-a','Example Show',1,2,'Example Show S01E02','no_verified_candidate','4',?,?,?)`, started, started+1, started+1); err != nil {
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

	var attempts int
	var lastSearch, nextSearch int64
	if err = db.QueryRow(`SELECT attempts,last_search_at,next_search_at
		FROM torrent_automation_episode_state WHERE episode_id='episode-a'`).Scan(&attempts, &lastSearch, &nextSearch); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 || lastSearch != started || nextSearch != started+30*60 {
		t.Fatalf("unexpected migrated retry gate: attempts=%d last=%d next=%d", attempts, lastSearch, nextSearch)
	}
	var version int
	if err = db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != Version {
		t.Fatalf("unexpected schema version %d err=%v", version, err)
	}
}
