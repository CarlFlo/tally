package metadata

import (
	"context"
		"time"
)

// QueueFollow records the latest intent. Reversing an in-flight add invalidates
// its revision, so finishing the older import cannot re-follow the show.
func (s *Service) QueueFollow(ctx context.Context, profile string, external int, name string, desired bool) error {
	if external <= 0 || len(name) > 200 {
		return queueError("choose a valid show")
	}
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var count int
	if e = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM show_actions WHERE status IN ('queued','running') AND profile_id=? AND external_id<>?", profile, external).Scan(&count); e != nil {
		return e
	}
	if count >= 100 {
		return queueError("your queue is full; wait for a few shows to finish")
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO show_actions(profile_id,external_id,name,desired,status,updated_at) VALUES(?,?,?,?,'queued',?) ON CONFLICT(profile_id,external_id) DO UPDATE SET name=CASE WHEN excluded.name='' THEN show_actions.name ELSE excluded.name END,desired=excluded.desired,status='queued',revision=show_actions.revision+1,error='',updated_at=excluded.updated_at`, profile, external, name, desired, time.Now().UnixMilli())
	if e != nil {
		return e
	}
	return tx.Commit()
}
