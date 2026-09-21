package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type throttle struct {
	Failures   int
	Next, Last time.Time
}

func (s *Service) Login(ctx context.Context, w http.ResponseWriter, r *http.Request, profile, password string) error {
	if len(profile) > 64 {
		return ErrIncorrectPassword
	}
	var exists int
	err := s.DB.QueryRowContext(ctx, "SELECT 1 FROM profiles WHERE id=?", profile).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrIncorrectPassword
	}
	if err != nil {
		return fmt.Errorf("read login profile: %w", err)
	}
	if len(password) > 4096 {
		return ErrPasswordTooLong
	}
	s.mu.Lock()
	a := s.attempts[profile]
	if time.Now().Before(a.Next) {
		s.mu.Unlock()
		return ErrAuthenticationThrottled
	}
	a.Next = time.Now().Add(time.Second)
	a.Last = time.Now()
	s.attempts[profile] = a
	s.mu.Unlock()

	release, err := s.acquireHashMemory(ctx)
	if err != nil {
		return err
	}
	var hash string
	var must bool
	err = s.DB.QueryRowContext(ctx, "SELECT hash,must_change FROM local_credentials WHERE profile_id=?", profile).Scan(&hash, &must)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		release()
		return fmt.Errorf("read login credentials: %w", err)
	}
	valid := err == nil && Verify(hash, password)
	release()

	if valid && NeedsRehash(hash) {
		release, hashErr := s.acquireHashMemory(ctx)
		if hashErr == nil {
			var upgraded string
			upgraded, hashErr = Hash(password)
			release()
			if hashErr == nil {
				_, hashErr = s.DB.ExecContext(ctx, "UPDATE local_credentials SET hash=? WHERE profile_id=? AND hash=?", upgraded, profile, hash)
			}
		}
		if hashErr != nil {
			slog.Warn("password hash migration failed", "profile_id", profile, "error", hashErr)
		}
	}

	if !valid {
		s.mu.Lock()
		a = s.attempts[profile]
		a.Failures++
		delay := a.Failures
		if delay > 10 {
			delay = 10
		}
		a.Next = time.Now().Add(time.Duration(delay) * time.Second)
		s.attempts[profile] = a
		s.mu.Unlock()
		return ErrIncorrectPassword
	}
	s.mu.Lock()
	delete(s.attempts, profile)
	s.mu.Unlock()
	return s.NewSession(ctx, w, r, profile, must)
}
