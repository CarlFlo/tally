package auth

import (
	"context"
)

func (s *Service) ResetPassword(ctx context.Context, profile, password string) error {
	hash, err := s.HashPassword(ctx, password)
	if err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, "UPDATE profiles SET auth_method=? WHERE id=?", ProfileAuthPassword, profile)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated != 1 {
		return ErrProfileNotFound
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO local_credentials(profile_id,hash,must_change) VALUES(?,?,1)
		ON CONFLICT(profile_id) DO UPDATE SET hash=excluded.hash,must_change=1`, profile, hash); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM sessions WHERE profile_id=?", profile); err != nil {
		return err
	}
	return tx.Commit()
}
