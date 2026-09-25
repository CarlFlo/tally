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

	"github.com/CarlFlo/tally/internal/activity"
	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/database"
	"github.com/CarlFlo/tally/internal/flows"
	"github.com/CarlFlo/tally/internal/torrent"
)

func (s *Server) flowStore() flows.Store { return flows.Store{DB: s.DB} }
func (s *Server) flowKinds(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	jsonResponse(w, 200, map[string]any{"nodes": flows.Kinds()})
	return nil
}
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
func (s *Server) flowCreate(w http.ResponseWriter, r *http.Request, _ auth.Session) error {
	var in flows.Flow
	if err := decode(r, &in); err != nil {
		return err
	}
	if in.ID != "" {
		return bad("new flow must not specify an ID")
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
func (s *Server) flowReplay(w http.ResponseWriter, r *http.Request, session auth.Session) error {
	var in struct {
		SourceRunID string `json:"source_run_id"`
		Event       *struct {
			Kind     string `json:"kind"`
			ShowName string `json:"show_name"`
			Season   int    `json:"season"`
			Episode  int    `json:"episode"`
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
		if in.Event.Kind != "show_available" || name == "" || utf8.RuneCountInString(name) > 200 || in.Event.Season < 0 || in.Event.Season > 100 || in.Event.Episode < 1 || in.Event.Episode > 1000 {
			return bad("invalid custom show event")
		}
		event = flows.Event{Kind: in.Event.Kind, ShowID: "custom-" + database.ID(), EpisodeID: "custom-" + database.ID(), ShowName: name, Season: in.Event.Season, Episode: in.Event.Episode, TriggeredAt: time.Now().Unix()}
	} else {
		source, sourceErr := s.torrentAutomationStore().GetRun(r.Context(), in.SourceRunID)
		if errors.Is(sourceErr, sql.ErrNoRows) {
			return apiError{404, "historical trigger not found"}
		}
		if sourceErr != nil {
			return sourceErr
		}
		event = flows.Event{Kind: "episode_released", SourceRunID: source.ID, ShowID: source.ShowID, EpisodeID: source.EpisodeID, ShowName: source.ShowName, Season: source.Season, Episode: source.Episode, TriggeredAt: source.StartedAt}
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
	run := flows.Execute(r.Context(), flow, event, search)
	// A cancelled HTTP request must not create a misleading historical record.
	if err = r.Context().Err(); err != nil {
		return err
	}
	run, err = s.flowStore().Record(r.Context(), run)
	if err != nil {
		return err
	}
	logged := false
	nodeTypes := make(map[string]string, len(flow.Definition.Nodes))
	for _, node := range flow.Definition.Nodes {
		nodeTypes[node.ID] = node.Type
	}
	for _, step := range run.Steps {
		if nodeTypes[step.NodeID] != "action.log" || step.Status != "successful" || step.Input == nil {
			continue
		}
		value, marshalErr := json.Marshal(step.Input)
		if marshalErr != nil {
			return marshalErr
		}
		if err = activity.Record(r.Context(), s.DB, activity.Event{
			Action:   "flow_test_log",
			Profile:  session.Profile,
			ShowID:   run.Event.ShowID,
			ShowName: run.Event.ShowName,
			Message:  "TEST log from flow \"" + flow.Name + "\": " + string(value),
		}); err != nil {
			return err
		}
		logged = true
	}
	if logged && s.Events != nil {
		s.Events.Publish("", "logs")
	}
	jsonResponse(w, 201, run)
	return nil
}
