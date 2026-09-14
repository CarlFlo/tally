package api

import (
	"net/http"

	"github.com/CarlFlo/mediaManager/internal/auth"
)

func (s *Server) bootstrap(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	profiles, e := s.DB.Rows(r.Context(), "SELECT p.id,p.display_name,p.avatar,EXISTS(SELECT 1 FROM local_credentials c WHERE c.profile_id=p.id) AS has_password FROM profiles p ORDER BY p.created_at,p.id")
	if e != nil {
		return e
	}
	session, _ := s.Auth.Resolve(r)
	if s.Config.AuthMode == "disabled" && session.Profile != "" {
		s.Auth.Cookie(w, r, "tally_profile", session.Profile, 365*24*3600)
	}
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
	warning := ""
	if s.Config.AuthMode == "disabled" && (publicHost(s.Config.PublicURL) || publicHost("http://"+r.Host)) {
		warning = "Authentication is disabled and this address appears public. Anyone with access can use every profile."
		if s.Jobs != nil {
			s.Jobs.Alert("auth:public", "warning", warning)
		}
	}
	var preferenceCount int
	_ = s.DB.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM profile_preferences WHERE profile_id=?", session.Profile).Scan(&preferenceCount)
	jsonResponse(w, 200, map[string]any{"browser_theme": s.browserTheme(r), "profiles": profiles, "profile": profile, "preferences": prefs, "preferences_initialized": preferenceCount > 0, "auth_mode": s.Config.AuthMode, "restricted": session.Restricted, "warning": warning, "max_profiles": s.Config.MaxProfiles, "password_min": s.Config.PasswordMin, "password_max": s.Config.PasswordMax})
	return nil
}
