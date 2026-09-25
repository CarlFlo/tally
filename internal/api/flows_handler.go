package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/database"
	"github.com/CarlFlo/tally/internal/flows"
	"github.com/CarlFlo/tally/internal/settings"
	"github.com/CarlFlo/tally/internal/torrent"
)

func (s *Server) flowStore() flows.Store { return flows.Store{DB: s.DB} }
func (s *Server) flowList(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	items, err := s.flowStore().List(r.Context())
	if err != nil {
		return err
	}
	jsonResponse(w, 200, map[string]any{"flows": items})
	return nil
}
func (s *Server) flowGet(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	flow, err := s.flowStore().Get(r.Context(), r.PathValue("id"))
	if errors.Is(err, sql.ErrNoRows) {
		return apiError{404, "flow not found"}
	}
	if err != nil {
		return err
	}
	jsonResponse(w, 200, flow)
	return nil
}
func (s *Server) flowCreate(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	var in flows.Flow
	if err := decode(r, &in); err != nil {
		return err
	}
	if in.ID != "" {
		return bad("new flow must not specify an ID")
	}
	if in.ShowID != "" {
		if err := s.requireTorrentShowAccess(r.Context(), session, in.ShowID); err != nil {
			return err
		}
	}
	flow, err := s.flowStore().Save(r.Context(), in, 0)
	if err != nil {
		return bad(err.Error())
	}
	jsonResponse(w, 201, flow)
	return nil
}
func (s *Server) flowUpdate(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	var in flows.Flow
	if err := decode(r, &in); err != nil {
		return err
	}
	in.ID = r.PathValue("id")
	flow, err := s.flowStore().Save(r.Context(), in, in.Revision)
	if errors.Is(err, sql.ErrNoRows) {
		return apiError{404, "flow not found"}
	}
	if err != nil {
		if strings.Contains(err.Error(), "changed since") {
			return apiError{409, err.Error()}
		}
		return bad(err.Error())
	}
	jsonResponse(w, 200, flow)
	return nil
}
func (s *Server) flowDelete(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	if err := s.flowStore().Delete(r.Context(), r.PathValue("id")); errors.Is(err, sql.ErrNoRows) {
		return apiError{404, "flow not found"}
	} else if err != nil {
		return bad(err.Error())
	}
	jsonResponse(w, 200, map[string]bool{"deleted": true})
	return nil
}
func (s *Server) flowReset(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	var in struct {
		Revision int `json:"revision"`
	}
	if err := decode(r, &in); err != nil {
		return err
	}
	flow, err := s.flowStore().ResetDefault(r.Context(), r.PathValue("id"), in.Revision)
	if err != nil {
		if strings.Contains(err.Error(), "changed since") {
			return apiError{409, err.Error()}
		}
		return bad(err.Error())
	}
	jsonResponse(w, 200, flow)
	return nil
}
func (s *Server) flowRuns(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	runs, err := s.flowStore().ListRuns(r.Context())
	if err != nil {
		return err
	}
	jsonResponse(w, 200, map[string]any{"runs": runs})
	return nil
}
func (s *Server) flowRun(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	run, err := s.flowStore().GetRun(r.Context(), r.PathValue("id"))
	if errors.Is(err, sql.ErrNoRows) {
		return apiError{404, "flow run not found"}
	}
	if err != nil {
		return err
	}
	jsonResponse(w, 200, run)
	return nil
}
func (s *Server) flowReplay(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	var in struct {
		SourceRunID string `json:"source_run_id"`
		Event       *struct {
			Kind           string `json:"kind"`
			ShowName       string `json:"show_name"`
			Season         int    `json:"season"`
			Episode        int    `json:"episode"`
			RuntimeMinutes int    `json:"runtime_minutes"`
			MediaProfile   string `json:"media_profile"`
		} `json:"event,omitempty"`
		Definition *flows.Definition `json:"definition,omitempty"`
	}
	if err := decode(r, &in); err != nil {
		return err
	}
	flow, err := s.flowStore().Get(r.Context(), r.PathValue("id"))
	if errors.Is(err, sql.ErrNoRows) {
		return apiError{404, "flow not found"}
	}
	if err != nil {
		return err
	}
	if in.Definition != nil {
		flow.Definition = *in.Definition
		flow.Revision = 0
	}
	if err = flows.Validate(flow.Name, flow.Definition); err != nil {
		return bad(err.Error())
	}
	if (in.SourceRunID == "") == (in.Event == nil) {
		return bad("select one historical or custom event")
	}
	var event flows.Event
	if in.Event != nil {
		name := strings.TrimSpace(in.Event.ShowName)
		if in.Event.Kind != "show_available" || name == "" || utf8.RuneCountInString(name) > 200 || in.Event.Season < 0 || in.Event.Season > 100 || in.Event.Episode < 1 || in.Event.Episode > 1000 || in.Event.RuntimeMinutes < 0 || in.Event.RuntimeMinutes > 600 || (in.Event.MediaProfile != "" && in.Event.MediaProfile != "live" && in.Event.MediaProfile != "animated") {
			return bad("invalid custom show event")
		}
		profile := in.Event.MediaProfile
		if profile == "" && flow.ID == "default-animated" {
			profile = "animated"
		}
		event = flows.Event{Kind: in.Event.Kind, ShowID: "custom-" + database.ID(), EpisodeID: "custom-" + database.ID(), ShowName: name, Season: in.Event.Season, Episode: in.Event.Episode, TriggeredAt: time.Now().Unix(), RuntimeMinutes: in.Event.RuntimeMinutes, MediaProfile: profile}
	} else {
		source, sourceErr := s.torrentAutomationStore().GetRun(r.Context(), in.SourceRunID)
		if errors.Is(sourceErr, sql.ErrNoRows) {
			return apiError{404, "historical trigger not found"}
		}
		if sourceErr != nil {
			return sourceErr
		}
		event = flows.Event{Kind: "episode_released", SourceRunID: source.ID, ShowID: source.ShowID, EpisodeID: source.EpisodeID, ShowName: source.ShowName, Season: source.Season, Episode: source.Episode, TriggeredAt: source.StartedAt}
		var snapshot struct {
			RuntimeMinutes int `json:"runtime_minutes"`
		}
		_ = json.Unmarshal(source.SettingsSnapshot, &snapshot)
		event.RuntimeMinutes = snapshot.RuntimeMinutes
		profile, profileErr := s.torrentAutomationStore().ShowMediaProfile(r.Context(), source.ShowID)
		if profileErr == nil {
			event.MediaProfile = profile.Effective
		}
	}
	searchEnabled, err := s.torrentSearchEnabled(r.Context())
	if err != nil {
		return err
	}
	provider, err := s.jackett(r.Context())
	if err != nil {
		return err
	}
	var search flows.SearchFunc
	if searchEnabled && provider != nil {
		search = func(ctx context.Context, query string) ([]torrent.SearchResult, error) {
			query, err := normalizedSearchQuery(query)
			if err != nil {
				return nil, err
			}
			return provider.Search(ctx, torrent.SearchQuery{Query: query, NoRetry: true})
		}
	}
	base := settings.DefaultTorrentAutomation()
	if _, err := (settings.Store{DB: s.DB}).Load(r.Context(), "torrent_automation", &base); err != nil {
		return err
	}
	run := flows.ExecuteWithSettings(r.Context(), flow, event, base, search)
	// A cancelled HTTP request must not create a misleading historical record.
	if err = r.Context().Err(); err != nil {
		return err
	}
	run, err = s.flowStore().Record(r.Context(), run)
	if err != nil {
		return err
	}
	jsonResponse(w, 201, run)
	return nil
}
