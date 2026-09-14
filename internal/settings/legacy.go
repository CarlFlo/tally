package settings

import (
	"context"
	"encoding/json"
	"github.com/CarlFlo/mediaManager/internal/config"
	"github.com/CarlFlo/mediaManager/internal/scheduling"
	"os"
	"strconv"
)

// LegacyDefaults is used only by startup, before the first settings seed.
// Removed environment variables can never invalidate or override saved settings.
func (s Store) LegacyDefaults(ctx context.Context, c config.Config) config.Config {
	var savedBackups int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM application_settings WHERE key='backups'").Scan(&savedBackups); err == nil && savedBackups == 0 {
		if keep, err := strconv.Atoi(os.Getenv("BACKUP_KEEP")); err == nil && keep >= 1 && keep <= 1000 {
			c.BackupKeep = keep
		}
	}
	var count int
	if e := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM application_settings").Scan(&count); e != nil || count != 0 {
		return c
	}
	search := Search{Providers: []Indexer{}}
	var old []config.TorznabConfig
	raw := os.Getenv("TORZNAB_PROVIDERS")
	if len(raw) <= 32768 {
		_ = json.Unmarshal([]byte(raw), &old)
	}
	if address := os.Getenv("TORZNAB_URL"); address != "" {
		name := os.Getenv("TORZNAB_NAME")
		if name == "" {
			name = "Torznab"
		}
		old = append([]config.TorznabConfig{{ID: "torznab", Name: name, URL: address, APIKey: os.Getenv("TORZNAB_API_KEY")}}, old...)
	}
	for _, p := range old {
		search.Providers = append(search.Providers, Indexer{p.ID, p.Name, p.URL, p.APIKey, true})
	}
	if search.Effective().Configured() {
		c.TorznabProviders = old
	}
	webhook := Webhook{Enabled: true, URL: os.Getenv("WEBHOOK_URL")}
	if webhook.URL != "" && ValidateWebhook(webhook) == nil {
		c.WebhookURL = webhook.URL
	}
	for key, target := range map[string]*string{"JOB_METADATA_CRON": &c.MetadataCron, "JOB_MAINTENANCE_CRON": &c.MaintenanceCron, "BACKUP_CRON": &c.BackupCron} {
		if raw := os.Getenv(key); raw != "" {
			if _, e := scheduling.Parse(raw); e == nil {
				*target = raw
			}
		}
	}
	if raw, ok := os.LookupEnv("BACKUP_ENABLED"); ok {
		if enabled, e := strconv.ParseBool(raw); e == nil {
			c.BackupEnabled = enabled
		}
	}
	return c
}
