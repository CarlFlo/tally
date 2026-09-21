package api

import (
	"context"
	"testing"

	"github.com/CarlFlo/tally/internal/database"
)

func TestFollowAccessPreservesDatabaseFailures(t *testing.T) {
	db, err := database.Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{DB: db}

	follows, err := s.follows(context.Background(), "missing-profile", "missing-show")
	if err != nil || follows {
		t.Fatalf("missing follow = %t, %v; want false, nil", follows, err)
	}

	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = s.follows(context.Background(), "profile", "show"); err == nil {
		t.Fatal("database failure was collapsed into a missing follow")
	}
}
