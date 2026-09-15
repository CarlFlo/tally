package api

import (
	"net/http"

	"github.com/CarlFlo/mediaManager/internal/auth"
	"github.com/CarlFlo/mediaManager/internal/database"
	"github.com/CarlFlo/mediaManager/internal/settings"
)

func (s *Server) settings(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	backupKeep := 10
	var retention settings.Backups
	if _, err := s.settingsStore().Load(r.Context(), "backups", &retention); err == nil {
		backupKeep = retention.Keep
	}
	webhookConfigured := false
	var webhook settings.Webhook
	if _, e := s.settingsStore().Load(r.Context(), "notifications", &webhook); e == nil {
		webhookConfigured = webhook.Enabled && webhook.URL != ""
	}
	metadataCron, maintenanceCron, backupCron := "0 * * * *", "30 3 * * *", "0 3 * * *"
	backupEnabled := true
	rows, _ := s.DB.Rows(r.Context(), "SELECT key,schedule,enabled FROM jobs")
	for _, row := range rows {
		switch row["key"] {
		case "metadata":
			metadataCron = row["schedule"].(string)
		case "maintenance":
			maintenanceCron = row["schedule"].(string)
		case "backup":
			backupCron = row["schedule"].(string)
			backupEnabled = row["enabled"].(int64) == 1
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
	c := s.Config
	var environment map[string]any
	if s.operator(session) == nil {
		environment = map[string]any{
			"TZ":                         c.Timezone,
			"APP_AUTH_MODE":              c.AuthMode,
			"APP_LANGUAGE":               c.Language,
			"APP_THEME_DEFAULT":          c.Theme,
			"APP_MAX_PROFILES":           c.MaxProfiles,
			"JOB_MAX_CONCURRENCY":        c.JobConcurrency,
			"JOB_MAX_RETRIES":            c.JobRetries,
			"JOB_MAX_BATCH_SIZE":         c.BatchSize,
			"PROVIDER_MAX_CONCURRENCY":   c.ProviderConcurrency,
			"STATS_RAW_RETENTION_DAYS":   c.RawRetention,
			"STATS_AGGREGATE_RETENTION_DAYS": c.AggregateRetention,
		}
	}
	jsonResponse(w, 200, map[string]any{"auth_mode": c.AuthMode, "max_profiles": c.MaxProfiles, "timezone": c.Timezone, "language": c.Language, "downloader": client.Adapter, "downloader_configured": client.Configured(), "jackett_configured": s.jackettConfigured(), "oidc_secret_configured": c.OIDCSecret != "", "webhook_configured": webhookConfigured, "backup_enabled": backupEnabled, "backup_keep": backupKeep, "backup_cron": backupCron, "metadata_cron": metadataCron, "maintenance_cron": maintenanceCron, "job_concurrency": c.JobConcurrency, "provider_concurrency": c.ProviderConcurrency, "operator": s.operator(session) == nil, "backups": backups, "schema_version": database.Version, "environment": environment})
	return nil
}
