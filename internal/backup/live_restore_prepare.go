package backup

import (
	"context"
	"time"

	"github.com/CarlFlo/mediaManager/internal/database"
	"github.com/CarlFlo/mediaManager/internal/scheduling"
)

func prepareStagedDatabase(ctx context.Context, dir string) error {
	db, err := database.Open(ctx, dir)
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err = db.ExecContext(ctx, "UPDATE show_actions SET status='queued' WHERE status='running'"); err != nil {
		return err
	}
	rows, err := db.QueryContext(ctx, "SELECT key,schedule,enabled,paused FROM jobs")
	if err != nil {
		return err
	}
	type schedule struct {
		key, spec       string
		enabled, paused bool
	}
	var saved []schedule
	for rows.Next() {
		var item schedule
		if err = rows.Scan(&item.key, &item.spec, &item.enabled, &item.paused); err != nil {
			rows.Close()
			return err
		}
		saved = append(saved, item)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err = rows.Close(); err != nil {
		return err
	}
	for _, item := range saved {
		parsed, parseErr := scheduling.Parse(item.spec)
		if parseErr != nil {
			return parseErr
		}
		var next int64
		if item.enabled && !item.paused {
			next = parsed.Next(time.Now().UTC()).Unix()
		}
		if _, err = db.ExecContext(ctx, "UPDATE jobs SET next_run=? WHERE key=?", next, item.key); err != nil {
			return err
		}
	}
	if err = database.Validate(ctx, db); err != nil {
		return err
	}
	if err = database.ValidateSchema(ctx, db); err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)")
	return err
}
