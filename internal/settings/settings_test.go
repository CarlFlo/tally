package settings

import (
	"context"
	"testing"

	"github.com/CarlFlo/mediaManager/internal/database"
)

func TestEnsureSeedsSQLiteDefaults(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := Store{DB: db}
	if err = store.Ensure(ctx); err != nil {
		t.Fatal(err)
	}
	var search Search
	if _, err = store.Load(ctx, "search", &search); err != nil {
		t.Fatal(err)
	}
	if search.Configured() {
		t.Fatal("search defaults should not configure Jackett")
	}
	var webhook Webhook
	if _, err = store.Load(ctx, "notifications", &webhook); err != nil {
		t.Fatal(err)
	}
	if webhook.Enabled || webhook.URL != "" {
		t.Fatal("notification defaults should be disabled")
	}
	var backups Backups
	if _, err = store.Load(ctx, "backups", &backups); err != nil || backups.Keep != 10 {
		t.Fatal("unexpected backup default", err)
	}
}

func TestSaveUnchangedSettingsDoesNotCreateActivity(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := Store{DB: db}
	if err = store.Ensure(ctx); err != nil {
		t.Fatal(err)
	}
	var saved Backups
	revision, err := store.Load(ctx, "backups", &saved)
	if err != nil {
		t.Fatal(err)
	}
	next, err := store.Save(ctx, "backups", saved, revision)
	if err != nil || next != revision {
		t.Fatalf("unchanged save changed revision: %d, %v", next, err)
	}
	var entries int
	if err = db.QueryRow("SELECT COUNT(*) FROM activity_log WHERE action='settings_updated'").Scan(&entries); err != nil {
		t.Fatal(err)
	}
	if entries != 0 {
		t.Fatalf("unchanged save created %d activity entries", entries)
	}
}
