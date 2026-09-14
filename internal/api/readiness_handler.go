package api

import (
	"context"
	"net/http"
	"time"
)

func (s *Server) readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if e := s.DB.PingContext(ctx); e != nil {
		jsonResponse(w, 503, map[string]string{"status": "unavailable"})
		return
	}
	jsonResponse(w, 200, map[string]string{"status": "ready"})
}
