package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/torrent"
)

func (s *Server) torrentAutomationStore() torrent.AutomationStore {
	return torrent.AutomationStore{DB: s.DB}
}

func (s *Server) torrentAutomationRuns(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 200 {
			return bad("run history limit must be between 1 and 200")
		}
		limit = value
	}
	runs, err := s.torrentAutomationStore().ListRuns(r.Context(), limit)
	if err != nil {
		return err
	}
	jsonResponse(w, 200, map[string]any{"runs": runs})
	return nil
}

func (s *Server) torrentAutomationRun(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	run, err := s.torrentAutomationStore().GetRun(r.Context(), r.PathValue("id"))
	if errors.Is(err, sql.ErrNoRows) {
		return apiError{404, "automation run not found"}
	}
	if err != nil {
		return err
	}
	jsonResponse(w, 200, run)
	return nil
}

func (s *Server) markTorrentAutomationRunBad(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	var in struct {
		Reason string `json:"reason"`
		Note   string `json:"note"`
	}
	if err := decode(r, &in); err != nil {
		return err
	}
	store := s.torrentAutomationStore()
	run, err := store.GetRun(r.Context(), r.PathValue("id"))
	if errors.Is(err, sql.ErrNoRows) {
		return apiError{404, "automation run not found"}
	}
	if err != nil {
		return err
	}
	if run.Feedback != nil {
		return apiError{409, "this run has already been marked bad"}
	}
	if run.Status == torrent.RunRunning {
		return bad("a running automation attempt cannot be marked bad")
	}
	if err = torrent.ValidateAutomationFeedback(in.Reason, in.Note); err != nil {
		return bad(err.Error())
	}
	if err = store.MarkBad(r.Context(), run.ID, session.Profile, in.Reason, in.Note); err != nil {
		return err
	}
	updated, err := store.GetRun(r.Context(), run.ID)
	if err != nil {
		return err
	}
	jsonResponse(w, 200, updated)
	return nil
}


func (s *Server) torrentAutomationShows(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	rows, err := s.DB.Rows(r.Context(), `SELECT
		s.id,s.name,s.image,s.status,s.premiered,s.network,
		COALESCE((SELECT MIN(e.airdate) FROM episodes e WHERE e.show_id=s.id AND e.airdate>=date('now')),'') AS next_episode,
		CASE WHEN EXISTS(SELECT 1 FROM episodes e WHERE e.show_id=s.id AND e.airdate>=date('now')) THEN 1 ELSE 0 END AS active,
		CASE WHEN COALESCE(p.policy,'default')='auto' THEN 1 ELSE 0 END AS automation_enabled
		FROM shows s
		JOIN profile_shows f ON f.show_id=s.id
		LEFT JOIN torrent_show_policy p ON p.show_id=s.id
		WHERE f.profile_id=?
		ORDER BY s.name COLLATE NOCASE`, session.Profile)
	if err != nil {
		return err
	}
	jsonResponse(w, 200, map[string]any{"shows": rows})
	return nil
}

func (s *Server) torrentShowPolicy(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	if err := torrent.ValidateShowMediaProfile(in.Mode); err != nil {
		return bad(err.Error())
	}
	showID := r.PathValue("id")
	var exists int
	if err := s.DB.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM shows WHERE id=?)", showID).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return apiError{404, "show not found"}
	}
	policy, err := s.torrentAutomationStore().ShowPolicy(r.Context(), showID)
	if err != nil {
		return err
	}
	jsonResponse(w, 200, map[string]any{"policy": policy, "enabled": policy == "auto"})
	return nil
}

func (s *Server) updateTorrentShowPolicy(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	if err := s.operator(session); err != nil {
		return err
	}
	var in struct {
		Policy  string `json:"policy"`
		Enabled *bool  `json:"enabled"`
	}
	if err := decode(r, &in); err != nil {
		return err
	}
	if in.Enabled != nil {
		if *in.Enabled {
			in.Policy = "auto"
		} else {
			in.Policy = "default"
		}
	}
	if in.Policy == "" {
		return bad("automation enrollment is required")
	}
	if err := torrent.ValidateShowPolicy(in.Policy); err != nil {
		return bad(err.Error())
	}
	showID := r.PathValue("id")
	var exists int
	if err := s.DB.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM shows WHERE id=?)", showID).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return apiError{404, "show not found"}
	}
	if err := s.torrentAutomationStore().SetShowPolicy(r.Context(), showID, in.Policy); err != nil {
		return err
	}
	policy, err := s.torrentAutomationStore().ShowPolicy(r.Context(), showID)
	if err != nil {
		return err
	}
	if s.Events != nil {
		s.Events.Publish("", "torrent-automation-shows")
	}
	jsonResponse(w, 200, map[string]any{"policy": policy, "enabled": policy == "auto"})
	return nil
}

func (s *Server) torrentShowMediaProfile(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	showID := r.PathValue("id")
	profile, err := s.torrentAutomationStore().ShowMediaProfile(r.Context(), showID)
	if errors.Is(err, sql.ErrNoRows) {
		return apiError{404, "show not found"}
	}
	if err != nil {
		return err
	}
	jsonResponse(w, 200, profile)
	return nil
}

func (s *Server) updateTorrentShowMediaProfile(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	if err := s.operator(session); err != nil {
		return err
	}
	var in struct {
		Mode string `json:"mode"`
	}
	if err := decode(r, &in); err != nil {
		return err
	}
	showID := r.PathValue("id")
	var exists int
	if err := s.DB.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM shows WHERE id=?)", showID).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return apiError{404, "show not found"}
	}
	if err := s.torrentAutomationStore().SetShowMediaProfile(r.Context(), showID, in.Mode); err != nil {
		return err
	}
	profile, err := s.torrentAutomationStore().ShowMediaProfile(r.Context(), showID)
	if err != nil {
		return err
	}
	jsonResponse(w, 200, profile)
	return nil
}
