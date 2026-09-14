package backup

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/CarlFlo/mediaManager/internal/database"
)

func TestConsistentBackupRetentionAndRestore(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, e := database.Open(ctx, dir)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	_, e = db.Exec("INSERT INTO profiles VALUES('user1','Saved','user1-test.png',?)", time.Now().Unix())
	if e != nil {
		t.Fatal(e)
	}
	_ = os.Mkdir(filepath.Join(dir, "avatars"), 0700)
	_ = os.WriteFile(filepath.Join(dir, "avatars", "user1-test.png"), []byte("avatar"), 0600)
	_ = os.WriteFile(filepath.Join(dir, "cache.db"), []byte("DISPOSABLE"), 0600)
	if _, e = db.Exec(`UPDATE download_client_settings SET adapter='qbittorrent',fields='{"url":"http://client.example","api_key":"qbt_0123456789abcdefghijklmnopqr"}',revision=1 WHERE id=1`); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(`INSERT INTO shows(id,name) VALUES('saved-show','Saved show');
		INSERT INTO profile_shows(profile_id,show_id,added_at,favorite) VALUES('user1','saved-show',1,1);
		INSERT INTO application_settings(key,data) VALUES('search','{"providers":[{"id":"saved","api_key":"fixture-key"}]}');
		INSERT INTO show_actions(profile_id,external_id,name,desired,status,updated_at) VALUES('user1',7,'Pending show',1,'queued',1);
		INSERT INTO jobs(key,type,schedule,enabled,paused,failures) VALUES('metadata','metadata','15 8 * * 1-5',0,1,3)`); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(`INSERT INTO activity_log(action,profile_id,message,created_at) VALUES('show_added','user1','Added saved show',1);
 INSERT INTO inbox_state VALUES('user1',1,0);
 INSERT INTO inbox_dismissals SELECT 'user1',id FROM activity_log WHERE action='show_added';
 INSERT INTO application_settings(key,data) VALUES('backups','{"keep":1}');
 INSERT INTO browser_preferences VALUES('browser-fixture','light',1);
 INSERT INTO application_settings(key,data) VALUES('notifications','{"enabled":true,"type":"discord","discord_url":"http://fixture.invalid/private","delivery_time":"18:30","events":["episode_released"]}');
 INSERT INTO notification_outbox(event_key,event,message,occurred_at,available_at) VALUES('release:fixture','episode_released','Release pending',1,2);`); e != nil {
		t.Fatal(e)
	}
	s := Service{DB: db, DataDir: dir, Path: filepath.Join(dir, "backups"), Keep: 1}
	manual, e := s.Create(ctx, "manual")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Create(ctx, "auto"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Create(ctx, "auto"); e != nil {
		t.Fatal(e)
	}
	files, e := ListFiles(s.Path)
	if e != nil || len(files) != 2 {
		t.Fatalf("retention failed: %v %v", files, e)
	}
	z, e := zip.OpenReader(filepath.Join(s.Path, manual))
	if e != nil {
		t.Fatal(e)
	}
	for _, f := range z.File {
		if f.Name == "cache.db" {
			t.Fatal("cache in backup")
		}
	}
	z.Close()
	target := t.TempDir()
	if e = Restore(ctx, filepath.Join(s.Path, manual), target); e != nil {
		t.Fatal(e)
	}
	restored, e := database.Open(ctx, target)
	if e != nil {
		t.Fatal(e)
	}
	defer restored.Close()
	var name string
	if e = restored.QueryRow("SELECT display_name FROM profiles WHERE id='user1'").Scan(&name); e != nil || name != "Saved" {
		t.Fatal("profile not restored")
	}
	if _, e = os.Stat(filepath.Join(target, "avatars", "user1-test.png")); e != nil {
		t.Fatal("avatar not restored")
	}
	var secret string
	if e = restored.QueryRow("SELECT json_extract(fields,'$.api_key') FROM download_client_settings WHERE id=1").Scan(&secret); e != nil || secret != "qbt_0123456789abcdefghijklmnopqr" {
		t.Fatal("client settings not restored", e)
	}
	for _, query := range []string{
		"SELECT COUNT(*) FROM activity_log WHERE action='show_added'",
		"SELECT COUNT(*) FROM browser_preferences WHERE theme='light'",
		"SELECT COUNT(*) FROM inbox_state WHERE profile_id='user1' AND seen_id=1",
		"SELECT COUNT(*) FROM inbox_dismissals WHERE profile_id='user1'",
		"SELECT COUNT(*) FROM application_settings WHERE key='backups' AND json_extract(data,'$.keep')=1",
		"SELECT COUNT(*) FROM application_settings WHERE json_extract(data,'$.discord_url')='http://fixture.invalid/private'",
		"SELECT COUNT(*) FROM notification_outbox WHERE event_key='release:fixture' AND status='pending'",
		"SELECT COUNT(*) FROM profile_shows WHERE profile_id='user1' AND favorite=1",
		"SELECT COUNT(*) FROM application_settings WHERE json_extract(data,'$.providers[0].api_key')='fixture-key'",
		"SELECT COUNT(*) FROM show_actions WHERE profile_id='user1' AND status='queued' AND desired=1",
		"SELECT COUNT(*) FROM jobs WHERE schedule='15 8 * * 1-5' AND enabled=0 AND paused=1 AND failures=3",
	} {
		var count int
		if e = restored.QueryRow(query).Scan(&count); e != nil || count != 1 {
			t.Fatal("durable library/settings state was not restored", query, e)
		}
	}
}
