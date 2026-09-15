package notifications

import (
	"context"
	"testing"

	"github.com/CarlFlo/mediaManager/internal/database"
	"github.com/CarlFlo/mediaManager/internal/settings"
)

func TestProfileLifecycleActivityUsesSingleSubscriptionEvent(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	store := settings.Store{DB: db}
	if err = store.Ensure(ctx); err != nil {
		t.Fatal(err)
	}
	config := settings.Webhook{
		Enabled: true,
		Events:  []string{"profile_access_changed"},
	}
	if _, err = store.Save(ctx, "notifications", config, 1); err != nil {
		t.Fatal(err)
	}

	for _, action := range []string{"profile_created", "admin_granted", "admin_revoked", "profile_deleted"} {
		if _, err = db.Exec(
			"INSERT INTO activity_log(action,message,created_at) VALUES(?,?,1)",
			action,
			action+" fixture",
		); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = db.Exec(
		"INSERT INTO activity_log(action,message,created_at) VALUES('show_added','unrelated fixture',1)",
	); err != nil {
		t.Fatal(err)
	}

	service := Service{DB: db}
	if err = service.collectActivity(ctx, config); err != nil {
		t.Fatal(err)
	}

	var count int
	if err = db.QueryRow(
		"SELECT COUNT(*) FROM notification_outbox WHERE event='profile_access_changed'",
	).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 4 {
		t.Fatalf("queued %d profile access notifications, want 4", count)
	}
	if err = db.QueryRow("SELECT COUNT(*) FROM notification_outbox").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 4 {
		t.Fatal("unsubscribed activity was queued")
	}
}
