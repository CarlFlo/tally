package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const (
	ProfileAuthPassword = "password"
	ProfileAuthNone     = "none"
)

func (s *Service) HashPassword(ctx context.Context, password string) (string, error) {
	if err := s.Policy(password); err != nil {
		return "", err
	}
	release, err := s.acquireHashMemory(ctx)
	if err != nil {
		return "", err
	}
	defer release()
	return Hash(password)
}

func (s *Service) ProfileAuthMethod(ctx context.Context, profile string) (string, error) {
	var method string
	var hasPassword bool
	if err := s.DB.QueryRowContext(ctx, `SELECT p.auth_method,EXISTS(SELECT 1 FROM local_credentials c WHERE c.profile_id=p.id)
		FROM profiles p WHERE p.id=?`, profile).Scan(&method, &hasPassword); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrProfileNotFound
		}
		return "", err
	}
	if hasPassword {
		return ProfileAuthPassword, nil
	}
	return method, nil
}

func (s *Service) SetProfileAuthentication(ctx context.Context, profile, method, password string) error {
	if method != ProfileAuthPassword && method != ProfileAuthNone {
		return fmt.Errorf("unsupported authentication method")
	}
	var hash string
	if method == ProfileAuthPassword {
		var err error
		hash, err = s.HashPassword(ctx, password)
		if err != nil {
			return err
		}
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, "UPDATE profiles SET auth_method=? WHERE id=?", method, profile)
	if err != nil {
		return err
	}
	if n, err := result.RowsAffected(); err != nil || n != 1 {
		if err != nil {
			return err
		}
		return ErrProfileNotFound
	}
	if method == ProfileAuthPassword {
		if _, err = tx.ExecContext(ctx, `INSERT INTO local_credentials(profile_id,hash,must_change) VALUES(?,?,0)
			ON CONFLICT(profile_id) DO UPDATE SET hash=excluded.hash,must_change=0`, profile, hash); err != nil {
			return err
		}
	} else if _, err = tx.ExecContext(ctx, "DELETE FROM local_credentials WHERE profile_id=?", profile); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM sessions WHERE profile_id=?", profile); err != nil {
		return err
	}
	return tx.Commit()
}
