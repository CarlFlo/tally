package notifications

import (
	"context"
	"time"

	"github.com/CarlFlo/mediaManager/internal/activity"
)

func (s *Service) collectBellReleases(ctx context.Context, now time.Time) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "INSERT INTO notification_state(key,value) VALUES('bell_release_cursor',?) ON CONFLICT(key) DO NOTHING", now.Unix()); err != nil {
		return err
	}
	last, err := cursor(ctx, tx, "bell_release_cursor")
	if err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, `SELECT p.profile_id,e.id,s.name,unixepoch(e.airstamp)
 FROM episodes e JOIN shows s ON s.id=e.show_id JOIN profile_shows p ON p.show_id=e.show_id
 WHERE unixepoch(e.airstamp)>? AND unixepoch(e.airstamp)<=? ORDER BY unixepoch(e.airstamp),e.id,p.profile_id LIMIT 100`, last, now.Unix())
	if err != nil {
		return err
	}
	defer rows.Close()
	count := 0
	changedProfiles := map[string]struct{}{}
	for rows.Next() {
		var profile, id, show string
		var released int64
		if err = rows.Scan(&profile, &id, &show, &released); err != nil {
			return err
		}
		if err = activity.Record(ctx, tx, activity.Event{Action: "episode_released", Profile: profile, ShowID: id, ShowName: show, Message: show + " has a new episode available."}); err != nil {
			return err
		}
		changedProfiles[profile] = struct{}{}
		last, count = released, count+1
	}
	if err = rows.Err(); err != nil {
		return err
	}
	if count < 100 {
		last = now.Unix()
	}
	if err = saveCursor(ctx, tx, "bell_release_cursor", last); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	for profile := range changedProfiles {
		s.changedProfile(profile, "logs", "inbox")
		s.changedProfile("", "logs", "inbox")
	}
	return nil
}
