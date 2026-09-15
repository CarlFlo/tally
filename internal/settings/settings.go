// Package settings stores editable deployment settings in the permanent database.
package settings

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/CarlFlo/tally/internal/activity"
	"github.com/CarlFlo/tally/internal/database"
)

type Store struct{ DB *database.Store }

var ErrConflict = errors.New("settings changed in another tab; reload before saving")

func (s Store) Load(ctx context.Context, key string, out any) (int64, error) {
	var raw string
	var rev int64
	e := s.DB.QueryRowContext(ctx, "SELECT data,revision FROM application_settings WHERE key=?", key).Scan(&raw, &rev)
	if e != nil {
		return 0, e
	}
	return rev, json.Unmarshal([]byte(raw), out)
}

func (s Store) Save(ctx context.Context, key string, value any, revision int64, actors ...string) (int64, error) {
	actor := ""
	if len(actors) > 0 {
		actor = actors[0]
	}
	raw, e := json.Marshal(value)
	if e != nil {
		return 0, e
	}
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return 0, e
	}
	defer tx.Rollback()
	var currentRaw string
	var currentRevision int64
	if e = tx.QueryRowContext(ctx, "SELECT data,revision FROM application_settings WHERE key=?", key).Scan(&currentRaw, &currentRevision); e != nil {
		return 0, e
	}
	if currentRevision != revision {
		return 0, ErrConflict
	}
	var current, next any
	if e = json.Unmarshal([]byte(currentRaw), &current); e != nil {
		return 0, e
	}
	if e = json.Unmarshal(raw, &next); e != nil {
		return 0, e
	}
	if reflect.DeepEqual(current, next) {
		return revision, nil
	}
	var wasEnabled bool
	if key == "notifications" {
		if e = tx.QueryRowContext(ctx, "SELECT COALESCE(json_extract(data,'$.enabled'),0) FROM application_settings WHERE key=?", key).Scan(&wasEnabled); e != nil {
			return 0, e
		}
	}
	res, e := tx.ExecContext(ctx, "UPDATE application_settings SET data=?,revision=revision+1 WHERE key=? AND revision=?", string(raw), key, revision)
	if e != nil {
		return 0, e
	}
	n, e := res.RowsAffected()
	if e != nil {
		return 0, e
	}
	if n != 1 {
		return 0, ErrConflict
	}
	if key == "notifications" {
		var config Webhook
		if e = json.Unmarshal(raw, &config); e != nil {
			return 0, e
		}
		if !config.Enabled {
			if _, e = tx.ExecContext(ctx, "UPDATE notification_outbox SET status='skipped' WHERE status='pending'"); e != nil {
				return 0, e
			}
		}
		// Changes take effect from now; events occurring while disabled are never replayed.
		if config.Enabled != wasEnabled {
			if _, e = tx.ExecContext(ctx, "UPDATE notification_state SET value=CASE key WHEN 'activity_cursor' THEN (SELECT COALESCE(MAX(id),0) FROM activity_log) ELSE unixepoch() END"); e != nil {
				return 0, e
			}
		}
	}
	if e = activity.Record(ctx, tx, activity.Event{Action: "settings_updated", Profile: actor, Message: "Updated " + key + " settings"}); e != nil {
		return 0, e
	}
	if e = tx.Commit(); e != nil {
		return 0, e
	}
	return revision + 1, nil
}
