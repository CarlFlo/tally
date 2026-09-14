package api

import (
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/CarlFlo/mediaManager/internal/activity"
	"github.com/CarlFlo/mediaManager/internal/auth"
)

func (s *Server) downloadBackup(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	var filename string
	var verified bool
	if err := s.DB.QueryRowContext(r.Context(), "SELECT filename,verified FROM backup_records WHERE id=?", r.PathValue("id")).Scan(&filename, &verified); err != nil {
		return apiError{404, "backup not found"}
	}
	if !verified {
		return apiError{409, "this backup failed verification"}
	}
	if filepath.Base(filename) != filename || strings.ContainsAny(filename, "/\\:") {
		return bad("invalid backup filename")
	}
	root, err := os.OpenRoot(filepath.Join(s.Config.DataDir, "backups"))
	if err != nil {
		return apiError{404, "backup storage is unavailable"}
	}
	defer root.Close()
	file, err := root.Open(filename)
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
	if err = activity.Record(r.Context(), s.DB, activity.Event{Action: "backup_download_requested", Profile: session.Profile, Message: "Requested download of " + filename}); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	w.Header().Set("Cache-Control", "no-store")
	http.ServeContent(w, r, filename, info.ModTime(), file)
	return nil
}
