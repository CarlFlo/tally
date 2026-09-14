package providers

import (
	"context"
	"testing"

	"github.com/CarlFlo/mediaManager/internal/database"
)

func control(t *testing.T) *Coordinator {
	t.Helper()
	dir := t.TempDir()
	db, e := database.Open(context.Background(), dir)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close() })
	c, e := New(context.Background(), db, dir, 2, 0)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { c.Close() })
	return c
}
