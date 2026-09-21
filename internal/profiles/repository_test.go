package profiles

import (
	"context"
	"errors"
	"testing"

	"github.com/CarlFlo/tally/internal/database"
)

func TestRoleRepositoryPreservesDatabaseFailures(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repo := Repository{DB: db, Limit: 3}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	if err = repo.SetAdmin(ctx, "actor", "target", true); err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("database failure was misclassified as not found: %v", err)
	}
	if _, _, err = repo.Delete(ctx, "actor", "target"); err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("database failure was misclassified as not found: %v", err)
	}
}
