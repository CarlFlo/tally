package api

import (
	"context"
	"net/http"
	"time"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/torrent"
)

func (s *Server) testDownloader(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	if e := s.operator(session); e != nil {
		return e
	}
	var in torrent.ClientUpdate
	if e := decode(r, &in); e != nil {
		return e
	}
	c, e := s.Clients.Prepare(r.Context(), in)
	if e != nil {
		return clientInputError(e)
	}
	client, e := s.Clients.Build(c)
	if e != nil {
		return clientInputError(e)
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	if e = client.TestConnection(ctx); e != nil {
		return remote(e)
	}
	jsonResponse(w, 200, map[string]string{"message": "Connected to " + client.Name() + ". Authentication and API access verified."})
	return nil
}
