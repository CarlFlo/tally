package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestVersionFiveUpgradeRewritesEveryProfileReference(t *testing.T) {
	dir := t.TempDir()
	raw, err := sql.Open("sqlite", filepath.ToSlash(filepath.Join(dir, "app.db")))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = raw.Exec(schema + "\n" + migration002 + "\n" + migration003 + "\n" + migration004 + "\n" + migration005 + "\nPRAGMA user_version=5;"); err != nil {
		t.Fatal(err)
	}
	if _, err = raw.Exec(`
		INSERT INTO profiles VALUES('user0','Owner','violet',1),('user1','Member','mint',2);
		INSERT INTO profile_preferences VALUES('user0','{"theme":"dark"}');
		INSERT INTO profile_identities VALUES('https://issuer.example','subject','user0');
		INSERT INTO local_credentials VALUES('user0','fixture-hash',0);
		INSERT INTO sessions VALUES('session','user0',1,1,9999999999,0,'fixture');
		INSERT INTO shows(id,name) VALUES('show','Example');
		INSERT INTO seasons(id,show_id,number) VALUES('season','show',1);
		INSERT INTO episodes(id,show_id,season,number,name) VALUES('episode','show',1,1,'Pilot');
		INSERT INTO profile_shows(profile_id,show_id,added_at,favorite) VALUES('user0','show',1,1);
		INSERT INTO profile_episode_state VALUES('user1','episode',1,1,1);
		INSERT INTO torrent_search_history VALUES('search','user1','query','provider',1,1);
		INSERT INTO torrent_send_history VALUES('send','user1','key','hash','name','sent','',1);
		INSERT INTO show_actions VALUES('user1',7,'Queued',1,'queued',1,'','',1);
		INSERT INTO activity_log(id,action,profile_id,message,created_at) VALUES(10,'show_added','user0','Owner action',1),(11,'show_added','user1','Member action',2);
		INSERT INTO inbox_state VALUES('user1',11,10);
		INSERT INTO inbox_dismissals VALUES('user1',10);
	`); err != nil {
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

	var ownerID, memberID string
	if err = db.QueryRow("SELECT profile_id FROM profile_id_aliases WHERE alias='user0'").Scan(&ownerID); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow("SELECT profile_id FROM profile_id_aliases WHERE alias='user1'").Scan(&memberID); err != nil {
		t.Fatal(err)
	}
	if ownerID == "user0" || memberID == "user1" || ownerID == memberID {
		t.Fatal("legacy IDs were not replaced", ownerID, memberID)
	}

	var ownerAdmin, memberAdmin int
	if err = db.QueryRow("SELECT is_admin FROM profile_roles WHERE profile_id=?", ownerID).Scan(&ownerAdmin); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow("SELECT is_admin FROM profile_roles WHERE profile_id=?", memberID).Scan(&memberAdmin); err != nil {
		t.Fatal(err)
	}
	if ownerAdmin != 1 || memberAdmin != 0 {
		t.Fatal("legacy administrator role was not migrated", ownerAdmin, memberAdmin)
	}

	checks := []struct {
		name  string
		query string
		args  []any
	}{
		{"preference", "SELECT COUNT(*) FROM profile_preferences WHERE profile_id=?", []any{ownerID}},
		{"identity", "SELECT COUNT(*) FROM profile_identities WHERE profile_id=?", []any{ownerID}},
		{"credential", "SELECT COUNT(*) FROM local_credentials WHERE profile_id=?", []any{ownerID}},
		{"session", "SELECT COUNT(*) FROM sessions WHERE profile_id=?", []any{ownerID}},
		{"follow", "SELECT COUNT(*) FROM profile_shows WHERE profile_id=?", []any{ownerID}},
		{"episode state", "SELECT COUNT(*) FROM profile_episode_state WHERE profile_id=?", []any{memberID}},
		{"torrent search", "SELECT COUNT(*) FROM torrent_search_history WHERE profile_id=?", []any{memberID}},
		{"torrent send", "SELECT COUNT(*) FROM torrent_send_history WHERE profile_id=?", []any{memberID}},
		{"show action", "SELECT COUNT(*) FROM show_actions WHERE profile_id=?", []any{memberID}},
		{"inbox state", "SELECT COUNT(*) FROM inbox_state WHERE profile_id=?", []any{memberID}},
		{"inbox dismissal", "SELECT COUNT(*) FROM inbox_dismissals WHERE profile_id=?", []any{memberID}},
		{"owner activity", "SELECT COUNT(*) FROM activity_log WHERE profile_id=?", []any{ownerID}},
		{"member activity", "SELECT COUNT(*) FROM activity_log WHERE profile_id=?", []any{memberID}},
	}
	for _, check := range checks {
		var count int
		if err = db.QueryRow(check.query, check.args...).Scan(&count); err != nil || count != 1 {
			t.Fatalf("%s was not migrated: count=%d err=%v", check.name, count, err)
		}
	}

	for _, table := range []string{
		"profiles", "profile_preferences", "profile_identities", "local_credentials",
		"sessions", "profile_shows", "profile_episode_state", "torrent_search_history",
		"torrent_send_history", "show_actions", "inbox_state", "inbox_dismissals", "activity_log",
	} {
		column := "profile_id"
		if table == "profiles" {
			column = "id"
		}
		var count int
		if err = db.QueryRow("SELECT COUNT(*) FROM "+table+" WHERE "+column+" IN ('user0','user1')").Scan(&count); err != nil || count != 0 {
			t.Fatalf("legacy profile ID survived in %s: count=%d err=%v", table, count, err)
		}
	}

	var counters int
	if err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='counters'").Scan(&counters); err != nil || counters != 0 {
		t.Fatal("obsolete profile counter survived schema 6", err)
	}
}
