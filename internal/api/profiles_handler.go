package api

import (
	"net/http"

	"github.com/CarlFlo/mediaManager/internal/auth"
)

func (s *Server) profiles(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	rows, e := s.DB.Rows(r.Context(), "SELECT p.id,p.display_name,p.avatar,r.is_admin FROM profiles p JOIN profile_roles r ON r.profile_id=p.id ORDER BY p.created_at,p.id")
	if e != nil {
		return e
	}
	jsonResponse(w, 200, rows)
	return nil
}
