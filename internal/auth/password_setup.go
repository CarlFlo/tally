package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/CarlFlo/mediaManager/internal/activity"
)

func (s *Service) HashPassword(ctx context.Context, password string) (string, error) {
	if err := s.Policy(password); err != nil {
		return "", err
	}
	select {
	case s.hashes <- struct{}{}:
		defer func() { <-s.hashes }()
	case <-ctx.Done():
		return "", ctx.Err()
	}
	return Hash(password)
}

// Setup claims a passwordless profile exactly once. Existing credentials can only
// be changed through authenticated password change or operator recovery.
func (s *Service) Setup(ctx context.Context, w http.ResponseWriter, r *http.Request, profile, password string) error {
	var exists, configured bool
	if err := s.DB.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM profiles WHERE id=?),EXISTS(SELECT 1 FROM local_credentials WHERE profile_id=?)", profile, profile).Scan(&exists, &configured); err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("profile no longer exists")
	}
	if configured {
		return fmt.Errorf("this profile already has a password; sign in instead")
	}
	hash, err := s.HashPassword(ctx, password)
	if err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var name string
	if err = tx.QueryRowContext(ctx, "SELECT display_name FROM profiles WHERE id=?", profile).Scan(&name); errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("profile no longer exists")
	} else if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, "INSERT INTO local_credentials(profile_id,hash,must_change) VALUES(?,?,0) ON CONFLICT DO NOTHING", profile, hash)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("this profile already has a password; sign in instead")
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM sessions WHERE profile_id=?", profile); err != nil {
		return err
	}
	if err = activity.Record(ctx, tx, activity.Event{Action: "password_created", Profile: profile, Message: "Set up a password for " + name}); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return s.NewSession(ctx, w, r, profile, false)
}
