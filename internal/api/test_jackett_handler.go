package api

import (
	"context"
	"net/http"
	"time"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/settings"
	"github.com/CarlFlo/tally/internal/torrent"
)

func (s *Server) testJackett(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	if err := s.operator(session); err != nil {
		return err
	}
	var input struct {
		Data settings.Search `json:"data"`
	}
	if err := decode(r, &input); err != nil {
		return err
	}
	input.Data.Enabled = true
	if err := settings.ValidateSearch(input.Data); err != nil {
		return bad(err.Error())
	}
	client := &torrent.Jackett{Control: s.Control, BaseURL: input.Data.BaseURL, APIKey: input.Data.APIKey}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	if err := client.TestConnection(ctx); err != nil {
		return remote(err)
	}
	jsonResponse(w, 200, map[string]string{"message": "Connected to Jackett. Authentication and Torznab search access verified."})
	return nil
}
