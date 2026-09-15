package api

import (
	"net/http"
	"strconv"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/inbox"
)

func (s *Server) inbox(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	result, err := (inbox.Store{DB: s.DB}).List(r.Context(), session.Profile, session.Admin)
	if err != nil {
		return err
	}
	jsonResponse(w, 200, result)
	return nil
}

func (s *Server) markInbox(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	if r.PathValue("action") != "seen" && r.PathValue("action") != "clear" {
		return apiError{404, "unknown notification action"}
	}
	var in struct {
		Through int64 `json:"through"`
	}
	if err := decode(r, &in); err != nil {
		return err
	}
	if in.Through < 0 {
		return bad("invalid notification marker")
	}
	if err := (inbox.Store{DB: s.DB}).Mark(r.Context(), session.Profile, session.Admin, in.Through, r.PathValue("action") == "clear"); err != nil {
		return err
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
	return nil
}

func (s *Server) dismissInbox(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		return bad("invalid notification")
	}
	if err = (inbox.Store{DB: s.DB}).Dismiss(r.Context(), session.Profile, session.Admin, id); err != nil {
		return err
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
	return nil
}
