package notifications

import (
	"context"
	"fmt"
	"time"

	"github.com/CarlFlo/tally/internal/settings"
)

func (s *Service) collectReleases(ctx context.Context, config settings.Webhook, now time.Time) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	last, err := cursor(ctx, tx, "release_cursor")
	if err != nil {
		return err
	}
	if !config.Subscribed("episode_released") {
		if err = saveCursor(ctx, tx, "release_cursor", now.Unix()); err != nil {
			return err
		}
		return tx.Commit()
	}
	// Confirmed timestamps avoid inventing a release time for date-only metadata.
	rows, err := tx.QueryContext(ctx, `SELECT e.id,s.name,e.name,e.season,e.number,unixepoch(e.airstamp) FROM episodes e JOIN shows s ON s.id=e.show_id
 WHERE unixepoch(e.airstamp)>=? AND unixepoch(e.airstamp)<=?
 AND EXISTS(SELECT 1 FROM profile_shows p WHERE p.show_id=e.show_id)
 AND NOT EXISTS(SELECT 1 FROM notification_outbox n WHERE n.event_key='release:'||e.id)
 ORDER BY unixepoch(e.airstamp),e.id LIMIT 100`, last, now.Unix())
	if err != nil {
		return err
	}
	type release struct {
		id, show, title string
		season, number  int
		at              int64
	}
	entries := []release{}
	for rows.Next() {
		var e release
		if err = rows.Scan(&e.id, &e.show, &e.title, &e.season, &e.number, &e.at); err != nil {
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
	location, err := time.LoadLocation(config.Timezone)
	if err != nil {
		return err
	}
	for _, e := range entries {
		released := time.Unix(e.at, 0)
		msg := Message{Event: "episode_released", Show: e.show, Text: fmt.Sprintf("%s - S%02dE%02d: %s is available", e.show, e.season, e.number, e.title), Time: released, Level: "info"}
		if err = enqueue(ctx, tx, "release:"+e.id, msg, DeliveryAt(released, config.DeliveryTime, location)); err != nil {
			return err
		}
		last = e.at
	}
	if len(entries) < 100 {
		last = now.Unix()
	}
	if err = saveCursor(ctx, tx, "release_cursor", last); err != nil {
		return err
	}
	return tx.Commit()
}
