package notifications

import (
	"context"
	"testing"
	"time"

	"github.com/CarlFlo/tally/internal/database"
)

func TestNextDelayUsesPendingDeliveryAndIdleFallback(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	service := &Service{DB: db}
	now := time.Now().Truncate(time.Second)

	if got := service.nextDelay(ctx, now); got != notificationIdleInterval {
		t.Fatalf("idle delay = %v, want %v", got, notificationIdleInterval)
	}

	if _, err = db.Exec(`INSERT INTO notification_outbox(event_key,event,message,occurred_at,available_at) VALUES('soon','system_error','test',?,?)`, now.Unix(), now.Add(12*time.Second).Unix()); err != nil {
		t.Fatal(err)
	}
	got := service.nextDelay(ctx, now)
	if got < 11*time.Second || got > 13*time.Second {
		t.Fatalf("pending delay = %v, want about 12s", got)
	}

	if _, err = db.Exec("UPDATE notification_outbox SET available_at=?", now.Add(-time.Second).Unix()); err != nil {
		t.Fatal(err)
	}
	if got = service.nextDelay(ctx, now); got != time.Second {
		t.Fatalf("due delay = %v, want 1s", got)
	}
}
