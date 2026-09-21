package api

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/database"
)

func (s *Server) browserTheme(r *http.Request) string {
	theme := "system"
	if cookie, err := r.Cookie("tally_browser"); err == nil {
		if err = s.DB.QueryRowContext(r.Context(), "SELECT theme FROM browser_preferences WHERE id=?", cookie.Value).Scan(&theme); err != nil && !errors.Is(err, sql.ErrNoRows) {
			slog.Warn("read browser theme", "error", err)
		}
	}
	return theme
}

func (s *Server) browserPreferences(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	var in struct {
		Theme string `json:"theme"`
	}
	if err := decode(r, &in); err != nil {
		return err
	}
	if in.Theme != "system" && in.Theme != "light" && in.Theme != "dark" {
		return bad("choose light, dark, or system")
	}
	id := ""
	if cookie, err := r.Cookie("tally_browser"); err == nil {
		err = s.DB.QueryRowContext(r.Context(), "SELECT id FROM browser_preferences WHERE id=?", cookie.Value).Scan(&id)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}
	if id == "" {
		id = database.ID()
	}
	_, err := s.DB.ExecContext(r.Context(), `INSERT INTO browser_preferences(id,theme,updated_at) VALUES(?,?,unixepoch()) ON CONFLICT(id) DO UPDATE SET theme=excluded.theme,updated_at=excluded.updated_at`, id, in.Theme)
	if err != nil {
		return err
	}
	s.Auth.Cookie(w, r, "tally_browser", id, 365*24*3600)
	jsonResponse(w, 200, map[string]string{"theme": in.Theme})
	return nil
}
