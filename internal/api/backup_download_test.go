package api

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CarlFlo/tally/internal/backup"
)

func TestBackupDownloadRequiresOwnerAndConfinesArchivePaths(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	service := backup.Service{DB: s.DB, DataDir: s.Config.DataDir, Path: filepath.Join(s.Config.DataDir, "backups"), Keep: 2}
	name, err := service.Create(context.Background(), "manual")
	if err != nil {
		t.Fatal(err)
	}
	id := backupIDByName(t, &service, name)
	data, err := os.ReadFile(filepath.Join(service.Path, name))
	if err != nil {
		t.Fatal(err)
	}
	result := request(t, h, "GET", "/api/backups/"+id+"/download", nil)
	expect(t, result, 200)
	if result.Body.String() != string(data) || !strings.Contains(result.Header().Get("Content-Disposition"), "attachment;") || result.Header().Get("Content-Type") != "application/zip" {
		t.Fatal("archive response is incorrect")
	}
	if _, err = s.DB.Exec("INSERT INTO profiles(id,display_name,avatar,created_at) VALUES('profile-member','Alex','mint',1)"); err != nil {
		t.Fatal(err)
	}
	owner := &http.Cookie{Name: "tally_profile", Value: "profile-admin"}
	member := &http.Cookie{Name: "tally_profile", Value: "profile-member"}
	expect(t, request(t, h, "GET", "/api/backups/"+id+"/download", nil, member), 403)

	corruptName := "copied-corrupt.zip"
	if err = os.WriteFile(filepath.Join(service.Path, corruptName), []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	corruptID := backupIDByName(t, &service, corruptName)
	listing := request(t, h, "GET", "/api/backups", nil, owner)
	expect(t, listing, 200)
	if !strings.Contains(listing.Body.String(), corruptName) || !strings.Contains(listing.Body.String(), "Archive metadata could not be read") {
		t.Fatal("filesystem zip was not exposed with validation state", listing.Body.String())
	}
	expect(t, request(t, h, "GET", "/api/backups/"+corruptID+"/download", nil, owner), 200)
	expect(t, request(t, h, "GET", "/api/backups/missing/download", nil, owner), 404)

	if err = os.Symlink(filepath.Join(s.Config.DataDir, "app.db"), filepath.Join(service.Path, "outside.zip")); err == nil {
		listing = request(t, h, "GET", "/api/backups", nil, owner)
		expect(t, listing, 200)
		if strings.Contains(listing.Body.String(), "outside.zip") {
			t.Fatal("symlink archive leaked into filesystem inventory")
		}
	}
}
