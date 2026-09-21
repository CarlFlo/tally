package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var (
	ErrSignInRequired = errors.New("sign in to continue")
	ErrSessionExpired = errors.New("session expired; sign in again")
)

type Session struct {
	ID, Profile string
	Restricted  bool
	Admin       bool
}

func (s *Service) Cookie(w http.ResponseWriter, r *http.Request, name, value string, age int) {
	secure := r.TLS != nil || strings.HasPrefix(s.Config.PublicURL, "https://")
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/", MaxAge: age, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode})
}

func (s *Service) Resolve(r *http.Request) (Session, error) {
	ctx := r.Context()
	c, e := r.Cookie("tally_session")
	if e != nil {
		return Session{}, ErrSignInRequired
	}
	var session Session
	var last, expires int64
	session.ID = Digest(c.Value)
	e = s.DB.QueryRowContext(ctx, `SELECT s.profile_id,s.last_seen,s.expires_at,s.restricted,r.is_admin
		FROM sessions s JOIN profile_roles r ON r.profile_id=s.profile_id WHERE s.id=?`, session.ID).
		Scan(&session.Profile, &last, &expires, &session.Restricted, &session.Admin)
	if errors.Is(e, sql.ErrNoRows) {
		return Session{}, ErrSessionExpired
	}
	if e != nil {
		return Session{}, fmt.Errorf("resolve session: %w", e)
	}
	now := time.Now().Unix()
	if now >= expires || now-last >= int64(s.Config.SessionIdle.Seconds()) {
		return Session{}, ErrSessionExpired
	}
	if now-last > 60 {
		if _, e = s.DB.ExecContext(ctx, "UPDATE sessions SET last_seen=? WHERE id=?", now, session.ID); e != nil {
			return Session{}, fmt.Errorf("refresh session activity: %w", e)
		}
	}
	return session, nil
}

func (s *Service) NewSession(ctx context.Context, w http.ResponseWriter, r *http.Request, profile string, restricted bool) error {
	token := Token()
	now := time.Now()
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if old, e := r.Cookie("tally_session"); e == nil {
		_, e = tx.ExecContext(ctx, "DELETE FROM sessions WHERE id=?", Digest(old.Value))
		if e != nil {
			return e
		}
	}
	_, e = tx.ExecContext(ctx, "INSERT INTO sessions VALUES(?,?,?,?,?,?,?)", Digest(token), profile, now.Unix(), now.Unix(), now.Add(s.Config.SessionAbsolute).Unix(), restricted, truncate(r.UserAgent(), 250))
	if e != nil {
		return e
	}
	if e = tx.Commit(); e != nil {
		return e
	}
	s.Cookie(w, r, "tally_session", token, int(s.Config.SessionAbsolute.Seconds()))
	return nil
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
