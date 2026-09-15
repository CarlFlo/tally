package api

import (
	"net/http"

	"github.com/CarlFlo/mediaManager/internal/appversion"
	"github.com/CarlFlo/mediaManager/internal/auth"
	"github.com/CarlFlo/mediaManager/internal/database"
)

func (s *Server) backups(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	records, err := s.DB.Rows(r.Context(), "SELECT * FROM backup_records ORDER BY created_at DESC,id DESC LIMIT 100")
	if err != nil {
		return err
	}
	for _, record := range records {
		record["compatible"] = false
		filename, ok := record["filename"].(string)
		if !ok || s.Backup == nil {
			continue
		}
		manifest, inspectErr := s.Backup.Inspect(r.Context(), filename)
		if inspectErr != nil {
			record["archive_error"] = "Archive metadata could not be read"
			continue
		}
		record["schema"] = manifest.Schema
		record["app_version"] = manifest.AppVersion
		record["legacy_version"] = manifest.AppVersion == ""
		record["different_version"] = manifest.AppVersion != "" && manifest.AppVersion != appversion.Version
		record["compatible"] = manifest.Format == 1 && manifest.Schema >= 1 && manifest.Schema <= database.Version
	}
	failures, err := s.DB.Rows(r.Context(), "SELECT id,started_at,status,trigger FROM job_runs WHERE job_key='backup' AND status IN ('failed','interrupted') ORDER BY started_at DESC LIMIT 25")
	if err != nil {
		return err
	}
	jsonResponse(w, 200, map[string]any{"records": records, "failures": failures})
	return nil
}
