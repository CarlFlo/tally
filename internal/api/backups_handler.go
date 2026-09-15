package api

import (
	"net/http"

	"github.com/CarlFlo/mediaManager/internal/appversion"
	"github.com/CarlFlo/mediaManager/internal/auth"
	"github.com/CarlFlo/mediaManager/internal/database"
)

func (s *Server) backups(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	records, err := s.DB.Rows(r.Context(), "SELECT * FROM backup_records WHERE verified=1 ORDER BY created_at DESC,id DESC LIMIT 100")
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
	jsonResponse(w, 200, map[string]any{"records": records})
	return nil
}
