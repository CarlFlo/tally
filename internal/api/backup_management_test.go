package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/CarlFlo/tally/internal/appversion"
	"github.com/CarlFlo/tally/internal/database"
)

func TestBackupManagementListsVersionRestoresLiveAndDeletes(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	if _, err := s.DB.Exec(`INSERT INTO shows(id,name) VALUES('restore-show','Restore show');
INSERT INTO profile_shows(profile_id,show_id,added_at,favorite) VALUES('profile-admin','restore-show',1,1);
INSERT INTO profile_preferences(profile_id,data) VALUES('profile-admin','{"theme":"dark"}');`); err != nil {
		t.Fatal(err)
	}
	filename, err := s.Backup.Create(context.Background(), "manual")
	if err != nil {
		t.Fatal(err)
	}
	var id string
	if err = s.DB.QueryRow("SELECT id FROM backup_records WHERE filename=?", filename).Scan(&id); err != nil {
		t.Fatal(err)
	}
	owner := &http.Cookie{Name: "tally_profile", Value: "profile-admin"}
	listing := request(t, h, "GET", "/api/backups", nil, owner)
	expect(t, listing, 200)
	for _, want := range []string{`"kind":"manual"`, fmt.Sprintf(`"schema":%d`, database.Version), fmt.Sprintf(`"app_version":"%s"`, appversion.Version), `"compatible":true`} {
		if !strings.Contains(listing.Body.String(), want) {
			t.Fatalf("backup metadata missing %s: %s", want, listing.Body.String())
		}
	}
	if _, err = s.DB.Exec(`UPDATE profiles SET display_name='Changed after backup' WHERE id='profile-admin';
DELETE FROM shows WHERE id='restore-show';
UPDATE profile_preferences SET data='{"theme":"light"}' WHERE profile_id='profile-admin';`); err != nil {
		t.Fatal(err)
	}
	expect(t, request(t, h, "POST", "/api/backups/"+id+"/restore", map[string]any{}, owner), 200)
	var name string
	if err = s.DB.QueryRow("SELECT display_name FROM profiles WHERE id='profile-admin'").Scan(&name); err != nil || name != "My profile" {
		t.Fatal("restore did not apply without reopening the database", name, err)
	}
	var followed int
	if err = s.DB.QueryRow("SELECT COUNT(*) FROM profile_shows WHERE profile_id='profile-admin' AND show_id='restore-show'").Scan(&followed); err != nil || followed != 1 {
		t.Fatal("restore lost followed shows", followed, err)
	}
	var theme string
	if err = s.DB.QueryRow("SELECT json_extract(data,'$.theme') FROM profile_preferences WHERE profile_id='profile-admin'").Scan(&theme); err != nil || theme != "dark" {
		t.Fatal("restore lost profile theme", theme, err)
	}
	expect(t, request(t, h, "DELETE", "/api/backups/"+id, nil, owner), 200)
	if _, err = os.Stat(filepath.Join(s.Backup.Path, filename)); !os.IsNotExist(err) {
		t.Fatal("deleted backup file still exists")
	}
}

func TestBackupManagementRequiresOperatorAndFailedRestorePreservesStateAndNotifies(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	if _, err := s.DB.Exec("INSERT INTO profiles VALUES('profile-member','Alex','mint',?)", time.Now().Unix()); err != nil {
		t.Fatal(err)
	}
	owner := &http.Cookie{Name: "tally_profile", Value: "profile-admin"}
	member := &http.Cookie{Name: "tally_profile", Value: "profile-member"}
	filename, err := s.Backup.Create(context.Background(), "manual")
	if err != nil {
		t.Fatal(err)
	}
	var id string
	_ = s.DB.QueryRow("SELECT id FROM backup_records WHERE filename=?", filename).Scan(&id)
	expect(t, request(t, h, "POST", "/api/backups/"+id+"/restore", map[string]any{}, member), 403)
	expect(t, request(t, h, "DELETE", "/api/backups/"+id, nil, member), 403)

	if _, err = s.DB.Exec("UPDATE profiles SET display_name='Keep current' WHERE id='profile-admin'"); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(s.Backup.Path, filename), []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	expect(t, request(t, h, "POST", "/api/backups/"+id+"/restore", map[string]any{}, owner), 409)
	var name string
	_ = s.DB.QueryRow("SELECT display_name FROM profiles WHERE id='profile-admin'").Scan(&name)
	if name != "Keep current" {
		t.Fatal("failed restore changed current state")
	}
	logs := request(t, h, "GET", "/api/logs?q=restore%20failed", nil, owner)
	expect(t, logs, 200)
	if !strings.Contains(logs.Body.String(), "backup: restore failed") || !strings.Contains(logs.Body.String(), "job_failed") {
		t.Fatal("restore failure was not saved in logs", logs.Body.String())
	}
	inbox := request(t, h, "GET", "/api/inbox", nil, owner)
	expect(t, inbox, 200)
	if !strings.Contains(inbox.Body.String(), "backup: restore failed") {
		t.Fatal("restore failure was not shown in notifications", inbox.Body.String())
	}
}
