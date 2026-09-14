package api

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CarlFlo/mediaManager/internal/backup"
)

func TestBackupDownloadRequiresOwnerAndConfinesArchivePaths(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	service := backup.Service{DB: s.DB, DataDir: s.Config.DataDir, Path: filepath.Join(s.Config.DataDir, "backups"), Keep: 2}
	name, err := service.Create(context.Background(), "manual")
	if err != nil {
		t.Fatal(err)
	}
	var id string
	if err = s.DB.QueryRow("SELECT id FROM backup_records WHERE filename=?", name).Scan(&id); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(service.Path, name))
	if err != nil {
		t.Fatal(err)
	}
	result := request(t, h, "GET", "/api/backups/"+id+"/download", nil)
	expect(t, result, 200)
	if result.Body.String() != string(data) || !strings.Contains(result.Header().Get("Content-Disposition"), "attachment;") || result.Header().Get("Content-Type") != "application/zip" {
		t.Fatal("archive response is incorrect")
	}
	if _, err = s.DB.Exec("INSERT INTO profiles VALUES('user1','Alex','mint',1)"); err != nil {
		t.Fatal(err)
	}
	owner := &http.Cookie{Name: "tally_profile", Value: "user0"}
	member := &http.Cookie{Name: "tally_profile", Value: "user1"}
	expect(t, request(t, h, "GET", "/api/backups/"+id+"/download", nil, member), 403)
	if _, err = s.DB.Exec(`INSERT INTO backup_records VALUES('escape','../app.db','manual',1,1,1);
 INSERT INTO backup_records VALUES('failed','failed.zip','manual',1,1,0);
 INSERT INTO job_runs(id,job_key,trigger,started_at,status) VALUES('failed-job','backup','manual',1,'failed');`); err != nil {
		t.Fatal(err)
	}
	expect(t, request(t, h, "GET", "/api/backups/escape/download", nil, owner), 400)
	expect(t, request(t, h, "GET", "/api/backups/failed/download", nil, owner), 409)
	expect(t, request(t, h, "GET", "/api/backups/missing/download", nil, owner), 404)
	listing := request(t, h, "GET", "/api/backups", nil, owner)
	expect(t, listing, 200)
	if !strings.Contains(listing.Body.String(), "failed-job") {
		t.Fatal("failed backup absent")
	}
	if err = os.Symlink(filepath.Join(s.Config.DataDir, "app.db"), filepath.Join(service.Path, "outside.zip")); err == nil {
		if _, err = s.DB.Exec("INSERT INTO backup_records VALUES('symlink','outside.zip','manual',1,1,1)"); err != nil {
			t.Fatal(err)
		}
		expect(t, request(t, h, "GET", "/api/backups/symlink/download", nil, owner), 404)
	}
}
