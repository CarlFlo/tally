package jobs

import (
	"context"
	"time"

	"github.com/CarlFlo/tally/internal/metadata"
)

func (s *Service) runMetadata(ctx context.Context, id, trigger, show string) (result runResult, err error) {
	result.Attempt = 1

	query := `SELECT s.id,x.external_id,s.provider_updated_at FROM shows s JOIN external_ids x ON x.internal_id=s.id AND x.provider='tvmaze' AND x.kind='show' WHERE EXISTS(SELECT 1 FROM profile_shows p WHERE p.show_id=s.id) AND s.next_check_at<=? ORDER BY s.next_check_at LIMIT ?`
	args := []any{time.Now().Unix(), s.Config.BatchSize}
	if show != "" {
		query = "SELECT s.id,x.external_id,s.provider_updated_at FROM shows s JOIN external_ids x ON x.internal_id=s.id AND x.provider='tvmaze' AND x.kind='show' WHERE s.id=?"
		args = []any{show}
	}
	rows, e := s.DB.Rows(ctx, query, args...)
	if e != nil {
		err = e
		return result, err
	}
	result.Candidates = len(rows)
	if result.Candidates == 0 {
		s.Control.Avoid("tvmaze", trigger, "shows", "not due")
	}
	for _, row := range rows {
		if e = ctx.Err(); e != nil {
			err = e
			return result, err
		}
		work := metadata.WithInfo(ctx, metadata.Info{Trigger: trigger, JobID: id, Force: trigger == "manual_refresh"})
		_, e = s.Metadata.Sync(work, row["external_id"].(string))
		if e != nil {
			result.Skipped++
			err = e
			_, _ = s.DB.ExecContext(ctx, "UPDATE shows SET next_check_at=? WHERE id=?", time.Now().Add(time.Hour).Unix(), row["id"])
			continue
		}
		result.Processed++
		s.changed("shows", "show", "calendar", "statistics")
		var updated int64
		_ = s.DB.QueryRowContext(ctx, "SELECT provider_updated_at FROM shows WHERE id=?", row["id"]).Scan(&updated)
		if updated != row["provider_updated_at"].(int64) {
			result.Changes++
		}
	}
	return result, err
}
