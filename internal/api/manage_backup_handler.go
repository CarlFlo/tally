package api

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/CarlFlo/tally/internal/activity"
	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/backup"
	"github.com/CarlFlo/tally/internal/settings"
)

func (s *Server) restoreBackup(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	if s.Backup == nil {
		return apiError{500, "backup service is unavailable"}
	}
	manifest, err := s.Backup.RestoreArchive(r.Context(), r.PathValue("id"))
	if errors.Is(err, backup.ErrNotFound) {
		return apiError{404, "backup not found"}
	}
	if err != nil {
		slog.Error("backup restore failed", "backup_id", r.PathValue("id"), "error", err)
		_ = activity.Record(r.Context(), s.DB, activity.Event{Action: "job_failed", Profile: session.Profile, Message: "Backup restore failed. Check server logs for details."})
		if s.Events != nil {
			s.Events.Publish("", "logs", "inbox")
		}
		return apiError{409, "backup restore failed; check server logs for details"}
	}
	// A backup created before a newly introduced application setting can be
	// schema-compatible but not contain that setting row. Seed any missing
	// defaults immediately so live restore does not require a restart.
	if err = (settings.Store{DB: s.DB}).Ensure(r.Context()); err != nil {
		return err
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
	err := s.Backup.DeleteArchive(r.Context(), r.PathValue("id"))
	if errors.Is(err, backup.ErrNotFound) {
		return apiError{404, "backup not found"}
	}
	if err != nil {
		slog.Error("backup deletion failed", "backup_id", r.PathValue("id"), "error", err)
		return apiError{409, "backup could not be deleted; check server logs for details"}
	}
	jsonResponse(w, 200, map[string]string{"status": "deleted"})
	return nil
}
