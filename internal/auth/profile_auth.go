package auth

import (
	"context"
	"fmt"
)

const (
	ProfileAuthPassword     = "password"
	ProfileAuthNone         = "none"
	ProfileAuthOIDCUnlinked = "oidc_unlinked"
)

func (s *Service) ProfileAuthMethod(ctx context.Context, profile string) (string, error) {
	var method string
	if err := s.DB.QueryRowContext(ctx, "SELECT auth_method FROM profiles WHERE id=?", profile).Scan(&method); err != nil {
		return "", err
	}
	return method, nil
}

func (s *Service) SetProfileAuthentication(ctx context.Context, profile, method, password string) error {
	if method != ProfileAuthPassword && method != ProfileAuthNone {
		return fmt.Errorf("unsupported authentication method")
	}
	var hash string
	if method == ProfileAuthPassword {
		if err := s.Policy(password); err != nil {
			return err
		}
		var err error
		hash, err = Hash(password)
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
		return fmt.Errorf("profile not found")
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
