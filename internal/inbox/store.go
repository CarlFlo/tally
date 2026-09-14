package inbox

import (
	"context"

	"github.com/CarlFlo/mediaManager/internal/database"
)

type Store struct{ DB *database.Store }
type Page struct {
	Entries []map[string]any `json:"entries"`
	Unread  int              `json:"unread"`
	Latest  int64            `json:"latest_id"`
}

func visibility(profile string) (string, []any) {
	if profile == "user0" {
		return "1=1", nil
	}
	return "l.profile_id=?", []any{profile}
}

func (s Store) List(ctx context.Context, profile string) (Page, error) {
	scope, args := visibility(profile)
	base := ` FROM activity_log l WHERE ` + scope + ` AND ` + categoryFilter(s.categories(ctx, profile))
	unread := base + ` AND l.id>COALESCE((SELECT MAX(seen_id,cleared_id) FROM inbox_state WHERE profile_id=?),0) AND NOT EXISTS(SELECT 1 FROM inbox_dismissals d WHERE d.profile_id=? AND d.activity_id=l.id)`
	var page Page
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*)"+unread, append(append([]any{}, args...), profile, profile)...).Scan(&page.Unread); err != nil {
		return page, err
	}
	if err := s.DB.QueryRowContext(ctx, "SELECT COALESCE(MAX(l.id),0)"+base, args...).Scan(&page.Latest); err != nil {
		return page, err
	}
	query := `SELECT l.*,CASE WHEN l.action LIKE '%failed%' OR l.action='system_error' THEN 'failed' WHEN l.action LIKE '%requested' THEN 'started' ELSE 'success' END AS status` + base + ` AND l.id>COALESCE((SELECT cleared_id FROM inbox_state WHERE profile_id=?),0) AND NOT EXISTS(SELECT 1 FROM inbox_dismissals d WHERE d.profile_id=? AND d.activity_id=l.id) ORDER BY l.id DESC LIMIT 25`
	var err error
	page.Entries, err = s.DB.Rows(ctx, query, append(append([]any{}, args...), profile, profile)...)
	return page, err
}
