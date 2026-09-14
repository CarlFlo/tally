package inbox

import (
	"context"
	"encoding/json"
	"strings"
)

var defaultCategories = []string{"scheduled_job_failures", "backup_failures", "episode_releases", "provider_api_failures", "torrent_client_failures"}

var categoryClauses = map[string]string{
	"scheduled_job_failures":  "(l.action='job_failed' AND l.message NOT LIKE 'backup:%')",
	"backup_failures":         "(l.action='job_failed' AND l.message LIKE 'backup:%')",
	"backup_successes":        "(l.action='job_succeeded' AND l.message='Completed backup job')",
	"episode_releases":        "l.action='episode_released'",
	"provider_api_failures":   "(l.action='system_error' AND l.message NOT LIKE 'Torrent submission failed.%')",
	"torrent_client_failures": "(l.action='system_error' AND l.message LIKE 'Torrent submission failed.%')",
	"routine_background":      "(l.action='job_succeeded' AND l.message<>'Completed backup job')",
}

func (s Store) categories(ctx context.Context, profile string) []string {
	categories := defaultCategories
	var raw string
	if s.DB.QueryRowContext(ctx, "SELECT data FROM profile_preferences WHERE profile_id=?", profile).Scan(&raw) != nil {
		return categories
	}
	var values struct {
		BellCategories []string `json:"bell_categories"`
	}
	if json.Unmarshal([]byte(raw), &values) != nil || values.BellCategories == nil {
		return categories
	}
	return values.BellCategories
}

func categoryFilter(categories []string) string {
	clauses := make([]string, 0, len(categories))
	for _, category := range categories {
		if clause, ok := categoryClauses[category]; ok {
			clauses = append(clauses, clause)
		}
	}
	if len(clauses) == 0 {
		return "0=1"
	}
	return "(" + strings.Join(clauses, " OR ") + ")"
}
