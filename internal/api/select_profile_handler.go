package api

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/CarlFlo/tally/internal/auth"
)

func (s *Server) selectProfile(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	var in struct{ Profile string }
	if e := decode(r, &in); e != nil {
		return e
	}
	session, err := s.optionalSession(r)
	if err != nil {
		return err
	}
	if session.Profile != "" && session.Profile != in.Profile {
		return apiError{409, "sign out before choosing another profile"}
	}
	var method string
	err = s.DB.QueryRowContext(r.Context(), "SELECT auth_method FROM profiles WHERE id=?", in.Profile).Scan(&method)
	if errors.Is(err, sql.ErrNoRows) {
		return apiError{404, "profile no longer exists"}
	}
	if err != nil {
		return err
	}
	switch method {
	case auth.ProfileAuthNone:
		if err := s.Auth.NewSession(r.Context(), w, r, in.Profile, false); err != nil {
			return err
		}
	case auth.ProfileAuthPassword:
		return apiError{403, "password required"}
	default:
		return apiError{403, "profile authentication is unavailable"}
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
	return nil
}
