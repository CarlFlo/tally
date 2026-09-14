package api

import (
	"net/http"

	"github.com/CarlFlo/mediaManager/internal/auth"
)

func (s *Server) password(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	if s.Config.AuthMode != "local" {
		return bad("password changes require local authentication")
	}
	var in struct{ Current, Password string }
	if e := decode(r, &in); e != nil {
		return e
	}
	if e := s.Auth.Change(r.Context(), w, r, session, in.Current, in.Password); e != nil {
		return bad(e.Error())
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
	return nil
}
