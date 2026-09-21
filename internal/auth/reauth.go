package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

func (s *Service) Reauthenticate(ctx context.Context, profile, password string) error {
	if len(password) > 4096 {
		return ErrIncorrectPassword
	}
	key := "reauth:" + profile
	s.mu.Lock()
	a := s.attempts[key]
	if time.Now().Before(a.Next) {
		s.mu.Unlock()
		return ErrAuthenticationThrottled
	}
	a.Next = time.Now().Add(time.Second)
	a.Last = time.Now()
	s.attempts[key] = a
	s.mu.Unlock()

	release, err := s.acquireHashMemory(ctx)
	if err != nil {
		return err
	}
	defer release()

	var hash string
	err = s.DB.QueryRowContext(ctx, "SELECT hash FROM local_credentials WHERE profile_id=?", profile).Scan(&hash)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("read reauthentication credential: %w", err)
	}
	if errors.Is(err, sql.ErrNoRows) || !Verify(hash, password) {
		s.mu.Lock()
		a = s.attempts[key]
		a.Failures++
		delay := a.Failures
		if delay > 10 {
			delay = 10
		}
		a.Next = time.Now().Add(time.Duration(delay) * time.Second)
		s.attempts[key] = a
		s.mu.Unlock()
		return ErrIncorrectPassword
	}
	s.mu.Lock()
	delete(s.attempts, key)
	s.mu.Unlock()
	return nil
}
