package library

import (
	"context"
	"database/sql"
	"time"

	"github.com/CarlFlo/mediaManager/internal/activity"
)

// SetFollow records actual membership changes in the caller's transaction.
func SetFollow(ctx context.Context, tx *sql.Tx, profile, show string, desired bool) error {
	var result sql.Result
	var err error
	action, verb := "show_removed", "Removed "
	if desired {
		result, err = tx.ExecContext(ctx, "INSERT INTO profile_shows(profile_id,show_id,added_at) VALUES(?,?,?) ON CONFLICT DO NOTHING", profile, show, time.Now().Unix())
		action, verb = "show_added", "Added "
	} else {
		result, err = tx.ExecContext(ctx, "DELETE FROM profile_shows WHERE profile_id=? AND show_id=?", profile, show)
	}
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil || count == 0 {
		return err
	}
	var name string
	if err = tx.QueryRowContext(ctx, "SELECT name FROM shows WHERE id=?", show).Scan(&name); err != nil {
		return err
	}
	return activity.Record(ctx, tx, activity.Event{Action: action, Profile: profile, ShowID: show, ShowName: name, Message: verb + name})
}
