package api

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/CarlFlo/mediaManager/internal/jobs"
)

func TestManualBackupJobCompletesForGeneratedAdministrator(t *testing.T) {
	s, h, _ := testServer(t, "disabled")

	if _, err := s.DB.Exec("DELETE FROM profiles WHERE id='profile-admin'"); err != nil {
		t.Fatal(err)
	}
	adminID := "0123456789abcdef0123456789abcdef"
	if _, err := s.DB.Exec("INSERT INTO profiles(id,display_name,avatar,created_at) VALUES(?,?,?,?)", adminID, "My profile", "violet", time.Now().Unix()); err != nil {
		t.Fatal(err)
	}

	s.Config.JobConcurrency = 2
	s.Config.JobRuntime = 10 * time.Second
	s.Jobs = jobs.New(context.Background(), s.DB, s.Config, s.Metadata, s.Control, s.Backup)

	owner := &http.Cookie{Name: "tally_profile", Value: adminID}
	started := request(t, h, "POST", "/api/backups", map[string]any{}, owner)
	expect(t, started, http.StatusAccepted)

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var count int
		if err := s.DB.QueryRow("SELECT COUNT(*) FROM backup_records WHERE kind='manual' AND verified=1").Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count > 0 {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}

	var status, jobErr string
	_ = s.DB.QueryRow("SELECT status,error FROM job_runs WHERE job_key='backup' ORDER BY started_at DESC LIMIT 1").Scan(&status, &jobErr)
	t.Fatalf("manual backup did not complete: status=%q error=%q", status, jobErr)
}
