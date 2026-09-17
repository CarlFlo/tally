package auth

import (
	"context"
	"fmt"
	"net/http"
)

func (s *Service) Change(ctx context.Context, w http.ResponseWriter, r *http.Request, session Session, current, password string) error {
	if e := s.Policy(password); e != nil {
		return e
	}
	release, err := s.acquireHashMemory(ctx)
	if err != nil {
		return err
	}
	defer release()
	if !session.Restricted {
		var hash string
		if e := s.DB.QueryRowContext(ctx, "SELECT hash FROM local_credentials WHERE profile_id=?", session.Profile).Scan(&hash); e != nil || !Verify(hash, current) {
			return fmt.Errorf("current password is incorrect")
		}
	}
	hash, e := Hash(password)
	if e != nil {
		return e
	}
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, "UPDATE local_credentials SET hash=?,must_change=0 WHERE profile_id=?", hash, session.Profile); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "DELETE FROM sessions WHERE profile_id=?", session.Profile); e != nil {
		return e
	}
	if e = tx.Commit(); e != nil {
		return e
	}
	s.mu.Lock()
	delete(s.recovery, session.Profile)
	s.mu.Unlock()
	return s.NewSession(ctx, w, r, session.Profile, false)
}
