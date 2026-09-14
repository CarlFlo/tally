package notifications

import (
	"context"
	"database/sql"
	"time"

	"github.com/CarlFlo/mediaManager/internal/activity"
	"github.com/CarlFlo/mediaManager/internal/database"
	"github.com/CarlFlo/mediaManager/internal/settings"
)

func enqueue(ctx context.Context, db activity.Execer, unique string, message Message, at time.Time) error {
	_, err := db.ExecContext(ctx, `INSERT INTO notification_outbox(event_key,event,message,show_name,occurred_at,available_at,source_key,level) VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(event_key) DO NOTHING`, unique, message.Event, message.Text, message.Show, message.Time.Unix(), at.Unix(), message.Key, message.Level)
	return err
}

func QueueAlert(ctx context.Context, db *database.Store, key, level, text string) error {
	var config settings.Webhook
	if _, err := (settings.Store{DB: db}).Load(ctx, "notifications", &config); err != nil {
		return err
	}
	event := "system_error"
	if len(key) > 4 && key[:4] == "job:" {
		event = "job_failed"
	}
	if !config.Subscribed(event) {
		return nil
	}
	now := time.Now()
	return enqueue(ctx, db, "alert:"+database.ID(), Message{Event: event, Key: key, Level: level, Text: text, Time: now}, now)
}

func cursor(ctx context.Context, tx *sql.Tx, key string) (int64, error) {
	var value int64
	err := tx.QueryRowContext(ctx, "SELECT value FROM notification_state WHERE key=?", key).Scan(&value)
	return value, err
}

func saveCursor(ctx context.Context, tx *sql.Tx, key string, value int64) error {
	_, err := tx.ExecContext(ctx, "UPDATE notification_state SET value=? WHERE key=?", value, key)
	return err
}
