package notifications

import (
	"context"
	"time"

	"github.com/CarlFlo/mediaManager/internal/activity"
	"github.com/CarlFlo/mediaManager/internal/settings"
)

func (s *Service) deliver(ctx context.Context, now time.Time) error {
	rows, err := s.DB.Rows(ctx, "SELECT * FROM notification_outbox WHERE status='pending' AND available_at<=? ORDER BY CASE WHEN event IN ('system_error','job_failed') THEN 0 ELSE 1 END,available_at,id LIMIT 10", now.Unix())
	if err != nil {
		return err
	}
	statsChanged := false
	adminActivityChanged := false
	for _, row := range rows {
		var config settings.Webhook
		revision, loadErr := (settings.Store{DB: s.DB}).Load(ctx, "notifications", &config)
		if loadErr != nil {
			return loadErr
		}
		event := row["event"].(string)
		status := "skipped"
		if config.Subscribed(event) {
			message := Message{Event: event, Key: row["source_key"].(string), Level: row["level"].(string), Text: row["message"].(string), Show: row["show_name"].(string), Time: time.Unix(row["occurred_at"].(int64), 0)}
			// Mark before sending: automatic retries after an ambiguous response could duplicate alerts.
			claim, claimErr := s.DB.ExecContext(ctx, "UPDATE notification_outbox SET status='failed' WHERE id=? AND status='pending' AND EXISTS(SELECT 1 FROM application_settings WHERE key='notifications' AND revision=?)", row["id"], revision)
			if claimErr != nil {
				return claimErr
			}
			count, err := claim.RowsAffected()
			if err != nil {
				return err
			}
			if count == 0 {
				continue
			}
			status = "sent"
			statsChanged = true
			if sendErr := Send(ctx, s.Requester, config, message); sendErr != nil {
				status = "failed"
				if err = activity.Record(ctx, s.DB, activity.Event{Action: "notification_failed", Message: "Notification delivery failed. Check Notification Services and test the connection."}); err != nil {
					return err
				}
				adminActivityChanged = true
			}
		}
		if _, err = s.DB.ExecContext(ctx, "UPDATE notification_outbox SET status=? WHERE id=?", status, row["id"]); err != nil {
			return err
		}
	}
	if statsChanged {
		s.changedProfile("", "statistics")
	}
	if adminActivityChanged {
		s.changedProfile("", "logs", "inbox")
	}
	return nil
}
