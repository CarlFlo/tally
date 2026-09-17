package api

import (
	"net/http"

	"github.com/CarlFlo/tally/internal/auth"
)

func (s *Server) password(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	method, err := s.Auth.ProfileAuthMethod(r.Context(), session.Profile)
	if err != nil {
		return err
	}
	if method != auth.ProfileAuthPassword {
		return bad("password changes require password authentication")
	}
	var in struct{ Current, Password string }
	if err = decode(r, &in); err != nil {
		return err
	}
	if err = s.Auth.Change(r.Context(), w, r, session, in.Current, in.Password); err != nil {
		return bad(err.Error())
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
	return nil
}
