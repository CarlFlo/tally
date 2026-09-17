package api

import (
	"net/http"

	"github.com/CarlFlo/tally/internal/appversion"
	"github.com/CarlFlo/tally/internal/auth"
)

func (s *Server) bootstrap(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	profiles, e := s.DB.Rows(r.Context(), "SELECT p.id,p.display_name,p.avatar,p.locale,r.is_admin,p.auth_method,(p.auth_method='password') AS has_password FROM profiles p JOIN profile_roles r ON r.profile_id=p.id ORDER BY p.created_at,p.id")
	if e != nil {
		return e
	}
	session, _ := s.Auth.Resolve(r)
	var profile any
	var prefs any = map[string]any{}
	for _, p := range profiles {
		if p["id"] == session.Profile {
			profile = p
		}
	}
	if session.Profile != "" {
		prefs = s.readPreferences(r, session.Profile)
	}
	var preferenceCount int
	_ = s.DB.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM profile_preferences WHERE profile_id=?", session.Profile).Scan(&preferenceCount)
	jsonResponse(w, 200, map[string]any{"version": appversion.Version, "browser_theme": s.browserTheme(r), "profiles": profiles, "profile": profile, "preferences": prefs, "preferences_initialized": preferenceCount > 0, "auth_mode": "local", "restricted": session.Restricted, "warning": "", "max_profiles": s.Config.MaxProfiles, "password_min": s.Config.PasswordMin, "password_max": s.Config.PasswordMax, "jackett_enabled": s.torrentSearchEnabled(r.Context()), "torrent_search_enabled": s.torrentSearchEnabled(r.Context()), "torrent_downloads_enabled": s.torrentDownloadsEnabled(r.Context())})
	return nil
}
