package metadata

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/CarlFlo/mediaManager/internal/database"
)

type queueTV struct {
	before func()
	fail   bool
	calls  int
}

func (p *queueTV) SearchShows(context.Context, string) ([]SearchResult, error) { return nil, nil }

func (p *queueTV) GetShow(_ context.Context, id string) (*Show, error) {
	p.calls++
	if p.before != nil {
		p.before()
		p.before = nil
	}
	if p.fail {
		return nil, errors.New("fixture unavailable")
	}
	n, _ := strconv.Atoi(id)
	return &Show{ID: n, Name: "Queued show"}, nil
}

func (p *queueTV) GetEpisodes(context.Context, string) ([]Episode, error) { return []Episode{}, nil }

func queueService(t *testing.T) (*Service, *queueTV) {
	t.Helper()
	db, e := database.Open(context.Background(), t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close() })
	if _, e = db.Exec("INSERT INTO profiles(id,display_name,avatar,created_at) VALUES('profile-a','Fixture','violet',1)"); e != nil {
		t.Fatal(e)
	}
	p := &queueTV{}
	return &Service{DB: db, Provider: p}, p
}

func TestQueuedFollowUndoWhileImportingAndRetry(t *testing.T) {
	ctx := context.Background()
	s, p := queueService(t)
	if e := s.QueueFollow(ctx, "profile-a", 7, "Queued show", true); e != nil {
		t.Fatal(e)
	}
	if p.calls != 0 {
		t.Fatal("enqueue performed metadata work")
	}
	p.before = func() {
		if e := s.QueueFollow(ctx, "profile-a", 7, "Queued show", false); e != nil {
			t.Error(e)
		}
	}
	if _, e := s.ProcessNext(ctx); e != nil {
		t.Fatal(e)
	}
	var count int
	s.DB.QueryRow("SELECT COUNT(*) FROM profile_shows").Scan(&count)
	if count != 0 {
		t.Fatal("older add ignored newer undo")
	}
	if _, e := s.ProcessNext(ctx); e != nil {
		t.Fatal(e)
	}
	var status string
	s.DB.QueryRow("SELECT status FROM show_actions").Scan(&status)
	if status != "done" {
		t.Fatal("undo was not completed")
	}
	p.fail = true
	s.QueueFollow(ctx, "profile-a", 8, "Failing show", true)
	if _, e := s.ProcessNext(ctx); e != nil {
		t.Fatal(e)
	}
	s.DB.QueryRow("SELECT status FROM show_actions WHERE external_id=8").Scan(&status)
	if status != "failed" {
		t.Fatal("failed import not recorded")
	}
	p.fail = false
	s.QueueFollow(ctx, "profile-a", 8, "Failing show", true)
	s.ProcessNext(ctx)
	s.DB.QueryRow("SELECT COUNT(*) FROM profile_shows").Scan(&count)
	if count != 1 {
		t.Fatal("retry did not follow the show")
	}
}
