package api

import (
	"net/http"
)

func (s *Server) oidcStart(w http.ResponseWriter, r *http.Request) {
	if session, _ := s.Auth.Resolve(r); session.Profile != "" {
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}
	if e := s.OIDC.Start(w, r); e != nil {
		http.Error(w, e.Error(), 502)
	}
}
