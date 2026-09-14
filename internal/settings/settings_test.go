package settings

import (
	"context"
	"testing"

	"github.com/CarlFlo/mediaManager/internal/config"
	"github.com/CarlFlo/mediaManager/internal/database"
)

func TestLegacySettingsImportOnlyOnceAndPersist(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, e := database.Open(ctx, dir)
	if e != nil {
		t.Fatal(e)
	}
	store := Store{DB: db}
	t.Setenv("TORZNAB_URL", "http://fixture.invalid/api/v2.0/indexers/all/results/torznab/api")
	t.Setenv("TORZNAB_NAME", "Old indexer")
	t.Setenv("TORZNAB_API_KEY", "fixture-key")
	t.Setenv("WEBHOOK_URL", "http://webhook.invalid/hook")
	t.Setenv("BACKUP_ENABLED", "false")
	t.Setenv("BACKUP_KEEP", "7")
	c := store.LegacyDefaults(ctx, config.Config{})
	if len(c.TorznabProviders) != 1 || c.WebhookURL == "" || c.BackupEnabled || c.BackupKeep != 7 {
		t.Fatal("legacy UI values not imported")
	}
	if e = store.Ensure(ctx, c); e != nil {
		t.Fatal(e)
	}
	if _, e = store.Save(ctx, "search", Search{}, 1); e != nil {
		t.Fatal(e)
	}
	db.Close()
	db, e = database.Open(ctx, dir)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	store.DB = db
	t.Setenv("TORZNAB_PROVIDERS", "invalid")
	t.Setenv("JOB_METADATA_CRON", "invalid")
	t.Setenv("BACKUP_KEEP", "99")
	c = store.LegacyDefaults(ctx, config.Config{})
	if c.WebhookURL != "" || len(c.TorznabProviders) > 0 {
		t.Fatal("environment reapplied to saved settings")
	}
	if e = store.Ensure(ctx, c); e != nil {
		t.Fatal(e)
	}
	var backups Backups
	if _, err := store.Load(ctx, "backups", &backups); err != nil || backups.Keep != 7 {
		t.Fatal("saved retention overwritten", err)
	}
	var search Search
	store.Load(ctx, "search", &search)
	if search.Configured() {
		t.Fatal("restart enabled removed Jackett connection")
	}
	var webhook Webhook
	store.Load(ctx, "notifications", &webhook)
	if webhook.URL != "http://webhook.invalid/hook" {
		t.Fatal("webhook not persisted")
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
	if err = store.Ensure(ctx, config.Config{}); err != nil {
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
