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

	"github.com/CarlFlo/mediaManager/internal/appversion"
	"github.com/CarlFlo/mediaManager/internal/database"
)

func TestBackupManagementListsVersionRestoresLiveAndDeletes(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	filename, err := s.Backup.Create(context.Background(), "manual")
	if err != nil {
		t.Fatal(err)
	}
	var id string
	if err = s.DB.QueryRow("SELECT id FROM backup_records WHERE filename=?", filename).Scan(&id); err != nil {
		t.Fatal(err)
	}
	owner := &http.Cookie{Name: "tally_profile", Value: "user0"}
	listing := request(t, h, "GET", "/api/backups", nil, owner)
	expect(t, listing, 200)
	for _, want := range []string{`"kind":"manual"`, fmt.Sprintf(`"schema":%d`, database.Version), fmt.Sprintf(`"app_version":"%s"`, appversion.Version), `"compatible":true`} {
		if !strings.Contains(listing.Body.String(), want) {
			t.Fatalf("backup metadata missing %s: %s", want, listing.Body.String())
		}
	}
	if _, err = s.DB.Exec("UPDATE profiles SET display_name='Changed after backup' WHERE id='user0'"); err != nil {
		t.Fatal(err)
	}
	expect(t, request(t, h, "POST", "/api/backups/"+id+"/restore", map[string]any{}, owner), 200)
	var name string
	if err = s.DB.QueryRow("SELECT display_name FROM profiles WHERE id='user0'").Scan(&name); err != nil || name != "My profile" {
		t.Fatal("restore did not apply without reopening the database", name, err)
	}
	expect(t, request(t, h, "DELETE", "/api/backups/"+id, nil, owner), 200)
	if _, err = os.Stat(filepath.Join(s.Backup.Path, filename)); !os.IsNotExist(err) {
		t.Fatal("deleted backup file still exists")
	}
}

func TestBackupManagementRequiresOperatorAndFailedRestorePreservesState(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	if _, err := s.DB.Exec("INSERT INTO profiles VALUES('user1','Alex','mint',?)", time.Now().Unix()); err != nil {
		t.Fatal(err)
	}
	owner := &http.Cookie{Name: "tally_profile", Value: "user0"}
	member := &http.Cookie{Name: "tally_profile", Value: "user1"}
	filename, err := s.Backup.Create(context.Background(), "manual")
	if err != nil {
		t.Fatal(err)
	}
	var id string
	_ = s.DB.QueryRow("SELECT id FROM backup_records WHERE filename=?", filename).Scan(&id)
	expect(t, request(t, h, "POST", "/api/backups/"+id+"/restore", map[string]any{}, member), 403)
	expect(t, request(t, h, "DELETE", "/api/backups/"+id, nil, member), 403)

	if _, err = s.DB.Exec("UPDATE profiles SET display_name='Keep current' WHERE id='user0'"); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(s.Backup.Path, filename), []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	expect(t, request(t, h, "POST", "/api/backups/"+id+"/restore", map[string]any{}, owner), 409)
	var name string
	_ = s.DB.QueryRow("SELECT display_name FROM profiles WHERE id='user0'").Scan(&name)
	if name != "Keep current" {
		t.Fatal("failed restore changed current state")
	}
}
