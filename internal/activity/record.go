// Package activity stores safe, durable application events independently of profile lifetimes.
package activity

import (
	"context"
	"database/sql"
	"time"
)

type Execer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type Event struct {
	Action   string
	Profile  string
	ShowID   string
	ShowName string
	Message  string
}

// Record accepts a transaction so the event commits together with its mutation.
// Callers supply human-readable descriptions, never credentials or request bodies.
func Record(ctx context.Context, db Execer, event Event) error {
	_, err := db.ExecContext(ctx, `INSERT INTO activity_log(action,profile_id,show_id,show_name,message,created_at) VALUES(?,?,?,?,?,?)`, event.Action, event.Profile, event.ShowID, event.ShowName, event.Message, time.Now().Unix())
	return err
}
