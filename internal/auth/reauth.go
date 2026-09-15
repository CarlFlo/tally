package auth

import (
	"context"
	"fmt"
	"time"
)

func (s *Service) Reauthenticate(ctx context.Context, profile, password string) error {
	if len(password) > 4096 {
		return fmt.Errorf("incorrect password or PIN")
	}
	key := "reauth:" + profile
	s.mu.Lock()
	a := s.attempts[key]
	if time.Now().Before(a.Next) {
		s.mu.Unlock()
		return fmt.Errorf("please wait a moment before trying again")
	}
	a.Next = time.Now().Add(time.Second)
	a.Last = time.Now()
	s.attempts[key] = a
	s.mu.Unlock()

	select {
	case s.hashes <- struct{}{}:
		defer func() { <-s.hashes }()
	case <-ctx.Done():
		return ctx.Err()
	}

	var hash string
	if err := s.DB.QueryRowContext(ctx, "SELECT hash FROM local_credentials WHERE profile_id=?", profile).Scan(&hash); err != nil || !Verify(hash, password) {
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
		return fmt.Errorf("incorrect password or PIN")
	}
	s.mu.Lock()
	delete(s.attempts, key)
	s.mu.Unlock()
	return nil
}
