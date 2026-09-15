package api

import (
	"errors"
	"mime"
	"net/http"
	"os"

	"github.com/CarlFlo/tally/internal/activity"
	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/backup"
)

func (s *Server) downloadBackup(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	if s.Backup == nil {
		return apiError{500, "backup service is unavailable"}
	}
	archive, err := s.Backup.FindArchive(r.Context(), r.PathValue("id"))
	if errors.Is(err, backup.ErrNotFound) {
		return apiError{404, "backup not found"}
	}
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(s.Backup.Path)
	if err != nil {
		return apiError{404, "backup storage is unavailable"}
	}
	defer root.Close()
	file, err := root.Open(archive.Filename)
	if err != nil {
		return apiError{404, "backup file is unavailable"}
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return bad("invalid backup file")
	}
	if err = activity.Record(r.Context(), s.DB, activity.Event{Action: "backup_download_requested", Profile: session.Profile, Message: "Requested download of " + archive.Filename}); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": archive.Filename}))
	w.Header().Set("Cache-Control", "no-store")
	http.ServeContent(w, r, archive.Filename, info.ModTime(), file)
	return nil
}
