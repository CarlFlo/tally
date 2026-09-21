package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
)

func (s *Service) Change(ctx context.Context, w http.ResponseWriter, r *http.Request, session Session, current, password string) error {
	if e := s.Policy(password); e != nil {
		return e
	}
	if !session.Restricted {
		release, err := s.acquireHashMemory(ctx)
		if err != nil {
			return err
		}
		var currentHash string
		queryErr := s.DB.QueryRowContext(ctx, "SELECT hash FROM local_credentials WHERE profile_id=?", session.Profile).Scan(&currentHash)
		if queryErr != nil && !errors.Is(queryErr, sql.ErrNoRows) {
			release()
			return fmt.Errorf("read current password: %w", queryErr)
		}
		valid := queryErr == nil && Verify(currentHash, current)
		release()
		if !valid {
			return ErrCurrentPasswordIncorrect
		}
	}

	release, err := s.acquireHashMemory(ctx)
	if err != nil {
		return err
	}
	hash, e := Hash(password)
	release()
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
	return s.NewSession(ctx, w, r, session.Profile, false)
}
