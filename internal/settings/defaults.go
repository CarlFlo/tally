package settings

import (
	"context"
	"encoding/json"
)

// Defaults seed the database once. All live settings come from SQLite.
func (s Store) Ensure(ctx context.Context) error {
	for key, value := range map[string]any{"notifications": Webhook{}, "search": Search{}, "torrent": Torrent{Enabled: true}, "backups": Backups{Keep: 10}} {
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
