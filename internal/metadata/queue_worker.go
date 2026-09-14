package metadata

import (
	"context"
	"database/sql"
	"errors"
	"github.com/CarlFlo/mediaManager/internal/library"
	"strconv"
	"time"
)

// ProcessNext performs one bounded item. It is called by the server's single
// queue worker, never by the browser request that enqueues the action.
func (s *Service) ProcessNext(ctx context.Context) (processed bool, resultErr error) {
	var profile, name string
	var external int
	var revision int64
	var desired bool
	e := s.DB.QueryRowContext(ctx, "SELECT profile_id,external_id,name,desired,revision FROM show_actions WHERE status='queued' ORDER BY updated_at,profile_id,external_id LIMIT 1").Scan(&profile, &external, &name, &desired, &revision)
	if errors.Is(e, sql.ErrNoRows) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	res, e := s.DB.ExecContext(ctx, "UPDATE show_actions SET status='running' WHERE profile_id=? AND external_id=? AND revision=? AND status='queued'", profile, external, revision)
	if e != nil {
		return false, e
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return true, nil
	}
	defer func() {
		s.changedProfile(profile, "show-actions", "shows", "calendar", "logs")
		if profile != "user0" {
			s.changedProfile("user0", "logs")
		}
	}()
	// A failed commit must not strand the action in 'running' until restart.
	// Later transaction rollbacks run before this cleanup.
	defer func() {
		if resultErr == nil {
			return
		}
		status, message := "failed", "Could not update "+name+". Please retry."
		if errors.Is(ctx.Err(), context.Canceled) {
			status, message = "queued", ""
		}
		_, _ = s.DB.Exec("UPDATE show_actions SET status=?,error=?,updated_at=? WHERE profile_id=? AND external_id=? AND revision=? AND status='running'", status, message, time.Now().UnixMilli(), profile, external, revision)
	}()
	var id string
	e = s.DB.QueryRowContext(ctx, "SELECT internal_id FROM external_ids WHERE provider='tvmaze' AND kind='show' AND external_id=?", strconv.Itoa(external)).Scan(&id)
	if errors.Is(e, sql.ErrNoRows) {
		e = nil
		if desired {
			id, e = s.Sync(WithInfo(ctx, Info{Trigger: "user_add"}), strconv.Itoa(external))
		}
	}
	if e != nil {
		status := "failed"
		message := "Could not add " + name + ". " + e.Error()
		if errors.Is(ctx.Err(), context.Canceled) {
			status = "queued"
			message = ""
		}
		_, writeErr := s.DB.Exec("UPDATE show_actions SET status=?,error=?,updated_at=? WHERE profile_id=? AND external_id=? AND revision=?", status, message, time.Now().UnixMilli(), profile, external, revision)
		return true, writeErr
	}
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return true, e
	}
	defer tx.Rollback()
	var current int64
	if e = tx.QueryRowContext(ctx, "SELECT revision FROM show_actions WHERE profile_id=? AND external_id=?", profile, external).Scan(&current); errors.Is(e, sql.ErrNoRows) {
		return true, nil
	} else if e != nil {
		return true, e
	}
	if current != revision {
		return true, nil
	}
	if id != "" {
		e = library.SetFollow(ctx, tx, profile, id, desired)
		if e != nil {
			return true, e
		}
	}
	_, e = tx.ExecContext(ctx, "UPDATE show_actions SET status='done',show_id=?,error='',updated_at=? WHERE profile_id=? AND external_id=? AND revision=?", id, time.Now().UnixMilli(), profile, external, revision)
	if e != nil {
		return true, e
	}
	return true, tx.Commit()
}
