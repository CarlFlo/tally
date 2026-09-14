package notifications

import (
	"context"
	"fmt"
	"time"

	"github.com/CarlFlo/mediaManager/internal/settings"
)

func (s *Service) collectActivity(ctx context.Context, config settings.Webhook) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	last, err := cursor(ctx, tx, "activity_cursor")
	if err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, "SELECT id,action,message,show_name,created_at FROM activity_log WHERE id>? ORDER BY id LIMIT 100", last)
	if err != nil {
		return err
	}
	type entry struct {
		id, at            int64
		event, text, show string
	}
	entries := []entry{}
	for rows.Next() {
		var e entry
		if err = rows.Scan(&e.id, &e.event, &e.text, &e.show, &e.at); err != nil {
			rows.Close()
			return err
		}
		entries = append(entries, e)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, e := range entries {
		if config.Subscribed(e.event) && e.event != "system_error" && e.event != "job_failed" && e.event != "episode_released" {
			msg := Message{Event: e.event, Text: e.text, Show: e.show, Time: time.Unix(e.at, 0), Level: "info"}
			if err = enqueue(ctx, tx, fmt.Sprintf("activity:%d", e.id), msg, msg.Time); err != nil {
				return err
			}
		}
		last = e.id
	}
	if err = saveCursor(ctx, tx, "activity_cursor", last); err != nil {
		return err
	}
	return tx.Commit()
}
