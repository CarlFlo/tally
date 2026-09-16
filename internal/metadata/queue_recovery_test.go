package metadata

import (
	"context"
	"testing"

	"github.com/CarlFlo/tally/internal/database"
)

func TestQueueSurvivesReopenAndSharesMetadataAcrossProfiles(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, e := database.Open(ctx, dir)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec("INSERT INTO profiles(id,display_name,avatar,created_at) VALUES('profile-a','Fixture','violet',1)"); e != nil {
		t.Fatal(e)
	}
	p := &queueTV{}
	s := &Service{DB: db, Provider: p}
	if e = s.QueueFollow(ctx, "profile-a", 7, "Saved queue", true); e != nil {
		t.Fatal(e)
	}
	db.Close()
	db, e = database.Open(ctx, dir)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	s.DB = db
	if _, e = s.ProcessNext(ctx); e != nil {
		t.Fatal(e)
	}
	db.Exec("INSERT INTO profiles(id,display_name,avatar,created_at) VALUES('profile-b','Second','mint',1)")
	if e = s.QueueFollow(ctx, "profile-b", 7, "Saved queue", true); e != nil {
		t.Fatal(e)
	}
	if _, e = s.ProcessNext(ctx); e != nil {
		t.Fatal(e)
	}
	if p.calls != 1 {
		t.Fatal("second profile did not reuse metadata")
	}
	s.QueueFollow(ctx, "profile-a", 7, "Saved queue", false)
	s.ProcessNext(ctx)
	var profile string
	if e = db.QueryRow("SELECT profile_id FROM profile_shows").Scan(&profile); e != nil || profile != "profile-b" {
		t.Fatal("undo affected another profile", e)
	}
}

func TestQueueCommitFailureCanBeRetried(t *testing.T) {
	ctx := context.Background()
	s, _ := queueService(t)
	if _, e := s.DB.Exec(`CREATE TRIGGER reject_follow BEFORE INSERT ON profile_shows BEGIN SELECT RAISE(ABORT, 'fixture storage failure'); END`); e != nil {
		t.Fatal(e)
	}
	if e := s.QueueFollow(ctx, "profile-a", 7, "Queued show", true); e != nil {
		t.Fatal(e)
	}
	if _, e := s.ProcessNext(ctx); e == nil {
		t.Fatal("expected a failed transaction")
	}
	var status string
	if e := s.DB.QueryRow("SELECT status FROM show_actions").Scan(&status); e != nil || status != "failed" {
		t.Fatal("failed transaction stranded the queue", status, e)
	}
	if _, e := s.DB.Exec("DROP TRIGGER reject_follow"); e != nil {
		t.Fatal(e)
	}
	if e := s.QueueFollow(ctx, "profile-a", 7, "Queued show", true); e != nil {
		t.Fatal(e)
	}
	if _, e := s.ProcessNext(ctx); e != nil {
		t.Fatal(e)
	}
	var count int
	if e := s.DB.QueryRow("SELECT COUNT(*) FROM profile_shows").Scan(&count); e != nil || count != 1 {
		t.Fatal("retry failed", e)
	}
}
