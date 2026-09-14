package api

import (
	"net/http"
)

func (s *Server) oidcCallback(w http.ResponseWriter, r *http.Request) {
	if session, _ := s.Auth.Resolve(r); session.Profile != "" {
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}
	if e := s.OIDC.Callback(w, r); e != nil {
		http.Error(w, e.Error(), 401)
	}
}
