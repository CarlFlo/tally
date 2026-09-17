package auth

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"
	"time"
)

type throttle struct {
	Failures   int
	Next, Last time.Time
}

func (s *Service) Login(ctx context.Context, w http.ResponseWriter, r *http.Request, profile, password string) error {
	if len(profile) > 64 {
		return fmt.Errorf("incorrect password")
	}
	var exists int
	if s.DB.QueryRowContext(ctx, "SELECT 1 FROM profiles WHERE id=?", profile).Scan(&exists) != nil {
		return fmt.Errorf("incorrect password")
	}
	if len(password) > 4096 {
		return fmt.Errorf("password is too long")
	}
	s.mu.Lock()
	a := s.attempts[profile]
	if time.Now().Before(a.Next) {
		s.mu.Unlock()
		return fmt.Errorf("please wait a moment before trying again")
	}
	a.Next = time.Now().Add(time.Second)
	a.Last = time.Now()
	s.attempts[profile] = a
	s.mu.Unlock()
	release, err := s.acquireHashMemory(ctx)
	if err != nil {
		return err
	}
	defer release()
	var hash string
	var must bool
	e := s.DB.QueryRowContext(ctx, "SELECT hash,must_change FROM local_credentials WHERE profile_id=?", profile).Scan(&hash, &must)
	valid := e == nil && Verify(hash, password)
	s.mu.Lock()
	rec, ok := s.recovery[profile]
	temporary := ok && time.Now().Before(rec.Expires) && subtle.ConstantTimeCompare([]byte(rec.Hash), []byte(Digest(password))) == 1
	if !valid && !temporary {
		a = s.attempts[profile]
		a.Failures++
		delay := a.Failures
		if delay > 10 {
			delay = 10
		}
		a.Next = time.Now().Add(time.Duration(delay) * time.Second)
		s.attempts[profile] = a
		s.mu.Unlock()
		return fmt.Errorf("incorrect password")
	}
	delete(s.attempts, profile)
	if temporary {
		delete(s.recovery, profile)
	}
	s.mu.Unlock()
	return s.NewSession(ctx, w, r, profile, must || temporary)
}
