package api

import (
	"errors"
	"net/http"

	"github.com/CarlFlo/tally/internal/activity"
	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/backup"
)

func (s *Server) restoreBackup(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	if s.Backup == nil {
		return apiError{500, "backup service is unavailable"}
	}
	manifest, err := s.Backup.RestoreRecord(r.Context(), r.PathValue("id"))
	if errors.Is(err, backup.ErrNotFound) {
		return apiError{404, "backup not found"}
	}
	if err != nil {
		message := "backup: restore failed: " + err.Error()
		_ = activity.Record(r.Context(), s.DB, activity.Event{Action: "job_failed", Profile: session.Profile, Message: message})
		if s.Events != nil {
			s.Events.Publish("", "logs", "inbox")
		}
		return apiError{409, "backup restore failed: " + err.Error()}
	}
	if s.Events != nil {
		s.Events.Publish("", "bootstrap", "calendar", "shows", "show-actions", "jobs", "schedules", "statistics", "logs", "settings", "editable-settings", "downloader", "backups", "inbox", "torrent-history", "capabilities", "sessions")
	}
	jsonResponse(w, 200, map[string]any{"status": "restored", "schema": manifest.Schema, "app_version": manifest.AppVersion})
	return nil
}

func (s *Server) deleteBackup(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	if s.Backup == nil {
		return apiError{500, "backup service is unavailable"}
	}
	err := s.Backup.DeleteRecord(r.Context(), r.PathValue("id"))
	if errors.Is(err, backup.ErrNotFound) {
		return apiError{404, "backup not found"}
	}
	if err != nil {
		return apiError{409, "backup could not be deleted: " + err.Error()}
	}
	if s.Events != nil {
		s.Events.Publish("", "backups")
	}
	jsonResponse(w, 200, map[string]string{"status": "deleted"})
	return nil
}
