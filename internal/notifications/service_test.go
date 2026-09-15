package notifications

import (
	"context"
	"testing"
	"time"

	"github.com/CarlFlo/tally/internal/database"
	"github.com/CarlFlo/tally/internal/settings"
)

func TestScheduledReleaseSurvivesRestartAndMasterSwitchSkipsPending(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec("INSERT INTO profiles(id,display_name,avatar,created_at) VALUES('profile-fixture','Fixture','violet',1)"); err != nil {
		t.Fatal(err)
	}
	store := settings.Store{DB: db}
	if err = store.Ensure(ctx); err != nil {
		t.Fatal(err)
	}
	saved := settings.Webhook{Enabled: true, URL: "http://fixture.invalid", Events: []string{"episode_released"}, DeliveryTime: "09:00", Timezone: "UTC"}
	revision, err := store.Save(ctx, "notifications", saved, 1)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 1, 2, 8, 0, 0, 0, time.UTC)
	_, err = db.Exec(`INSERT INTO shows(id,name) VALUES('show','Example'); INSERT INTO profile_shows(profile_id,show_id,added_at) VALUES('profile-fixture','show',1); INSERT INTO episodes(id,show_id,season,number,name,airstamp) VALUES('ep','show',1,1,'Pilot','2026-01-02T08:00:00Z'); INSERT INTO notification_state(key,value) VALUES('bell_release_cursor',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value; UPDATE notification_state SET value=? WHERE key='release_cursor'`, now.Add(-time.Minute).Unix(), now.Add(-time.Minute).Unix())
	if err != nil {
		t.Fatal(err)
	}
	r := &recorder{}
	s := &Service{DB: db, Requester: r}
	if err = s.Tick(ctx, now); err != nil {
		t.Fatal(err)
	}
	if len(r.requests) != 0 {
		t.Fatal("sent before delivery time")
	}
	// Recreate worker with the same persisted state, as on restart.
	s = &Service{DB: db, Requester: r}
	if err = s.Tick(ctx, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err = s.Tick(ctx, now.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if len(r.requests) != 1 {
		t.Fatal("release lost or repeated", len(r.requests))
	}
	var bellReleases int
	if err = db.QueryRow("SELECT COUNT(*) FROM activity_log WHERE action='episode_released'").Scan(&bellReleases); err != nil || bellReleases != 1 {
		t.Fatal("bell release activity missing or repeated", bellReleases, err)
	}
	if err = enqueue(ctx, db, "pending", Message{Event: "episode_released", Text: "Pending", Time: now}, now.Add(3*time.Hour)); err != nil {
		t.Fatal(err)
	}
	saved.Enabled = false
	revision, err = store.Save(ctx, "notifications", saved, revision)
	if err != nil {
		t.Fatal(err)
	}
	saved.Enabled = true
	if _, err = store.Save(ctx, "notifications", saved, revision); err != nil {
		t.Fatal(err)
	}
	if err = s.Tick(ctx, now.Add(4*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if len(r.requests) != 1 {
		t.Fatal("disabled queue replayed")
	}
	if err = QueueAlert(ctx, db, "job:metadata", "error", "Unsubscribed failure"); err != nil {
		t.Fatal(err)
	}
	var alerts int
	_ = db.QueryRow("SELECT COUNT(*) FROM notification_outbox WHERE event='job_failed'").Scan(&alerts)
	if alerts != 0 {
		t.Fatal("ignored subscription")
	}
}
