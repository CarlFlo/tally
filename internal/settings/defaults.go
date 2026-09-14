package settings

import (
	"context"
	"encoding/json"

	"github.com/CarlFlo/mediaManager/internal/config"
)

// Legacy values seed the database only once. All live settings come from SQLite.
func (s Store) Ensure(ctx context.Context, c config.Config) error {
	search := Search{Providers: []Indexer{}}
	for _, p := range c.TorznabProviders {
		search.Providers = append(search.Providers, Indexer{p.ID, p.Name, p.URL, p.APIKey, true})
	}
	if len(search.Providers) == 0 && c.TorznabURL != "" {
		search.Providers = append(search.Providers, Indexer{"torznab", c.TorznabName, c.TorznabURL, c.TorznabKey, true})
	}
	search = search.Effective()
	for key, value := range map[string]any{"notifications": Webhook{Enabled: c.WebhookURL != "", URL: c.WebhookURL}, "search": search, "backups": Backups{Keep: backupKeep(c.BackupKeep)}} {
		raw, e := json.Marshal(value)
		if e != nil {
			return e
		}
		if _, e = s.DB.ExecContext(ctx, "INSERT INTO application_settings(key,data) VALUES(?,?) ON CONFLICT DO NOTHING", key, string(raw)); e != nil {
			return e
		}
	}
	return nil
}

func backupKeep(value int) int {
	if value < 1 || value > 1000 {
		return 10
	}
	return value
}
