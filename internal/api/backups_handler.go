package api

import (
	"net/http"

	"github.com/CarlFlo/tally/internal/appversion"
	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/database"
)

func (s *Server) backups(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	if s.Backup == nil {
		return apiError{500, "backup service is unavailable"}
	}
	archives, err := s.Backup.Archives(r.Context())
	if err != nil {
		return err
	}
	records := make([]map[string]any, 0, len(archives))
	for _, archive := range archives {
		record := map[string]any{
			"id":         archive.ID,
			"filename":   archive.Filename,
			"kind":       archive.Kind,
			"size":       archive.Size,
			"created_at": archive.CreatedAt,
			"compatible": false,
		}
		manifest, inspectErr := s.Backup.Inspect(r.Context(), archive.Filename)
		if inspectErr != nil {
			record["archive_error"] = "Archive metadata could not be read"
			records = append(records, record)
			continue
		}
		record["schema"] = manifest.Schema
		record["app_version"] = manifest.AppVersion
		record["legacy_version"] = manifest.AppVersion == ""
		record["different_version"] = manifest.AppVersion != "" && manifest.AppVersion != appversion.Version
		record["compatible"] = manifest.Format == 1 && manifest.Schema >= 1 && manifest.Schema <= database.Version
		records = append(records, record)
	}
	jsonResponse(w, 200, map[string]any{"records": records})
	return nil
}
