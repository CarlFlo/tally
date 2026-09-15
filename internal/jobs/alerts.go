package jobs

import (
	"log/slog"
	"strings"
	"time"

	"github.com/CarlFlo/mediaManager/internal/activity"
	"github.com/CarlFlo/mediaManager/internal/database"
	"github.com/CarlFlo/mediaManager/internal/notifications"
)

func (s *Service) Alert(key, level, message string) {
	if len(message) > 500 {
		message = message[:500]
	}
	now := time.Now().Unix()
	var old string
	var active int
	_ = s.DB.QueryRow("SELECT message,active FROM alerts WHERE key=?", key).Scan(&old, &active)
	if old == message && active == 1 {
		return
	}
	_, e := s.DB.Exec("INSERT INTO alerts VALUES(?,?,?,?,1,?,?) ON CONFLICT(key) DO UPDATE SET level=excluded.level,message=excluded.message,active=1,updated_at=excluded.updated_at", database.ID(), key, level, message, now, now)
	if e != nil {
		slog.Error("persist alert", "error", e)
		return
	}
	slog.Warn("application alert", "key", key, "message", message)
	if key != "provider:webhook" {
		action := "system_alert"
		if level == "error" {
			action = "system_error"
		}
		if strings.HasPrefix(key, "job:") {
			action = "job_failed"
		}
		if err := activity.Record(s.ctx, s.DB, activity.Event{Action: action, Message: message}); err != nil {
			slog.Error("record alert activity", "error", err)
		}
		if level == "error" {
			if err := notifications.QueueAlert(s.ctx, s.DB, key, level, message); err != nil {
				slog.Error("queue notification", "error", err)
			}
		}
		select {
		case s.notifications <- struct{}{}:
		default:
		}
	}
	s.changed("jobs")
	s.changedProfile("", "logs", "inbox")
}
