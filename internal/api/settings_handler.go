package api

import (
	"net/http"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/database"
	"github.com/CarlFlo/tally/internal/settings"
)

func (s *Server) settings(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	var retention settings.Backups
	if _, err := s.settingsStore().Load(r.Context(), "backups", &retention); err != nil {
		return err
	}

	var webhook settings.Webhook
	if _, err := s.settingsStore().Load(r.Context(), "notifications", &webhook); err != nil {
		return err
	}
	webhook = webhook.Defaults(s.Config.Timezone)
	webhookConfigured := webhook.Enabled && webhook.Endpoint() != ""

	rows, err := s.DB.Rows(r.Context(), "SELECT key,schedule,enabled FROM jobs")
	if err != nil {
		return err
	}
	metadataCron, maintenanceCron, backupCron := "0 * * * *", "30 3 * * *", "0 3 * * *"
	backupEnabled := true
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

	backups, err := s.DB.Rows(r.Context(), "SELECT * FROM backup_records ORDER BY created_at DESC LIMIT 100")
	if err != nil {
		return err
	}
	client, err := s.Clients.Load(r.Context())
	if err != nil {
		return err
	}
	jackettConfigured, err := s.jackettConfigured(r.Context())
	if err != nil {
		return err
	}
	searchEnabled, err := s.torrentSearchEnabled(r.Context())
	if err != nil {
		return err
	}
	downloadsEnabled, err := s.torrentDownloadsEnabled(r.Context())
	if err != nil {
		return err
	}

	c := s.Config
	environment := map[string]any{
		"TZ":                             c.Timezone,
		"APP_THEME_DEFAULT":              c.Theme,
		"APP_MAX_PROFILES":               c.MaxProfiles,
		"JOB_MAX_CONCURRENCY":            c.JobConcurrency,
		"JOB_MAX_RETRIES":                c.JobRetries,
		"JOB_MAX_BATCH_SIZE":             c.BatchSize,
		"PROVIDER_MAX_CONCURRENCY":       c.ProviderConcurrency,
		"STATS_RAW_RETENTION_DAYS":       c.RawRetention,
		"STATS_AGGREGATE_RETENTION_DAYS": c.AggregateRetention,
	}
	jsonResponse(w, 200, map[string]any{
		"max_profiles": c.MaxProfiles, "timezone": c.Timezone,
		"downloader": client.Adapter, "downloader_configured": client.Configured(),
		"jackett_configured": jackettConfigured,
		"torrent_search_enabled": searchEnabled, "torrent_downloads_enabled": downloadsEnabled,
		"webhook_configured": webhookConfigured,
		"backup_enabled": backupEnabled, "backup_keep": retention.Keep,
		"backup_cron": backupCron, "metadata_cron": metadataCron, "maintenance_cron": maintenanceCron,
		"job_concurrency": c.JobConcurrency, "provider_concurrency": c.ProviderConcurrency,
		"operator": session.Admin, "backups": backups, "schema_version": database.Version,
		"environment": environment,
	})
	return nil
}
