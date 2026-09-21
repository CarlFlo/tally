package api

import (
	"database/sql"
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/CarlFlo/tally/internal/auth"
)

func (s *Server) avatar(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	name := r.PathValue("name")
	if filepath.Base(name) != name || !strings.HasSuffix(name, ".png") {
		return apiError{404, "avatar not found"}
	}
	var exists int
	err := s.DB.QueryRowContext(r.Context(), "SELECT 1 FROM profiles WHERE avatar=?", name).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return apiError{404, "avatar not found"}
	}
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public,max-age=86400")
	http.ServeFile(w, r, filepath.Join(s.Config.DataDir, "avatars", name))
	return nil
}
