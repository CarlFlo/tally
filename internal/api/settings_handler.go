package api

import (
	"net/http"

	"github.com/CarlFlo/mediaManager/internal/auth"
	"github.com/CarlFlo/mediaManager/internal/database"
	"github.com/CarlFlo/mediaManager/internal/settings"
)

func (s *Server) settings(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	c := s.Config
	var retention settings.Backups
	if _, err := s.settingsStore().Load(r.Context(), "backups", &retention); err == nil {
		c.BackupKeep = retention.Keep
	}
	var webhook settings.Webhook
	if _, e := s.settingsStore().Load(r.Context(), "notifications", &webhook); e == nil {
		c.WebhookURL = ""
		if webhook.Enabled {
			c.WebhookURL = webhook.URL
		}
	}
	rows, _ := s.DB.Rows(r.Context(), "SELECT key,schedule,enabled FROM jobs")
	for _, row := range rows {
		switch row["key"] {
		case "metadata":
			c.MetadataCron = row["schedule"].(string)
		case "maintenance":
			c.MaintenanceCron = row["schedule"].(string)
		case "backup":
			c.BackupCron = row["schedule"].(string)
			c.BackupEnabled = row["enabled"].(int64) == 1
		}
	}
	backups := []map[string]any{}
	if s.operator(session) == nil {
		var e error
		backups, e = s.DB.Rows(r.Context(), "SELECT * FROM backup_records ORDER BY created_at DESC LIMIT 100")
		if e != nil {
			return e
		}
	}
	client, e := s.Clients.Load(r.Context())
	if e != nil {
		return e
	}
	jsonResponse(w, 200, map[string]any{"auth_mode": c.AuthMode, "max_profiles": c.MaxProfiles, "timezone": c.Timezone, "language": c.Language, "downloader": client.Adapter, "downloader_configured": client.Configured(), "jackett_configured": s.jackettConfigured(), "oidc_secret_configured": c.OIDCSecret != "", "webhook_configured": c.WebhookURL != "", "backup_enabled": c.BackupEnabled, "backup_keep": c.BackupKeep, "backup_cron": c.BackupCron, "metadata_cron": c.MetadataCron, "maintenance_cron": c.MaintenanceCron, "job_concurrency": c.JobConcurrency, "provider_concurrency": c.ProviderConcurrency, "operator": s.operator(session) == nil, "backups": backups, "schema_version": database.Version})
	return nil
}
