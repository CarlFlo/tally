package auth

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type recovery struct {
	Hash          string
	Expires, Last time.Time
}

func (s *Service) Recover(ctx context.Context, profile string) error {
	var exists int
	if e := s.DB.QueryRowContext(ctx, "SELECT 1 FROM local_credentials WHERE profile_id=?", profile).Scan(&exists); e != nil {
		return fmt.Errorf("this profile has no local account")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	old := s.recovery[profile]
	if time.Since(old.Last) < s.Config.ResetCooldown {
		return fmt.Errorf("recovery was requested recently; wait before trying again")
	}
	password := Token()
	now := time.Now()
	s.recovery[profile] = recovery{Digest(password), now.Add(30 * time.Minute), now}
	slog.Warn("LOCAL PASSWORD RECOVERY — read from operator logs only", "profile_id", profile, "temporary_password", password, "expires_at", now.Add(30*time.Minute))
	return nil
}
