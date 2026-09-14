package notifications

import (
	"context"
	"testing"
	"time"

	"github.com/CarlFlo/mediaManager/internal/config"
	"github.com/CarlFlo/mediaManager/internal/database"
	"github.com/CarlFlo/mediaManager/internal/settings"
)

func TestMasterSwitchInvalidatesAlreadySelectedPendingMessages(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := settings.Store{DB: db}
	if err = store.Ensure(ctx, config.Config{}); err != nil {
		t.Fatal(err)
	}
	saved := settings.Webhook{Enabled: true, URL: "http://fixture.invalid/private", Events: []string{"system_error", "episode_released"}}
	revision, err := store.Save(ctx, "notifications", saved, 1)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	message := Message{Event: "episode_released", Text: "Earlier media", Time: now}
	if err = enqueue(ctx, db, "media", message, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	message.Event = "system_error"
	message.Text = "Immediate error"
	if err = enqueue(ctx, db, "error", message, now); err != nil {
		t.Fatal(err)
	}
	r := &recorder{hook: func() {
		saved.Enabled = false
		revision, err = store.Save(ctx, "notifications", saved, revision)
		if err != nil {
			t.Fatal(err)
		}
		saved.Enabled = true
		_, err = store.Save(ctx, "notifications", saved, revision)
		if err != nil {
			t.Fatal(err)
		}
	}}
	s := Service{DB: db, Requester: r}
	if err = s.deliver(ctx, now); err != nil {
		t.Fatal(err)
	}
	if len(r.requests) != 1 {
		t.Fatal("disabled messages were replayed", len(r.requests))
	}
	var status string
	if err = db.QueryRow("SELECT status FROM notification_outbox WHERE event_key='error'").Scan(&status); err != nil || status != "sent" {
		t.Fatal("immediate error did not take priority", status, err)
	}
	if err = db.QueryRow("SELECT status FROM notification_outbox WHERE event_key='media'").Scan(&status); err != nil || status != "skipped" {
		t.Fatal("master switch did not skip pending media", status, err)
	}
	// Failed deliveries are visible and never retried automatically.
	r.status = 401
	message.Event = "system_error"
	if err = enqueue(ctx, db, "failure", message, now); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err = s.deliver(ctx, now); err != nil {
			t.Fatal(err)
		}
	}
	if len(r.requests) != 2 {
		t.Fatal("ambiguous failure was retried")
	}
	var count int
	if err = db.QueryRow("SELECT COUNT(*) FROM activity_log WHERE action='notification_failed' AND message NOT LIKE '%fixture.invalid%'").Scan(&count); err != nil || count != 1 {
		t.Fatal("missing safe failure activity", err)
	}
}
