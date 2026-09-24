package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/CarlFlo/tally/internal/flows"
	"github.com/CarlFlo/tally/internal/torrent"
)

func TestAdvancedFlowAPIIsAdminOnlyAndReplaysHistoricalTrigger(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	if _, err := s.DB.Exec("INSERT INTO profiles(id,display_name,avatar,created_at) VALUES('member','Member','mint',1)"); err != nil {
		t.Fatal(err)
	}
	member := &http.Cookie{Name: "tally_profile", Value: "member"}
	definition := flows.Definition{Nodes: []flows.Node{{ID: "trigger", Type: "trigger.episode"}, {ID: "query", Type: "query.build"}, {ID: "search", Type: "jackett.search"}}, Edges: []flows.Edge{{ID: "one", Source: "trigger", SourcePort: "event", Target: "query", TargetPort: "event"}, {ID: "two", Source: "query", SourcePort: "query", Target: "search", TargetPort: "query"}}}
	if response := request(t, h, "POST", "/api/flows", flows.Flow{Name: "Test", Definition: definition}, member); response.Code != http.StatusForbidden {
		t.Fatalf("member created flow: %d", response.Code)
	}
	created := request(t, h, "POST", "/api/flows", flows.Flow{Name: "Test", Definition: definition})
	expect(t, created, http.StatusCreated)
	var flow flows.Flow
	if err := json.Unmarshal(created.Body.Bytes(), &flow); err != nil {
		t.Fatal(err)
	}
	if response := request(t, h, "GET", "/api/flows", nil, member); response.Code != http.StatusForbidden {
		t.Fatalf("member listed flows: %d", response.Code)
	}
	if response := request(t, h, "GET", "/api/flows/runs", nil, member); response.Code != http.StatusForbidden {
		t.Fatalf("member listed runs: %d", response.Code)
	}
	if response := request(t, h, "PUT", "/api/flows/"+flow.ID, flow, member); response.Code != http.StatusForbidden {
		t.Fatalf("member edited flow: %d", response.Code)
	}
	if response := request(t, h, "POST", "/api/flows/"+flow.ID+"/replay", map[string]string{"source_run_id": "x"}, member); response.Code != http.StatusForbidden {
		t.Fatalf("member replayed flow: %d", response.Code)
	}
	invalid := flow
	invalid.Definition.Nodes[1].Type = "shell.execute"
	expect(t, request(t, h, "PUT", "/api/flows/"+flow.ID, invalid), http.StatusBadRequest)
	sourceID, err := (torrent.AutomationStore{DB: s.DB}).StartRun(t.Context(), torrent.AutomationRun{ShowID: "show", EpisodeID: "episode", ShowName: "Example Show", Season: 1, Episode: 2, Query: "Example Show S01E02"})
	if err != nil {
		t.Fatal(err)
	}
	if err = (torrent.AutomationStore{DB: s.DB}).FinishRun(t.Context(), sourceID, torrent.RunSkipped, torrent.ReleaseAssessment{}, ""); err != nil {
		t.Fatal(err)
	}
	response := request(t, h, "POST", "/api/flows/"+flow.ID+"/replay", map[string]string{"source_run_id": sourceID})
	expect(t, response, http.StatusCreated)
	var run flows.Run
	if err = json.Unmarshal(response.Body.Bytes(), &run); err != nil {
		t.Fatal(err)
	}
	if run.SourceRunID != sourceID || run.FlowRevision != 1 || run.Event.ShowName != "Example Show" || run.Status != "failed" {
		t.Fatalf("bad replay: %+v", run)
	}
	if strings.Contains(response.Body.String(), "api_key") || strings.Contains(response.Body.String(), "magnet:") {
		t.Fatal("replay leaked transport or credential")
	}
	expect(t, request(t, h, "GET", "/api/flows/runs/"+run.ID, nil), http.StatusOK)
	expect(t, request(t, h, "GET", "/api/flows/runs/"+run.ID, nil, member), http.StatusForbidden)
}
